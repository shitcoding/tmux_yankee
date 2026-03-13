# Text Objects: vi", va{, vi[, vi( on JSON config
Output assets/text-objects.gif

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

Type "tmux -L vhs-textobj kill-server 2>/dev/null; true"
Enter
Sleep 500ms

Type "tmux -L vhs-textobj new-session -d -s rec '/opt/homebrew/bin/bat --color=always --plain --paging=never --theme=Dracula --language=json /tmp/yankee-demo-text-objects.txt; exec bash --norc --noprofile'"
Enter
Sleep 1500ms

Type "tmux -L vhs-textobj set -t rec status off"
Enter
Sleep 200ms

Type "tmux -L vhs-textobj set -t rec assume-paste-time 0"
Enter
Sleep 200ms

# Briefly attach+detach so tmux picks up VHS terminal dimensions
Type "tmux -L vhs-textobj attach -t rec \\; detach-client"
Enter
Sleep 500ms

# Launch yankee via send-keys (now has correct VHS dimensions)
Type "tmux -L vhs-textobj send-keys -t rec 'bash /tmp/yankee-launch-text-objects.sh' Enter"
Enter
Sleep 3s

# Attach for real (yankee already running with correct viewport)
Type "tmux -L vhs-textobj attach -t rec"
Enter
Sleep 1500ms

Show

# === Visible demo ===
Sleep 600ms

# Demo 1: vi" — select inside quotes
# Navigate to "payment-gateway" line
Type@250ms ":3"
Sleep 200ms
Enter
Sleep 400ms

# Move to the quoted value
Type "f-"
Sleep 300ms

# Select inside quotes
Type@150ms "vi"
Sleep 100ms
Type '"'
Sleep 1200ms

# Escape back to normal
Escape
Sleep 500ms

# Demo 2: va{ — select around braces
# Go to the tls block
Type@250ms ":10"
Sleep 200ms
Enter
Sleep 400ms

# Select around braces (whole tls object)
Type@150ms "va"
Sleep 100ms
Type "{"
Sleep 1200ms

# Escape back
Escape
Sleep 500ms

# Demo 3: vi[ — select inside square brackets
# Navigate to whitelist line
Type@250ms ":26"
Sleep 200ms
Enter
Sleep 400ms

# Select inside brackets
Type@150ms "vi"
Sleep 100ms
Type "["
Sleep 1200ms

# Escape back
Escape
Sleep 500ms

# Demo 4: vi( — select inside parentheses
# Navigate to maintenance window line
Type@250ms ":31"
Sleep 200ms
Enter
Sleep 400ms

# Select inside parens
Type@150ms "vi"
Sleep 100ms
Type "("
Sleep 1200ms

# Yank the last selection
Type "y"
Sleep 1000ms

# === Hidden cleanup ===
Hide
Type "q"
Sleep 500ms
Ctrl+B
Type "d"
Sleep 500ms
Type "tmux -L vhs-textobj kill-server 2>/dev/null; true"
Enter
Sleep 300ms
