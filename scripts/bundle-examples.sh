#!/usr/bin/env bash
# Fetches the latest ref-examples tag from GitHub, extracts *.txt + VERSION
# into internal/bundled/examples/, and stages the result for the release build.
set -euo pipefail

REPO="ref-cli/ref-examples"
DEST="internal/bundled/examples"
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

echo "Fetching latest ref-examples tag…"
TAG=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')

if [[ -z "$TAG" ]]; then
  echo "Error: could not determine latest tag" >&2
  exit 1
fi

echo "Downloading ref-examples ${TAG}…"
ZIP_URL="https://github.com/${REPO}/archive/refs/tags/${TAG}.zip"
curl -fsSL "$ZIP_URL" -o "$TMPDIR/examples.zip"

echo "Extracting…"
unzip -q "$TMPDIR/examples.zip" -d "$TMPDIR/extracted"

# The zip contains a single top-level directory: ref-examples-<tag>/
SRC=$(ls "$TMPDIR/extracted")

mkdir -p "$DEST"

# Copy examples/*.txt and VERSION
cp "$TMPDIR/extracted/$SRC"/examples/*.txt "$DEST/" 2>/dev/null || true
cp "$TMPDIR/extracted/$SRC/VERSION" "$DEST/" 2>/dev/null || true

echo "Bundled ref-examples ${TAG} → ${DEST}/"
echo "Staging changes…"
git add "$DEST"
