#!/bin/sh
set -eu
cd "$(dirname "$0")"
mkdir -p bin
case "${1:-pixctl}" in
  pixctl) package=./cmd/pixctl; output=bin/pixctl ;;
  live) package=./examples/live; output=bin/pix-live ;;
  text) package=./examples/display-text; output=bin/pix-text ;;
  *) echo 'usage: build.sh [pixctl|live|text]' >&2; exit 1 ;;
esac
if [ "$(go env GOOS)" = darwin ]; then
  go build -ldflags "-linkmode=external -extldflags '-Wl,-sectcreate,__TEXT,__info_plist,$PWD/macos/Info.plist'" -o "$output" "$package"
else
  go build -o "$output" "$package"
fi
