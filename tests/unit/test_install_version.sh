#!/usr/bin/env bash
# test_install_version.sh - Unit tests for install.sh version detection
#
# Exercises want_version / installed_version (the version-aware upgrade path) by
# overriding VERSION_FILE and BINARY to sandbox paths. Verifies:
#   - want_version reads and strips the VERSION file; empty when absent
#   - installed_version reports a binary's -version; empty when the binary is
#     missing or predates the flag — and, critically, a non-zero exit from an
#     old binary must NOT abort the caller under `set -e`.

set -euo pipefail

TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROJECT_ROOT="$(cd "$TESTS_DIR/.." && pwd)"

# Source the install script (guarded main keeps this side-effect-free).
# shellcheck disable=SC1091
source "$PROJECT_ROOT/scripts/install.sh"

PASS=0
FAIL=0
fail() { printf '  FAIL: %s\n' "$*" >&2; FAIL=$((FAIL + 1)); }
ok() { PASS=$((PASS + 1)); }

# want_version strips whitespace and returns the file contents.
test_want_version_reads_file() {
    local sandbox
    sandbox=$(mktemp -d)
    trap 'rm -rf "$sandbox"' RETURN
    printf '1.2.3\n' > "$sandbox/VERSION"
    VERSION_FILE="$sandbox/VERSION"
    [[ "$(want_version)" == "1.2.3" ]] || { fail "want_version: expected 1.2.3, got '$(want_version)'"; return; }
    ok
}

# No VERSION file → empty (install.sh then falls back to the latest release).
test_want_version_absent_is_empty() {
    VERSION_FILE="/nonexistent/VERSION-xyz"
    [[ -z "$(want_version)" ]] || { fail "want_version: expected empty for missing file"; return; }
    ok
}

# A malformed VERSION (not X.Y.Z: stray 'v', suffix, spaces) is rejected so it
# can't build a bad release URL or drive a re-download loop → treated as unknown.
test_want_version_malformed_is_empty() {
    local sandbox v
    sandbox=$(mktemp -d)
    trap 'rm -rf "$sandbox"' RETURN
    VERSION_FILE="$sandbox/VERSION"
    for v in 'v1.0.2' '1.0.2-rc1' '1.0' '1.0.2 notes' 'garbage'; do
        printf '%s\n' "$v" > "$VERSION_FILE"
        [[ -z "$(want_version 2>/dev/null)" ]] || { fail "want_version: expected empty for malformed '$v'"; return; }
    done
    ok
}

# A current binary reports its version.
test_installed_version_reports() {
    local sandbox
    sandbox=$(mktemp -d)
    trap 'rm -rf "$sandbox"' RETURN
    cat > "$sandbox/fake" <<'EOF'
#!/bin/sh
[ "$1" = "-version" ] && { echo "9.9.9"; exit 0; }
exit 1
EOF
    chmod +x "$sandbox/fake"
    BINARY="$sandbox/fake"
    [[ "$(installed_version)" == "9.9.9" ]] || { fail "installed_version: expected 9.9.9, got '$(installed_version)'"; return; }
    ok
}

# Missing binary → empty.
test_installed_version_missing_is_empty() {
    BINARY="/nonexistent/tmux-yankee-xyz"
    [[ -z "$(installed_version)" ]] || { fail "installed_version: expected empty for missing binary"; return; }
    ok
}

# A binary predating -version exits non-zero on the unknown flag. The helper
# must return empty AND exit 0 so the caller isn't aborted under set -e — this
# is exactly the v1.0.0/v1.0.1 → v1.0.2 upgrade transition.
test_installed_version_old_binary_is_empty() {
    local sandbox out
    sandbox=$(mktemp -d)
    trap 'rm -rf "$sandbox"' RETURN
    cat > "$sandbox/old" <<'EOF'
#!/bin/sh
echo "flag provided but not defined: -version" >&2
exit 2
EOF
    chmod +x "$sandbox/old"
    BINARY="$sandbox/old"
    out="$(installed_version)" || { fail "installed_version: aborted on old binary (exit $?)"; return; }
    [[ -z "$out" ]] || { fail "installed_version: expected empty for old binary, got '$out'"; return; }
    ok
}

echo "test_install_version.sh:"
test_want_version_reads_file
test_want_version_absent_is_empty
test_want_version_malformed_is_empty
test_installed_version_reports
test_installed_version_missing_is_empty
test_installed_version_old_binary_is_empty

echo "  Passed: $PASS"
echo "  Failed: $FAIL"
[[ $FAIL -gt 0 ]] && exit 1
exit 0
