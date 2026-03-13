# Search + Navigation + Counts: regex search, n/N, :line, motions, gg/G
Output assets/search-navigation.gif

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

Type "tmux -L vhs-searchnav kill-server 2>/dev/null; true"
Enter
Sleep 500ms

Type "tmux -L vhs-searchnav new-session -d -s rec 'cat /tmp/yankee-demo-search-nav.txt; exec bash --norc --noprofile'"
Enter
Sleep 1500ms

Type "tmux -L vhs-searchnav set -t rec status off"
Enter
Sleep 200ms

Type "tmux -L vhs-searchnav set -t rec assume-paste-time 0"
Enter
Sleep 200ms

# Briefly attach+detach so tmux picks up VHS terminal dimensions
Type "tmux -L vhs-searchnav attach -t rec \\; detach-client"
Enter
Sleep 500ms

# Launch yankee via send-keys (now has correct VHS dimensions)
Type "tmux -L vhs-searchnav send-keys -t rec 'bash /tmp/yankee-launch-search-nav.sh' Enter"
Enter
Sleep 3s

# Attach for real (yankee already running with correct viewport)
Type "tmux -L vhs-searchnav attach -t rec"
Enter
Sleep 1500ms

Show

# === Visible demo ===
Sleep 600ms

# Search for "timeout" pattern
Type@250ms "/timeout"
Sleep 500ms
Enter
Sleep 500ms

# Next match
Type "n"
Sleep 500ms

# Previous match
Type "N"
Sleep 500ms

# Go to specific line
Type@250ms ":42"
Sleep 200ms
Enter
Sleep 500ms

# Count motion: 5 lines down
Type "5j"
Sleep 500ms

# Word forward with count
Type "3w"
Sleep 500ms

# Go to top of file
Type "gg"
Sleep 500ms

# Go to bottom of file
Type "G"
Sleep 1000ms

# === Hidden cleanup ===
Hide
Type "q"
Sleep 500ms
Ctrl+B
Type "d"
Sleep 500ms
Type "tmux -L vhs-searchnav kill-server 2>/dev/null; true"
Enter
Sleep 300ms
