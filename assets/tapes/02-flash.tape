# Flash Jump: instant labeled navigation to any visible match
Output assets/flash-jump.gif

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

Type "tmux -L vhs-flash kill-server 2>/dev/null; true"
Enter
Sleep 500ms

# bat displays content, then sleep+exec launches yankee (no visible command in scrollback)
Type "tmux -L vhs-flash new-session -d -s rec '/opt/homebrew/bin/bat --color=always --plain --paging=never --theme=Dracula --language=go /tmp/yankee-demo-flash.txt; sleep 8; exec bash /tmp/yankee-launch-flash.sh'"
Enter
Sleep 1500ms

Type "tmux -L vhs-flash set -t rec status off"
Enter
Sleep 200ms

Type "tmux -L vhs-flash set -t rec assume-paste-time 0"
Enter
Sleep 200ms

# Briefly attach+detach so tmux picks up VHS terminal dimensions
Type "tmux -L vhs-flash attach -t rec \\; detach-client"
Enter
Sleep 500ms

# Wait for sleep to expire and yankee to auto-launch
Sleep 7s

# Attach for real (yankee already running with correct viewport)
Type "tmux -L vhs-flash attach -t rec"
Enter
Sleep 1500ms

Show

# === Visible demo ===
Sleep 500ms

# === First flash: "err" — multiple matches, labeled jump ===
Type "s"
Sleep 300ms

Type@200ms "err"
Sleep 1000ms

# Press label 'c' to jump to line 39 (validateOrder area)
Type "c"
Sleep 1200ms

# === Second flash: "ctx" — multiple matches, labeled jump ===
Type "s"
Sleep 300ms

Type@200ms "ctx"
Sleep 1000ms

# Press label 'm' to jump to line 63 (notify function area)
Type "m"
Sleep 1200ms

Sleep 300ms

# === Hidden cleanup ===
Hide
Type "q"
Sleep 500ms
Ctrl+B
Type "d"
Sleep 500ms
Type "tmux -L vhs-flash kill-server 2>/dev/null; true"
Enter
Sleep 300ms
