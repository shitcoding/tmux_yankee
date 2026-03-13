# Flash + Visual Select: extend selection to distant target via flash jump
Output assets/flash-visual.gif

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

# === Hidden: create tmux session, auto-launch yankee via sleep+exec ===
Hide

Type "tmux -L vhs-flashvis kill-server 2>/dev/null; true"
Enter
Sleep 500ms

# bat displays content, then sleep+exec launches yankee (no visible command in scrollback)
Type "tmux -L vhs-flashvis new-session -d -s rec '/opt/homebrew/bin/bat --color=always --plain --paging=never --theme=Dracula --language=go /tmp/yankee-demo-flash.txt; sleep 8; exec bash /tmp/yankee-launch-flash-visual.sh'"
Enter
Sleep 1500ms

Type "tmux -L vhs-flashvis set -t rec status off"
Enter
Sleep 200ms

Type "tmux -L vhs-flashvis set -t rec assume-paste-time 0"
Enter
Sleep 200ms

# Briefly attach+detach so tmux picks up VHS terminal dimensions
Type "tmux -L vhs-flashvis attach -t rec \\; detach-client"
Enter
Sleep 500ms

# Wait for sleep to expire and yankee to auto-launch
Sleep 7s

# Attach for real (yankee already running with correct viewport)
Type "tmux -L vhs-flashvis attach -t rec"
Enter
Sleep 1500ms

Show

# === Visible demo ===
Sleep 500ms

# Navigate to line 42 (inside validateOrder function)
Type@150ms "42G"
Sleep 400ms

# Enter character visual mode
Type "v"
Sleep 400ms

# First flash: "err" — extend selection downward
Type "s"
Sleep 300ms

Type@200ms "err"
Sleep 1000ms

# Press label 'd' to extend selection to line 51
Type "d"
Sleep 1200ms

# Second flash: "ctx" — extend selection further down
Type "s"
Sleep 300ms

Type@200ms "ctx"
Sleep 1000ms

# Press label 'm' to extend selection to line 63
Type "m"
Sleep 1200ms

# Yank the extended selection
Type "y"
Sleep 800ms

# === Hidden cleanup ===
Hide
Type "q"
Sleep 500ms
Ctrl+B
Type "d"
Sleep 500ms
Type "tmux -L vhs-flashvis kill-server 2>/dev/null; true"
Enter
Sleep 300ms
