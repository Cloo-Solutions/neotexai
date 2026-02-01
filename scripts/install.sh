#!/bin/sh
set -e

REPO="cloo-solutions/neotexai"
BINARY="neotex"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case $ARCH in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# darwin/amd64 excluded per .goreleaser.yml
if [ "$OS" = "darwin" ] && [ "$ARCH" = "amd64" ]; then
  echo "Error: macOS Intel (amd64) not supported. Use Apple Silicon or Linux."
  exit 1
fi

# Get latest version
VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/')

# Download and install
ARCHIVE="neotexai_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/v${VERSION}/${ARCHIVE}"

INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

curl -fsSL "$URL" -o "$TMP_DIR/$ARCHIVE"
tar -xzf "$TMP_DIR/$ARCHIVE" -C "$TMP_DIR"

if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP_DIR/$BINARY" "$INSTALL_DIR/"
else
  sudo mv "$TMP_DIR/$BINARY" "$INSTALL_DIR/"
fi

echo "Installed $BINARY v$VERSION to $INSTALL_DIR"
echo "Run 'neotex --help' to get started"
