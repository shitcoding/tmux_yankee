# Incremental Search: regex search with all-match highlighting
Output assets/search.gif

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

Type "tmux -L vhs-search kill-server 2>/dev/null; true"
Enter
Sleep 500ms

Type "tmux -L vhs-search new-session -d -s rec 'cat /tmp/yankee-demo-search.txt; exec bash --norc --noprofile'"
Enter
Sleep 1500ms

Type "tmux -L vhs-search set -t rec status off"
Enter
Sleep 200ms

Type "tmux -L vhs-search set -t rec assume-paste-time 0"
Enter
Sleep 200ms

# Briefly attach+detach so tmux picks up VHS terminal dimensions
Type "tmux -L vhs-search attach -t rec \\; detach-client"
Enter
Sleep 500ms

# Launch yankee via send-keys (now has correct VHS dimensions)
Type "tmux -L vhs-search send-keys -t rec 'bash /tmp/yankee-launch-search.sh' Enter"
Enter
Sleep 3s

# Attach for real (yankee already running with correct viewport)
Type "tmux -L vhs-search attach -t rec"
Enter
Sleep 1500ms

Show

# === Visible demo ===
Sleep 600ms

# Type search slowly to show incremental highlighting
Type@300ms "/ERROR"
Sleep 1s
Enter
Sleep 500ms

# Jump between matches
Type "n"
Sleep 500ms

Type "n"
Sleep 500ms

Type "n"
Sleep 400ms

# Jump backwards
Type "N"
Sleep 500ms

# Clear and try word-under-cursor search
Escape
Sleep 300ms

Type "/payment"
Sleep 300ms
Enter
Sleep 400ms

# Word-under-cursor search
Type "*"
Sleep 1500ms

# === Hidden cleanup ===
Hide
Type "q"
Sleep 500ms
Ctrl+B
Type "d"
Sleep 500ms
Type "tmux -L vhs-search kill-server 2>/dev/null; true"
Enter
Sleep 300ms
