#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")/.."
: "${P2P_PUBLIC_ADDR:?Defina P2P_PUBLIC_ADDR com /ip4/IP_PUBLICO_REAL/tcp/4001 ou /dns4/DNS_REAL/tcp/4001 (sem /p2p/ID)}"
exec "${P2P_BINARY:-./dist/projetop2p-linux-amd64}" \
  --data "${P2P_DATA:-./data/relay}" \
  --listen "${P2P_LISTEN:-/ip4/0.0.0.0/tcp/4001}" \
  --announce "$P2P_PUBLIC_ADDR" \
  --relay-server --public --daemon --status-every 30s "$@"
