#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")/.."
# Linux Mint 22 / Ubuntu 24.04; use a native Linux environment with GTK/WebKit dev packages.
GOFLAGS="${GOFLAGS:-} -buildvcs=false" wails build -trimpath -tags webkit2_41
