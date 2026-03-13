# Block Selection: column select on process table
Output assets/block-select.gif

Set Shell bash
Set FontSize 16
Set FontFamily "MesloLGS Nerd Font Mono"
Set Width 1200
Set Height 700
Set Theme "Dracula"
Set WindowBar Colorful
Set Padding 15
Set Framerate 24
Set TypingSpeed 80ms
Set CursorBlink false

# === Hidden: create tmux session, launch yankee via send-keys, then attach ===
Hide

Type "tmux -L vhs-block kill-server 2>/dev/null; true"
Enter
Sleep 500ms

Type "tmux -L vhs-block new-session -d -s rec 'cat /tmp/yankee-demo-block-select.txt; exec bash --norc --noprofile'"
Enter
Sleep 1500ms

Type "tmux -L vhs-block set -t rec status off"
Enter
Sleep 200ms

Type "tmux -L vhs-block set -t rec assume-paste-time 0"
Enter
Sleep 200ms

# Briefly attach+detach so tmux picks up VHS terminal dimensions
Type "tmux -L vhs-block attach -t rec \\; detach-client"
Enter
Sleep 500ms

# Launch yankee via send-keys (now has correct VHS dimensions)
Type "tmux -L vhs-block send-keys -t rec 'bash /tmp/yankee-launch-block-select.sh' Enter"
Enter
Sleep 3s

# Attach for real (yankee already running with correct viewport)
Type "tmux -L vhs-block attach -t rec"
Enter
Sleep 1500ms

Show

# === Visible demo ===
Sleep 600ms

# Go to first data row
Type@250ms ":2"
Sleep 200ms
Enter
Sleep 400ms

# Move to the CMD column
Type "fa"
Sleep 300ms

# Enter visual block mode
Ctrl+V
Sleep 600ms

# Extend down through process rows
Type "7j"
Sleep 400ms

# Extend to end of line (cursor past EOL on shorter lines)
Type "$"
Sleep 800ms

# Yank the column
Type "y"
Sleep 2s

# === Hidden cleanup ===
Hide
Type "q"
Sleep 500ms
Ctrl+B
Type "d"
Sleep 500ms
Type "tmux -L vhs-block kill-server 2>/dev/null; true"
Enter
Sleep 300ms
