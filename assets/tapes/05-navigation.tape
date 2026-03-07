# Navigation: :line, counts, gg/G on Go source code
Output assets/navigation.gif

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

Type "tmux -L vhs-nav kill-server 2>/dev/null; true"
Enter
Sleep 500ms

Type "tmux -L vhs-nav new-session -d -s rec 'cat /tmp/yankee-demo-navigation.txt; exec bash --norc --noprofile'"
Enter
Sleep 1500ms

Type "tmux -L vhs-nav set -t rec status off"
Enter
Sleep 200ms

Type "tmux -L vhs-nav set -t rec assume-paste-time 0"
Enter
Sleep 200ms

# Briefly attach+detach so tmux picks up VHS terminal dimensions
Type "tmux -L vhs-nav attach -t rec \\; detach-client"
Enter
Sleep 500ms

# Launch yankee via send-keys (now has correct VHS dimensions)
Type "tmux -L vhs-nav send-keys -t rec 'bash /tmp/yankee-launch-navigation.sh' Enter"
Enter
Sleep 3s

# Attach for real (yankee already running with correct viewport)
Type "tmux -L vhs-nav attach -t rec"
Enter
Sleep 1500ms

Show

# === Visible demo ===
Sleep 800ms

# Go to a specific line (setupRoutes function)
Type@150ms ":42"
Sleep 200ms
Enter
Sleep 700ms

# Jump down with count
Type "5j"
Sleep 600ms

# Move 3 words forward
Type "3w"
Sleep 600ms

# Go to bottom of file
Type "G"
Sleep 600ms

# Half-page up
Ctrl+U
Sleep 600ms

# Go to top
Type "gg"
Sleep 600ms

# Toggle line number mode (hybrid → absolute)
Type "L"
Sleep 700ms

# Toggle again (absolute → relative)
Type "L"
Sleep 700ms

# Toggle back (relative → hybrid)
Type "L"
Sleep 800ms

# === Hidden cleanup ===
Hide
Type "q"
Sleep 500ms
Ctrl+B
Type "d"
Sleep 500ms
Type "tmux -L vhs-nav kill-server 2>/dev/null; true"
Enter
Sleep 300ms
