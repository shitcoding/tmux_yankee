# Line Number Modes: absolute, relative, hybrid toggle via Alt+Shift+L
Output assets/line-number-modes.gif

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

Type "tmux -L vhs-linenums kill-server 2>/dev/null; true"
Enter
Sleep 500ms

Type "tmux -L vhs-linenums new-session -d -s rec 'cat /tmp/yankee-demo-linenums.txt; exec bash --norc --noprofile'"
Enter
Sleep 1500ms

Type "tmux -L vhs-linenums set -t rec status off"
Enter
Sleep 200ms

Type "tmux -L vhs-linenums set -t rec assume-paste-time 0"
Enter
Sleep 200ms

# Force prefix to C-b
Type "tmux -L vhs-linenums set -g prefix C-b"
Enter
Sleep 200ms

# Bind P to send Alt+Shift+L (ESC + L) — toggles line number mode
Type "tmux -L vhs-linenums bind-key P send-keys Escape L"
Enter
Sleep 200ms

# Briefly attach+detach so tmux picks up VHS terminal dimensions
Type "tmux -L vhs-linenums attach -t rec \\; detach-client"
Enter
Sleep 500ms

# Launch yankee via send-keys (now has correct VHS dimensions)
Type "tmux -L vhs-linenums send-keys -t rec 'bash /tmp/yankee-launch-linenums.sh' Enter"
Enter
Sleep 3s

# Attach for real (yankee already running with correct viewport)
Type "tmux -L vhs-linenums attach -t rec"
Enter
Sleep 1500ms

Show

# === Visible demo ===
Sleep 400ms

# Jump to middle of file (starts in hybrid mode — cursor line absolute, rest relative)
Type@250ms ":40"
Sleep 200ms
Enter
Sleep 800ms

# Move cursor to show hybrid numbers (relative offsets visible above/below cursor)
Type "3j"
Sleep 500ms
Type "2k"
Sleep 1000ms

# Toggle to absolute mode (all line numbers are absolute)
Ctrl+B
Type "P"
Sleep 1200ms

# Move to show absolute numbers stay fixed
Type "3j"
Sleep 500ms
Type "2k"
Sleep 1000ms

# Toggle to relative mode (all line numbers are relative to cursor)
Ctrl+B
Type "P"
Sleep 1200ms

# Move to show relative numbers shift with cursor
Type "3j"
Sleep 500ms
Type "2k"
Sleep 1000ms

# Toggle back to hybrid mode (full cycle)
Ctrl+B
Type "P"
Sleep 1200ms

# === Hidden cleanup ===
Hide
Type "q"
Sleep 500ms
Ctrl+B
Type "d"
Sleep 500ms
Type "tmux -L vhs-linenums kill-server 2>/dev/null; true"
Enter
Sleep 300ms
