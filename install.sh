#!/usr/bin/env bash
set -euo pipefail
REPO="latif-essam/app-dev-clean"
BIN="app-dev-clean"
os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "$arch" in x86_64|amd64) arch=amd64;; arm64|aarch64) arch=arm64;; esac
tag="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep -o '"tag_name": *"[^"]*"' | head -1 | cut -d'"' -f4)"
url="https://github.com/${REPO}/releases/download/${tag}/${BIN}_${os}_${arch}.tar.gz"
tmp="$(mktemp -d)"
echo "Downloading ${url}"
curl -fsSL "$url" | tar -xz -C "$tmp"
dest="${HOME}/.local/bin"
mkdir -p "$dest"
install -m 0755 "$tmp/${BIN}" "$dest/${BIN}"
ln -sf "$dest/${BIN}" "$dest/adc" || true
echo "Installed to ${dest}/${BIN} (alias: adc). Ensure ${dest} is on your PATH."
