#!/usr/bin/env bash
set -euo pipefail

# Capture themed screenshots of real yankee mode on JSON content
# Uses VHS Screenshot (renders Nerd Font/Powerline glyphs correctly, unlike freeze)
# then montage to combine into composite

cd "$(dirname "$0")/../.."
ASSETS_DIR="assets"
THEMES=(default dracula gruvbox nord solarized)
PROJECT_DIR="$(pwd)"

mkdir -p "$ASSETS_DIR"

# Ensure content file and launcher scripts exist
bash assets/tapes/setup.sh text-objects

for theme in "${THEMES[@]}"; do
    echo "Capturing theme: $theme"

    SOCKET="vhs-theme-${theme}"
    TAPE_FILE="/tmp/yankee-theme-${theme}.tape"
    LAUNCH_SCRIPT="/tmp/yankee-theme-launch-${theme}.sh"
    SCREENSHOT_NAME="theme-${theme}.png"

    # Create per-theme launcher script
    cat > "$LAUNCH_SCRIPT" << SCRIPT
#!/usr/bin/env bash
cd '${PROJECT_DIR}'
pane=\$(tmux display-message -p '#{pane_id}')
exec ./bin/tmux-yankee --pane "\$pane" --theme ${theme} --scrollback-lines 500 --exit-on-yank off --start-position top
SCRIPT
    chmod +x "$LAUNCH_SCRIPT"

    # Kill any leftover tmux server from previous runs
    tmux -L "$SOCKET" kill-server 2>/dev/null || true

    # Generate VHS tape with Screenshot (simple filename only — no slashes)
    # Output directive is required for VHS to properly initialize rendering pipeline
    DUMMY_GIF="theme-${theme}-dummy.gif"
    cat > "$TAPE_FILE" << TAPE
Output ${DUMMY_GIF}

Set Shell bash
Set FontSize 16
Set FontFamily "MesloLGS Nerd Font Mono"
Set Width 1200
Set Height 700
Set Theme "Dracula"
Set WindowBar Colorful
Set Padding 15
Set Framerate 1
Set CursorBlink false

Hide

Type "tmux -L ${SOCKET} kill-server 2>/dev/null; true"
Enter
Sleep 500ms

Type "tmux -L ${SOCKET} new-session -d -s rec 'cat /tmp/yankee-demo-text-objects.txt; exec bash --norc --noprofile'"
Enter
Sleep 1500ms

Type "tmux -L ${SOCKET} set -t rec status off"
Enter
Sleep 200ms

Type "tmux -L ${SOCKET} set -t rec assume-paste-time 0"
Enter
Sleep 200ms

Type "tmux -L ${SOCKET} attach -t rec \\\\; detach-client"
Enter
Sleep 500ms

Type "tmux -L ${SOCKET} send-keys -t rec 'bash ${LAUNCH_SCRIPT}' Enter"
Enter
Sleep 3s

Type "tmux -L ${SOCKET} attach -t rec"
Enter
Sleep 1500ms

Show

Sleep 300ms

Type "/gateway"
Sleep 200ms
Enter
Sleep 300ms

Type "V"
Sleep 200ms
Type "2j"
Sleep 500ms

Screenshot ${SCREENSHOT_NAME}
Sleep 1s

Hide
Type "q"
Sleep 500ms
Ctrl+B
Type "d"
Sleep 500ms
Type "tmux -L ${SOCKET} kill-server 2>/dev/null; true"
Enter
Sleep 300ms
TAPE

    # VHS writes Screenshot to CWD, so run from project root
    vhs "$TAPE_FILE"

    # Clean up dummy GIF
    rm -f "$DUMMY_GIF"

    # Move screenshot to assets dir
    if [[ -f "$SCREENSHOT_NAME" ]]; then
        mv "$SCREENSHOT_NAME" "$ASSETS_DIR/$SCREENSHOT_NAME"
        echo "  ✓ $ASSETS_DIR/$SCREENSHOT_NAME"
    else
        echo "  ✗ FAILED: $SCREENSHOT_NAME not created by VHS" >&2
        exit 1
    fi
done

echo ""
echo "Combining into composite..."

# Create labels and combine with montage (3 top + 2 bottom, centered)
montage \
    -label "default" "$ASSETS_DIR/theme-default.png" \
    -label "dracula" "$ASSETS_DIR/theme-dracula.png" \
    -label "gruvbox" "$ASSETS_DIR/theme-gruvbox.png" \
    -label "nord" "$ASSETS_DIR/theme-nord.png" \
    -label "solarized" "$ASSETS_DIR/theme-solarized.png" \
    -tile 3x2 \
    -geometry +10+10 \
    -background "#1e1e2e" \
    -fill "#cdd6f4" \
    -font ".SF-NS-Mono" \
    -pointsize 18 \
    "$ASSETS_DIR/themes-composite.png"

# Resize for README
magick "$ASSETS_DIR/themes-composite.png" -resize 2400x -depth 8 -quality 95 "$ASSETS_DIR/themes-composite.png"

echo "  → $ASSETS_DIR/themes-composite.png"
echo ""
echo "Done!"
ls -lh "$ASSETS_DIR"/theme-*.png "$ASSETS_DIR/themes-composite.png"
