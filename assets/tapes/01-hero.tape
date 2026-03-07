# Hero Workflow: search → select → yank in server logs
Output assets/hero-workflow.gif

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

Type "tmux -L vhs-hero kill-server 2>/dev/null; true"
Enter
Sleep 500ms

Type "tmux -L vhs-hero new-session -d -s rec 'cat /tmp/yankee-demo-hero.txt; exec bash --norc --noprofile'"
Enter
Sleep 1500ms

Type "tmux -L vhs-hero set -t rec status off"
Enter
Sleep 200ms

# Disable paste detection to prevent character reordering
Type "tmux -L vhs-hero set -t rec assume-paste-time 0"
Enter
Sleep 200ms

# Briefly attach+detach so tmux picks up VHS terminal dimensions
Type "tmux -L vhs-hero attach -t rec \\; detach-client"
Enter
Sleep 500ms

# Launch yankee via send-keys (now has correct VHS dimensions)
Type "tmux -L vhs-hero send-keys -t rec 'bash /tmp/yankee-launch-hero.sh' Enter"
Enter
Sleep 3s

# Attach for real (yankee already running with correct viewport)
Type "tmux -L vhs-hero attach -t rec"
Enter
Sleep 1500ms

Show

# === Visible demo ===
Sleep 600ms

# Search for errors (type slowly for incremental effect)
Type@250ms "/timeout|panic"
Sleep 600ms
Enter
Sleep 600ms

# Jump to next match
Type "n"
Sleep 500ms

# Visual line select
Type "V"
Sleep 400ms

# Extend selection to cover the stack trace
Type "5j"
Sleep 800ms

# Yank
Type "y"
Sleep 2s

# === Hidden cleanup ===
Hide
Type "q"
Sleep 500ms
Ctrl+B
Type "d"
Sleep 500ms
Type "tmux -L vhs-hero kill-server 2>/dev/null; true"
Enter
Sleep 300ms
