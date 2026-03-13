# Theme Cycling: cycle through all 5 themes with Alt+t
Output assets/theme-cycling.gif

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

Type "tmux -L vhs-themes kill-server 2>/dev/null; true"
Enter
Sleep 500ms

Type "tmux -L vhs-themes new-session -d -s rec 'cat /tmp/yankee-demo-text-objects.txt; exec bash --norc --noprofile'"
Enter
Sleep 1500ms

Type "tmux -L vhs-themes set -t rec status off"
Enter
Sleep 200ms

Type "tmux -L vhs-themes set -t rec assume-paste-time 0"
Enter
Sleep 200ms

# Force prefix to C-b (user's tmux.conf may set C-a or other prefix)
Type "tmux -L vhs-themes set -g prefix C-b"
Enter
Sleep 200ms

# Bind T to send Alt+t (ESC+t) to the current pane.
# IMPORTANT: use "send-keys Escape t" NOT "send-keys -l $'\x1bt'"
# tmux send-keys -l drops/mishandles raw ESC byte (0x1b), but
# send-keys Escape t sends proper key events that reach yankee correctly.
Type "tmux -L vhs-themes bind-key T send-keys Escape t"
Enter
Sleep 200ms

# Briefly attach+detach so tmux picks up VHS terminal dimensions
Type "tmux -L vhs-themes attach -t rec \\; detach-client"
Enter
Sleep 500ms

# Launch yankee via send-keys (uses text-objects launcher with dracula theme)
Type "tmux -L vhs-themes send-keys -t rec 'bash /tmp/yankee-launch-text-objects.sh' Enter"
Enter
Sleep 3s

# Attach for real (yankee already running with correct viewport)
Type "tmux -L vhs-themes attach -t rec"
Enter
Sleep 1500ms

Show

# === Visible demo ===
Sleep 400ms

# Set up visual state: search + visual select for colorful display
Type@250ms "/gateway"
Sleep 200ms
Enter
Sleep 300ms

# Visual line select to show colored selection across theme changes
Type "V"
Sleep 200ms
Type "2j"
Sleep 600ms

# Cycle through themes with prefix+T (bound to send Alt+t)
# dracula -> gruvbox
Ctrl+B
Type "T"
Sleep 1200ms

# gruvbox -> nord
Ctrl+B
Type "T"
Sleep 1200ms

# nord -> solarized
Ctrl+B
Type "T"
Sleep 1200ms

# solarized -> default
Ctrl+B
Type "T"
Sleep 1200ms

# default -> dracula (full cycle)
Ctrl+B
Type "T"
Sleep 1500ms

# === Hidden cleanup ===
Hide
Type "q"
Sleep 500ms
Ctrl+B
Type "d"
Sleep 500ms
Type "tmux -L vhs-themes kill-server 2>/dev/null; true"
Enter
Sleep 300ms
