#!/bin/sh
# install.sh — download and install the iflow CLI.
#
#   curl -fsSL https://raw.githubusercontent.com/aice109/ice-flow-cli/main/install.sh | sh
#
# Environment:
#   VERSION        install a specific version (default: latest release)
#   INSTALL_DIR    install directory (default: $HOME/.local/bin)
set -eu

REPO="aice109/ice-flow-cli"
BIN="iflow"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${VERSION:-latest}"

err() { echo "install: $*" >&2; exit 1; }

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
  darwin | linux) ;;
  *) err "unsupported OS: $os (macOS and Linux only)" ;;
esac

arch=$(uname -m)
case "$arch" in
  x86_64 | amd64) arch=amd64 ;;
  arm64 | aarch64) arch=arm64 ;;
  *) err "unsupported architecture: $arch" ;;
esac

if [ "$VERSION" = "latest" ]; then
  VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
    | sed -n 's/.*"tag_name":[ ]*"\([^"]*\)".*/\1/p' | head -n1)
fi
[ -n "$VERSION" ] || err "could not determine latest version"
VERSION="${VERSION#v}"

archive="${BIN}_${VERSION}_${os}_${arch}.tar.gz"
base="https://github.com/$REPO/releases/download/v${VERSION}"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT INT TERM

echo "Downloading ${archive} (${os}/${arch}) ..."
curl -fsSL "$base/$archive" -o "$tmp/$archive" || err "download failed: $base/$archive"

if curl -fsSL "$base/checksums.txt" -o "$tmp/checksums.txt" 2>/dev/null; then
  if command -v sha256sum >/dev/null 2>&1; then
    (cd "$tmp" && grep " ${archive}\$" checksums.txt | sha256sum -c - >/dev/null) || err "checksum verification failed"
    echo "Checksum verified."
  elif command -v shasum >/dev/null 2>&1; then
    (cd "$tmp" && grep " ${archive}\$" checksums.txt | shasum -a 256 -c - >/dev/null) || err "checksum verification failed"
    echo "Checksum verified."
  fi
fi

tar -xzf "$tmp/$archive" -C "$tmp"
mkdir -p "$INSTALL_DIR"
install -m 0755 "$tmp/$BIN" "$INSTALL_DIR/$BIN"

echo "Installed $BIN $VERSION to $INSTALL_DIR/$BIN"
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *) echo "Add it to your PATH:  export PATH=\"$INSTALL_DIR:\$PATH\"" ;;
esac
echo "Get started:  $BIN init && $BIN --help"
