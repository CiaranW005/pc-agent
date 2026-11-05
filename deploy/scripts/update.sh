#!/usr/bin/env bash
set -euo pipefail

# Agent binaries are published as releases in the parent BMC repository.
# Requires curl, jq, sha256sum, and a token with access to its release assets.
REPO="CiaranW005/BMC"
TOKEN_FILE="/etc/bmc-agent/github-token"
INSTALL_PATH="/usr/local/bin/bmc-agent"

API="https://api.github.com/repos/$REPO"

echo "Checking GitHub for PC Agent updates..."

TOKEN=$(tr -d '\r\n' < "$TOKEN_FILE")

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

# Retrieve latest release metadata.
curl -fsSL \
    --connect-timeout 5 \
    --max-time 15 \
    -H "Authorization: Bearer $TOKEN" \
    -H "Accept: application/vnd.github+json" \
    "$API/releases/latest" \
    -o "$TMP_DIR/release.json"

# Find the asset IDs.
BINARY_ID=$(jq -r '.assets[] | select(.name == "bmc-agent") | .id' "$TMP_DIR/release.json")
CHECKSUM_ID=$(jq -r '.assets[] | select(.name == "bmc-agent.sha256") | .id' "$TMP_DIR/release.json")

if [[ -z "$BINARY_ID" || -z "$CHECKSUM_ID" ]]; then
    echo "Release is missing required assets."
    exit 1
fi

# Download an authenticated release asset.
download_asset() {
    local asset_id="$1"
    local destination="$2"

    curl -fsSL \
        --connect-timeout 5 \
        --max-time 30 \
        -H "Authorization: Bearer $TOKEN" \
        -H "Accept: application/octet-stream" \
        -H "X-GitHub-Api-Version: 2022-11-28" \
        "$API/releases/assets/$asset_id" \
        -o "$destination"
}

download_asset "$BINARY_ID" "$TMP_DIR/bmc-agent"
download_asset "$CHECKSUM_ID" "$TMP_DIR/bmc-agent.sha256"

# The checksum file names bmc-agent relative to the current directory.
cd "$TMP_DIR"
sha256sum --check bmc-agent.sha256

# Check whether an update is necessary.
if [[ -f "$INSTALL_PATH" ]] && cmp -s bmc-agent "$INSTALL_PATH"; then
    echo "PC Agent is already up to date."
    exit 0
fi

echo "Installing updated PC Agent..."

# Stage and install the executable.
install -m 755 bmc-agent /usr/local/bin/.bmc-agent.new

if [[ -f "$INSTALL_PATH" ]]; then
    cp -p "$INSTALL_PATH" /usr/local/bin/bmc-agent.previous
fi

# Replace the executable by renaming within the same filesystem.
mv -f /usr/local/bin/.bmc-agent.new "$INSTALL_PATH"

echo "PC Agent updated successfully."