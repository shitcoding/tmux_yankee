#!/usr/bin/env bash
# test_install_atomic.sh - Unit tests for install.sh atomic download
#
# Tests install_atomic helper using a fake curl stubbed via PATH. Verifies:
#   - successful download produces the final binary and no leftover temp
#   - failed download before any bytes (non-zero curl exit) leaves no final
#     binary and cleans up its temp file
#   - failed download after a partial write cleans up the temp file and leaves
#     no final binary
#   - an unwritable destination dir leaves any pre-existing binary untouched
#     and cleans up the temp file

set -euo pipefail

TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROJECT_ROOT="$(cd "$TESTS_DIR/.." && pwd)"

# Source the install script (guarded main keeps this side-effect-free).
# shellcheck disable=SC1091
source "$PROJECT_ROOT/scripts/install.sh"

PASS=0
FAIL=0

fail() {
    printf '  FAIL: %s\n' "$*" >&2
    FAIL=$((FAIL + 1))
}

ok() {
    PASS=$((PASS + 1))
}

# Set up a sandbox dir with a fake curl on PATH. The fake reads
# CURL_STUB_MODE and CURL_STUB_PAYLOAD from env.
make_fake_curl() {
    local sandbox="$1"
    local fake_bin="$sandbox/bin"
    mkdir -p "$fake_bin"
    cat > "$fake_bin/curl" <<'EOF'
#!/bin/sh
# Parse out -o <dest> argv pair; everything else is ignored.
dest=""
while [ $# -gt 0 ]; do
    case "$1" in
        -o) dest="$2"; shift 2 ;;
        *)  shift ;;
    esac
done

case "${CURL_STUB_MODE:-success}" in
    success)
        printf '%s' "${CURL_STUB_PAYLOAD:-binary-bytes}" > "$dest"
        exit 0
        ;;
    fail_before_write)
        # Mimic DNS/TLS failure with no bytes written.
        exit 22
        ;;
    fail_after_partial)
        # Write some bytes, then exit non-zero (mid-download abort).
        printf '%s' "PARTIAL" > "$dest"
        exit 18
        ;;
esac
EOF
    chmod +x "$fake_bin/curl"
    PATH="$fake_bin:$PATH"
    export PATH
}

# --- test_success_writes_final_and_no_tmp ---
test_success() {
    local sandbox dest
    sandbox=$(mktemp -d)
    trap 'rm -rf "$sandbox"' RETURN

    make_fake_curl "$sandbox"
    export CURL_STUB_MODE=success
    export CURL_STUB_PAYLOAD="hello-binary"

    dest="$sandbox/tmux-yankee"
    install_atomic "https://example.invalid/dl" "$dest" || { fail "success: install_atomic returned non-zero"; return; }

    [[ -f "$dest" ]] || { fail "success: final binary missing at $dest"; return; }
    if compgen -G "$dest.??????" > /dev/null; then
        fail "success: temp file(s) leftover near $dest"
        return
    fi
    [[ -x "$dest" ]] || { fail "success: final binary not executable"; return; }
    [[ "$(cat "$dest")" == "hello-binary" ]] || { fail "success: payload mismatch"; return; }
    ok
}

# --- test_failed_before_write_leaves_nothing ---
test_failed_before_write() {
    local sandbox dest
    sandbox=$(mktemp -d)
    trap 'rm -rf "$sandbox"' RETURN

    make_fake_curl "$sandbox"
    export CURL_STUB_MODE=fail_before_write
    export CURL_STUB_PAYLOAD=""

    dest="$sandbox/tmux-yankee"
    if install_atomic "https://example.invalid/dl" "$dest"; then
        fail "fail_before_write: install_atomic unexpectedly returned 0"
        return
    fi

    [[ ! -e "$dest" ]] || { fail "fail_before_write: final binary unexpectedly exists"; return; }
    if compgen -G "$dest.??????" > /dev/null; then
        fail "fail_before_write: temp file(s) leftover near $dest"
        return
    fi
    ok
}

# --- test_failed_after_partial_cleans_tmp_keeps_no_dest ---
test_failed_after_partial() {
    local sandbox dest
    sandbox=$(mktemp -d)
    trap 'rm -rf "$sandbox"' RETURN

    make_fake_curl "$sandbox"
    export CURL_STUB_MODE=fail_after_partial

    dest="$sandbox/tmux-yankee"
    if install_atomic "https://example.invalid/dl" "$dest"; then
        fail "fail_after_partial: install_atomic unexpectedly returned 0"
        return
    fi

    [[ ! -e "$dest" ]] || { fail "fail_after_partial: final binary unexpectedly exists at $dest"; return; }
    if compgen -G "$dest.??????" > /dev/null; then
        fail "fail_after_partial: temp file(s) leftover near $dest"
        return
    fi
    ok
}

# --- test_readonly_dest_dir_leaves_dest_untouched ---
# When the destination directory cannot be written, install_atomic must
# fail cleanly and leave any pre-existing $dest untouched. (mktemp is the
# step that fails here, but the failure mode is the one a user would
# encounter most often: insufficient permissions in the install dir.)
test_readonly_dest_dir() {
    local sandbox dest_dir dest
    sandbox=$(mktemp -d)
    trap 'chmod u+w "$dest_dir" 2>/dev/null; rm -rf "$sandbox"' RETURN

    make_fake_curl "$sandbox"
    export CURL_STUB_MODE=success
    export CURL_STUB_PAYLOAD="payload"

    dest_dir="$sandbox/locked"
    mkdir -p "$dest_dir"
    # Pre-existing $dest with sentinel content we expect NOT to be touched.
    printf 'PRE-EXISTING' > "$dest_dir/tmux-yankee"
    chmod a-w "$dest_dir"

    dest="$dest_dir/tmux-yankee"

    # 2>/dev/null suppresses the expected "mkstemp failed" noise from mktemp.
    if install_atomic "https://example.invalid/dl" "$dest" 2>/dev/null; then
        chmod u+w "$dest_dir"
        fail "readonly_dest: install_atomic unexpectedly returned 0"
        return
    fi

    chmod u+w "$dest_dir"
    if compgen -G "$dest.??????" > /dev/null; then
        fail "readonly_dest: temp file(s) leftover near $dest"
        return
    fi
    [[ -f "$dest" ]] || { fail "readonly_dest: pre-existing dest was destroyed"; return; }
    [[ "$(cat "$dest")" == "PRE-EXISTING" ]] || { fail "readonly_dest: pre-existing dest was modified"; return; }
    ok
}

# --- run all ---
echo "test_install_atomic.sh:"
test_success
test_failed_before_write
test_failed_after_partial
test_readonly_dest_dir

echo "  Passed: $PASS"
echo "  Failed: $FAIL"

if [[ $FAIL -gt 0 ]]; then
    exit 1
fi
exit 0
