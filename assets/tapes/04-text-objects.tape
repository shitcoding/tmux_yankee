# Text Objects: vi" and va{ on JSON config
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

Type "tmux -L vhs-textobj new-session -d -s rec 'cat /tmp/yankee-demo-text-objects.txt; exec bash --norc --noprofile'"
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

# Search for a string value
Type@200ms "/payment-gateway"
Sleep 400ms
Enter
Sleep 500ms

# Select inside quotes (vi")
Type@150ms "vi"
Sleep 100ms
Type '"'
Sleep 1500ms

# Escape back to normal
Escape
Sleep 400ms

# Now demo va{ — select around the "tls" block
Type@200ms "/tls"
Sleep 400ms
Enter
Sleep 500ms

# Select around braces
Type@150ms "va{"
Sleep 1500ms

# Escape
Escape
Sleep 400ms

# One more: select inner paragraph (vip) on a config section
Type "}"
Sleep 300ms
Type@150ms "vip"
Sleep 1500ms

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
