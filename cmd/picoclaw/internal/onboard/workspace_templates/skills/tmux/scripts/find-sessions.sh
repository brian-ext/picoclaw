#!/usr/bin/env bash
set -euo pipefail

SOCKET=""
ALL=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    -S|--socket)
      SOCKET="$2"; shift 2 ;;
    --all)
      ALL=1; shift 1 ;;
    *)
      echo "Unknown arg: $1" >&2
      exit 2
      ;;
  esac
done

if [[ "$ALL" -eq 1 ]]; then
  BASE="${NANOBOT_TMUX_SOCKET_DIR:-${TMPDIR:-/tmp}/nanobot-tmux-sockets}"
  if [[ ! -d "$BASE" ]]; then
    exit 0
  fi
  find "$BASE" -type s -maxdepth 1 -print 2>/dev/null || true
  exit 0
fi

if [[ -z "$SOCKET" ]]; then
  echo "Usage: $0 -S <socket>" >&2
  exit 2
fi

tmux -S "$SOCKET" list-sessions 2>/dev/null || true
