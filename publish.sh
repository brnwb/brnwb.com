#!/usr/bin/env bash
set -euo pipefail

rm -rf html
pnpm run build

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
OUT_DIR="$ROOT_DIR/html"
REMOTE_HOST="brnwb"
REMOTE_DIR="/var/www/brnwb.com"

echo "Publishing $OUT_DIR -> ${REMOTE_HOST}:${REMOTE_DIR}"
rsync -avz --delete "$OUT_DIR/" "${REMOTE_HOST}:${REMOTE_DIR}/"

echo "Publish complete."
