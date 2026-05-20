#!/usr/bin/env bash
set -euo pipefail

PLUGIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN_DIR="${PLUGIN_DIR}/bin"
BINARY="${BIN_DIR}/tmux-yankee"

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
    local platform repo api_url download_url asset_name

    platform="$(detect_platform)" || return 1
    repo="$(resolve_repo)"
    asset_name="tmux-yankee-${platform}"
    api_url="https://api.github.com/repos/${repo}/releases/latest"

    echo "tmux-yankee: detecting platform... ${platform}"
    echo "tmux-yankee: fetching latest release from ${repo}..."

    # Query GitHub API for the download URL of the matching asset
    local api_response
    if ! api_response="$(curl -fsSL "$api_url" 2>&1)"; then
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
    if curl -fsSL -o "$tmpfile" "$url" \
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
    if [[ -x "$BINARY" ]]; then
        echo "tmux-yankee: binary already exists at ${BINARY}"
        exit 0
    fi

    if download_binary; then
        exit 0
    fi

    echo "tmux-yankee: download failed, trying to build from source..." >&2
    if build_from_source; then
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
