#!/bin/bash
set -e

cd src

# Get the latest tag from renatokeys/whatsmeow
LATEST_TAG=$(curl -s https://api.github.com/repos/renatokeys/whatsmeow/releases/latest | grep '"tag_name"' | cut -d'"' -f4)

if [ -z "$LATEST_TAG" ]; then
  # If no release, use latest commit
  LATEST_COMMIT=$(curl -s https://api.github.com/repos/renatokeys/whatsmeow/commits/main | grep '"sha"' | head -1 | cut -d'"' -f4 | cut -c1-12)
  WHATSMEOW_VERSION="v0.0.0-$(date +%Y%m%d)-${LATEST_COMMIT}"
else
  WHATSMEOW_VERSION=$LATEST_TAG
fi

# Update the replace directive in go.mod
if grep -q "replace go.mau.fi/whatsmeow =>" go.mod; then
  # Update existing replace
  sed -i "s|replace go.mau.fi/whatsmeow => github.com/[^ ]* [^ ]*|replace go.mau.fi/whatsmeow => github.com/renatokeys/whatsmeow ${WHATSMEOW_VERSION}|" go.mod
else
  # Add new replace
  echo "" >> go.mod
  echo "replace go.mau.fi/whatsmeow => github.com/renatokeys/whatsmeow ${WHATSMEOW_VERSION}" >> go.mod
fi

# Tidy up
go mod tidy || true

echo "go.mod updated to use renatokeys/whatsmeow ${WHATSMEOW_VERSION}"
