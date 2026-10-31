#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")/.."
mkdir -p dist
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=false -trimpath -o dist/projetop2p-linux-amd64 ./cmd/projetop2p
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -buildvcs=false -trimpath -o dist/projetop2p-windows-amd64.exe ./cmd/projetop2p
