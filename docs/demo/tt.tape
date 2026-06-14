# Demo tape for tt. Rendered with ascii-gif (github.com/tamnd/ascii-gif),
# which supplies the window chrome and theme; this file is just the action.
#
#   ascii-gif render docs/demo/tt.tape -o docs/static/demo.gif
#
# tt must be on PATH inside the recording shell.

Hide
Type "export PS1='$ ' PATH=/tmp:$PATH"
Enter
Type "clear"
Enter
Show

Sleep 500ms
Type "tt video 7106594312292453675 --fields author,play_count,digg_count,comment_count,duration -o table"
Sleep 800ms
Enter
Sleep 3.5s

Type "tt video 7106594312292453675 -o jsonl | jq '{author, plays: .play_count, likes: .digg_count}'"
Sleep 800ms
Enter
Sleep 4s
