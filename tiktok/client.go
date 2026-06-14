// Package tiktok is the library behind the tt command line: the HTTP client,
// the SSR blob parsing, the signed API calls, and the typed data models.
//
// Two planes feed the records. The SSR plane reads the
// __UNIVERSAL_DATA_FOR_REHYDRATION__ JSON a logged-out page ships and needs no
// signing. The API plane calls www.tiktok.com/api/* with an X-Bogus signature
// and an msToken. The API plane sits behind a Web Application Firewall that
// scores the caller; when it wins, the client returns ErrWalled so the command
// layer can report it honestly instead of a silent empty result.
package tiktok

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strings"
	"time"

	"github.com/tamnd/tiktok-cli/pkg/tthtml"
	"github.com/tamnd/tiktok-cli/pkg/ttsign"
)

// Host is the web origin every request targets.
const Host = "https://www.tiktok.com"

// DefaultUserAgent is a current desktop Chrome string. An honest, real
// User-Agent is both polite and the thing most likely to keep a session
// unblocked.
const DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

// ErrWalled means the WAF served a challenge or a signed call came back empty.
// The surface needs a residential session.
var ErrWalled = errors.New("tiktok served a WAF challenge (this surface needs a residential session)")

// ErrNotFound means the page parsed but carried no record for the request, for
// example a handle that does not exist or a private profile with no items.
var ErrNotFound = errors.New("not found")

// Config holds the tunable client settings.
type Config struct {
	UserAgent string
	Rate      time.Duration
	Timeout   time.Duration
	Retries   int
}

// DefaultConfig returns polite defaults: a 600ms gap between requests, a 30s
// timeout, and five retries on transient errors.
func DefaultConfig() Config {
	return Config{
		UserAgent: DefaultUserAgent,
		Rate:      600 * time.Millisecond,
		Timeout:   30 * time.Second,
		Retries:   5,
	}
}

// Client talks to TikTok over HTTP.
type Client struct {
	cfg    Config
	http   *http.Client
	signer *ttsign.Signer
	last   time.Time
}

// NewClient builds a Client from cfg.
func NewClient(cfg Config) *Client {
	if cfg.UserAgent == "" {
		cfg.UserAgent = DefaultUserAgent
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
		signer: &ttsign.Signer{
			NowMillis: func() int64 { return time.Now().UnixMilli() },
			Rand: func(n int) []byte {
				b := make([]byte, n)
				_, _ = rand.Read(b)
				return b
			},
		},
	}
}

// UserAgent returns the configured User-Agent.
func (c *Client) UserAgent() string { return c.cfg.UserAgent }

// GetPage fetches an SSR page and returns its body. It detects the WAF
// challenge stub and returns ErrWalled for it.
func (c *Client) GetPage(ctx context.Context, url string) (string, error) {
	body, err := c.get(ctx, url, map[string]string{
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"Accept-Language": "en-US,en;q=0.9",
		"Referer":         Host + "/",
	})
	if err != nil {
		return "", err
	}
	html := string(body)
	if tthtml.IsWAFChallenge(html) {
		return "", ErrWalled
	}
	return html, nil
}

// GetAPI signs an /api/{path} call, fetches it, and returns the JSON body. An
// empty body on a 200 is the WAF gating the call, so it maps to ErrWalled.
func (c *Client) GetAPI(ctx context.Context, path, rawQuery string) ([]byte, error) {
	signed := c.signer.Sign(rawQuery, c.cfg.UserAgent)
	url := Host + path + "?" + signed.Query
	headers := map[string]string{
		"Accept":          "application/json, text/plain, */*",
		"Accept-Language": "en-US,en;q=0.9",
		"Referer":         Host + "/",
		"Origin":          Host,
	}
	maps.Copy(headers, signed.Headers)
	body, err := c.get(ctx, url, headers)
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return nil, ErrWalled
	}
	return body, nil
}

func (c *Client) get(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, url, headers)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", url, lastErr)
}

func (c *Client) do(ctx context.Context, url string, headers map[string]string) (body []byte, retry bool, err error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

// pace blocks until at least Rate has passed since the previous request.
func (c *Client) pace() {
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	return min(time.Duration(attempt)*500*time.Millisecond, 5*time.Second)
}
