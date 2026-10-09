#!/usr/bin/env bash
set -euo pipefail

REPO="CiaranW005/BMC"
INSTALL_PATH="/usr/local/bin/bmc-agent"
RELEASE_URL="https://github.com/$REPO/releases/latest/download"

echo "Checking GitHub for PC Agent updates..."

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

# Download the latest binary and checksum
curl -fsSL \
    --connect-timeout 5 \
    --max-time 30 \
    "$RELEASE_URL/bmc-agent" \
    -o "$TMP_DIR/bmc-agent"

curl -fsSL \
    --connect-timeout 5 \
    --max-time 15 \
    "$RELEASE_URL/bmc-agent.sha256" \
    -o "$TMP_DIR/bmc-agent.sha256"

# Verify integrity
cd "$TMP_DIR"
sha256sum --check bmc-agent.sha256

# Check whether an update is necessary
if [[ -f "$INSTALL_PATH" ]] && cmp -s bmc-agent "$INSTALL_PATH"; then
    echo "PC Agent is already up to date."
    exit 0
fi

echo "Installing updated PC Agent..."

# Stage the new executable on the same filesystem
install -m 755 bmc-agent /usr/local/bin/.bmc-agent.new

# Preserve the previous binary
if [[ -f "$INSTALL_PATH" ]]; then
    cp -p "$INSTALL_PATH" /usr/local/bin/bmc-agent.previous
fi

# Atomically replace the executable
mv -f /usr/local/bin/.bmc-agent.new "$INSTALL_PATH"

echo "PC Agent updated successfully."