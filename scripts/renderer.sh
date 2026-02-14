#!/usr/bin/env bash
# renderer.sh - Format captured pane content with line numbers
# This module is a pure function: no tmux calls, no side effects.

# --- Public function ---
render_line_numbers() {
    # Signature: render_line_numbers(
    #     content,          # Newline-delimited pane text
    #     base_absolute,    # First visible line's absolute number
    #     cursor_absolute,  # Cursor line's absolute number
    #     pane_width,       # Pane width in columns
    #     gutter_width,     # Width reserved for gutter (digits + separator)
    #     mode,             # "absolute" | "relative" | "hybrid"
    #     style_absolute,   # ANSI/tmux style for absolute numbers
    #     style_relative,   # ANSI/tmux style for relative numbers
    #     style_cursor      # ANSI/tmux style for cursor line number
    # )
    # Returns: rendered content on stdout
    # Side effects: none

    local content="$1"
    local base_absolute="$2"
    local cursor_absolute="$3"
    local pane_width="$4"
    local gutter_width="$5"
    local mode="$6"
    local style_absolute="$7"
    local style_relative="$8"
    local style_cursor="$9"

    local content_width=$(( pane_width - gutter_width ))
    local row_index=0
    local line_absolute line_relative display_num
    local ansi_reset=$'\033[0m'

    while IFS= read -r line || [[ -n "$line" ]]; do
        line_absolute=$(( base_absolute + row_index ))
        line_relative=$(( line_absolute - cursor_absolute ))
        if [[ $line_relative -lt 0 ]]; then
            line_relative=$(( -line_relative ))
        fi

        # Determine display number and style based on mode
        local style_code=""
        case "$mode" in
            absolute)
                display_num=$line_absolute
                if [[ $line_absolute -eq $cursor_absolute ]]; then
                    style_code=$(tmux_style_to_ansi "$style_cursor")
                else
                    style_code=$(tmux_style_to_ansi "$style_absolute")
                fi
                ;;
            relative)
                if [[ $line_absolute -eq $cursor_absolute ]]; then
                    display_num=$line_absolute
                    style_code=$(tmux_style_to_ansi "$style_cursor")
                else
                    display_num=$line_relative
                    style_code=$(tmux_style_to_ansi "$style_relative")
                fi
                ;;
            hybrid)
                if [[ $line_absolute -eq $cursor_absolute ]]; then
                    display_num=$line_absolute
                    style_code=$(tmux_style_to_ansi "$style_cursor")
                else
                    display_num=$line_relative
                    style_code=$(tmux_style_to_ansi "$style_relative")
                fi
                ;;
        esac

        # Format: right-align number in gutter, then separator, then content
        # Gutter layout: [number][space][separator_char][space]
        # Example with gutter_width=6: "  42 | content here"
        local separator="|"
        local num_field_width=$(( gutter_width - 3 ))  # subtract " | "

        # Truncate content line to fit remaining width
        local truncated_line="${line:0:$content_width}"

        printf '%s%*d %s%s %s\n' \
            "$style_code" \
            "$num_field_width" "$display_num" \
            "$separator" "$ansi_reset" \
            "$truncated_line"

        row_index=$(( row_index + 1 ))
    done <<< "$content"
}

# --- Helper: calculate gutter width ---
calculate_gutter_width() {
    # Signature: calculate_gutter_width(max_line_number)
    # Returns: integer gutter width (stdout)
    # Gutter = digits + " | " (3 chars)
    # Minimum gutter width: 5 (fits up to 99, with " | ")
    # Maximum: dynamic based on history_size

    local max_line="$1"
    local digits=${#max_line}
    if [[ $digits -lt 2 ]]; then
        digits=2
    fi
    printf '%d' $(( digits + 3 ))  # digits + " | "
}

# --- Helper: convert tmux style string to ANSI escape ---
tmux_style_to_ansi() {
    # Signature: tmux_style_to_ansi(tmux_style_string)
    # Input:  "fg=green,bold" or "fg=yellow" or "fg=white"
    # Output: ANSI escape sequence string
    # Returns: ANSI string on stdout

    local style="$1"
    local codes=""

    # Empty style returns empty string
    if [[ -z "$style" ]]; then
        return
    fi

    # Parse bold
    if [[ "$style" == *"bold"* ]]; then
        codes="${codes};1"
    fi

    # Parse fg color
    local fg_color=""
    if [[ "$style" =~ fg=([a-z]+) ]]; then
        fg_color="${BASH_REMATCH[1]}"
    fi

    case "$fg_color" in
        black)   codes="${codes};30" ;;
        red)     codes="${codes};31" ;;
        green)   codes="${codes};32" ;;
        yellow)  codes="${codes};33" ;;
        blue)    codes="${codes};34" ;;
        magenta) codes="${codes};35" ;;
        cyan)    codes="${codes};36" ;;
        white)   codes="${codes};37" ;;
    esac

    # Remove leading semicolon
    codes="${codes#;}"

    if [[ -n "$codes" ]]; then
        printf '\033[%sm' "$codes"
    fi
}
