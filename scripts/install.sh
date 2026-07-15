#!/usr/bin/env bash
set -euo pipefail

PLUGIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN_DIR="${PLUGIN_DIR}/bin"
BINARY="${BIN_DIR}/tmux-yankee"
VERSION_FILE="${PLUGIN_DIR}/VERSION"

# want_version prints the version this checkout expects (from the VERSION file),
# or nothing if the file is absent/unreadable/malformed. Whitespace is stripped.
# The value is interpolated into the release URL (releases/tags/v<want>), so a
# malformed VERSION (a stray 'v', spaces, a query string) would hit the wrong
# endpoint and make every load retry the download. Reject anything that is not a
# plain X.Y.Z and behave as "version unknown" (loud on stderr).
# ponytail: strict X.Y.Z only; extend the regex if pre-release tags are ever used.
want_version() {
    [[ -f "$VERSION_FILE" ]] || return 0
    local v
    v="$(tr -d '[:space:]' < "$VERSION_FILE" 2>/dev/null)" || return 0
    if [[ "$v" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        printf '%s' "$v"
    elif [[ -n "$v" ]]; then
        echo "tmux-yankee: ignoring malformed VERSION ('${v}')" >&2
    fi
}

# installed_version prints the version the installed binary reports via
# `-version`, or nothing if the binary is missing or predates that flag. Always
# returns 0 so callers can use it under `set -e` without aborting.
installed_version() {
    [[ -x "$BINARY" ]] || return 0
    "$BINARY" -version 2>/dev/null | tr -d '[:space:]' || true
}

detect_platform() {
    local os arch

    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    case "$os" in
        darwin) os="darwin" ;;
        linux)  os="linux" ;;
        *)
            echo "tmux-yankee: unsupported OS: $os" >&2
            return 1
            ;;
    esac

    arch="$(uname -m)"
    case "$arch" in
        x86_64)       arch="amd64" ;;
        arm64|aarch64) arch="arm64" ;;
        *)
            echo "tmux-yankee: unsupported architecture: $arch" >&2
            return 1
            ;;
    esac

    echo "${os}-${arch}"
}

