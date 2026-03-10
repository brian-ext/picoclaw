#!/usr/bin/env bash
set -euo pipefail

TARGET=""
PATTERN=""
TIMEOUT=15
INTERVAL=0.5
LINES=200
FIXED=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    -t|--target) TARGET="$2"; shift 2 ;;
    -p|--pattern) PATTERN="$2"; shift 2 ;;
    -T) TIMEOUT="$2"; shift 2 ;;
    -i) INTERVAL="$2"; shift 2 ;;
    -l) LINES="$2"; shift 2 ;;
    -F) FIXED=1; shift 1 ;;
    *) echo "Unknown arg: $1" >&2; exit 2 ;;
  esac
done

if [[ -z "$TARGET" || -z "$PATTERN" ]]; then
  echo "Usage: $0 -t session:0.0 -p 'pattern' [-F] [-T 20] [-i 0.5] [-l 200]" >&2
  exit 2
fi

start=$(date +%s)
while true; do
  out=$(tmux capture-pane -p -J -t "$TARGET" -S -"$LINES" 2>/dev/null || true)
  if [[ "$FIXED" -eq 1 ]]; then
    if grep -Fq -- "$PATTERN" <<< "$out"; then
      exit 0
    fi
  else
    if grep -Eq -- "$PATTERN" <<< "$out"; then
      exit 0
    fi
  fi

  now=$(date +%s)
  if (( now - start >= TIMEOUT )); then
    exit 1
  fi
  sleep "$INTERVAL"
done