resolve_repo() {
    local remote_url path repo owner
    if remote_url="$(git -C "$PLUGIN_DIR" remote get-url origin 2>/dev/null)"; then
        # Strip .git suffix
        path="${remote_url%.git}"
        # Handle SSH (git@host:owner/repo) — take part after ':'
        if [[ "$path" == *:* && "$path" != *://* ]]; then
            path="${path##*:}"
        fi
        # Extract last two path components: owner/repo
        repo="${path##*/}"
        path="${path%/*}"
        owner="${path##*/}"
        echo "${owner}/${repo}"
        return
    fi
    echo "shitcoding/tmux_yankee"
}

download_binary() {
    local platform repo api_url download_url asset_name want

    platform="$(detect_platform)" || return 1
    repo="$(resolve_repo)"
    asset_name="tmux-yankee-${platform}"

    # Pin the download to the checkout's version (releases/tags/v<VERSION>) so
    # the binary always matches the source. Fall back to the latest release only
    # when the checkout has no VERSION file.
    want="$(want_version)"
    if [[ -n "$want" ]]; then
        api_url="https://api.github.com/repos/${repo}/releases/tags/v${want}"
        echo "tmux-yankee: fetching v${want} for ${platform} from ${repo}..."
    else
        api_url="https://api.github.com/repos/${repo}/releases/latest"
        echo "tmux-yankee: fetching latest release for ${platform} from ${repo}..."
    fi

    # Query GitHub API for the download URL of the matching asset.
    # Timeouts matter now that yankee.tmux runs this on every load: a black-holed
    # network must not hang tmux config loading indefinitely on the upgrade path.
    local api_response
    if ! api_response="$(curl -fsSL --connect-timeout 10 --max-time 30 "$api_url" 2>&1)"; then
        echo "tmux-yankee: GitHub API request failed: ${api_response}" >&2
        return 1
    fi
    download_url="$(printf '%s' "$api_response" \
        | grep -o "\"browser_download_url\": *\"[^\"]*${asset_name}\"" \
        | head -1 \
        | sed 's/.*"browser_download_url": *"//' \
        | sed 's/"$//')" || true

    if [[ -z "$download_url" ]]; then
        echo "tmux-yankee: no release asset found for ${asset_name}" >&2
        return 1
    fi

    echo "tmux-yankee: downloading ${asset_name}..."
    mkdir -p "$BIN_DIR"
    if install_atomic "$download_url" "$BINARY"; then
        echo "tmux-yankee: installed to ${BINARY}"
        return 0
    else
        echo "tmux-yankee: download failed" >&2
        return 1
    fi
}

# install_atomic downloads $url to a unique temp file in the same directory
# as $dest, marks it executable, then renames it onto $dest. A failure in
# any step (curl exit, chmod, mv) removes the temp file and reports failure
# without leaving partial state at $dest. The temp file uses mktemp so
# concurrent installs cannot share a predictable path.
install_atomic() {
    local url="$1"
    local dest="$2"
    local tmpfile
    tmpfile=$(mktemp "${dest}.XXXXXX") || return 1
    # Chain so any step's failure short-circuits to the cleanup path. Inside
    # `if`, bash disables errexit, so explicit chaining is the only reliable
    # way to catch a failed mv after a successful curl.
    if curl -fsSL --connect-timeout 10 --max-time 300 -o "$tmpfile" "$url" \
        && chmod +x "$tmpfile" \
        && mv "$tmpfile" "$dest"; then
        return 0
    fi
    rm -f "$tmpfile"
    return 1
}

build_from_source() {
    if ! command -v go >/dev/null 2>&1; then
        return 1
    fi
    echo "tmux-yankee: building from source..."
    make -C "$PLUGIN_DIR" build
}

main() {
    local want have
    want="$(want_version)"
    have="$(installed_version)"

    # Up to date when the binary exists and either the wanted version is unknown
    # (no VERSION file) or the installed version already matches. Silent on this
    # path — yankee.tmux runs install.sh on every load, so a stale binary is
    # picked up automatically after a plugin update (git pull bumps VERSION).
    # Also treat an unstamped local dev build ("dev") as up-to-date so a plain
    # `go build` (no make) is never clobbered by the release binary.
    if [[ -x "$BINARY" ]] \
        && { [[ -z "$want" ]] || [[ "$have" == "$want" ]] || [[ "$have" == "dev" ]]; }; then
        exit 0
    fi

    if [[ -x "$BINARY" && -n "$want" ]]; then
        echo "tmux-yankee: updating ${have:-unknown} -> ${want}"
    fi

    if download_binary; then
        # Surface a stamp mismatch (e.g. a release whose VERSION != tag) rather
        # than silently re-downloading the same binary on every future load.
        local now
        now="$(installed_version)"
        if [[ -n "$want" && -n "$now" && "$now" != "$want" ]]; then
            echo "tmux-yankee: warning: installed binary reports ${now}, expected ${want}" >&2
        fi
        exit 0
    fi

    echo "tmux-yankee: download failed, trying to build from source..." >&2
    if build_from_source; then
        exit 0
    fi

    # Best-effort upgrade: if a (stale) binary is already present, keep it working
    # instead of failing on every load — e.g. during the brief window between a
    # VERSION bump and its release being published.
    if [[ -x "$BINARY" ]]; then
        echo "tmux-yankee: WARNING: could not fetch ${want:-latest}; keeping existing binary (${have:-unknown})." >&2
        exit 0
    fi

    echo "tmux-yankee: ERROR: could not install binary." >&2
    echo "  Either create a release at https://github.com/$(resolve_repo)/releases" >&2
    echo "  or install Go and run 'make build' in ${PLUGIN_DIR}" >&2
    exit 1
}

# Only execute main when run as a script. Sourcing this file (e.g. from
# tests) exposes the helpers without invoking main.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
