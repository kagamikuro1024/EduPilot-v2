#!/usr/bin/env bash
# Dựng lại video giới thiệu trên VPS: quay bản demo (container Playwright) rồi mã hoá (container debian + ffmpeg). Chạy ở gốc repo trên VPS.
#   bash scripts/landing-video/run.sh                 # cần /opt/edupilot/.env.local (EP_HOST, SEED_DEFAULT_PASSWORD); node_modules playwright ở $PWMOD
set -euo pipefail
ROOT=$(cd "$(dirname "$0")/../.." && pwd); OUT=${OUT:-/srv/work/dev/landing-out}; PWMOD=${PWMOD:-/srv/work/dev/pw}
PW=$(grep '^SEED_DEFAULT_PASSWORD=' /opt/edupilot/.env.local | cut -d= -f2-); EPH=$(grep '^EP_HOST=' /opt/edupilot/.env.local | cut -d= -f2-)
[ -d "$PWMOD/node_modules/playwright" ] || docker run --rm -v "$PWMOD":/w -w /w mcr.microsoft.com/playwright:v1.63.0-noble sh -c 'npm init -y >/dev/null && npm i playwright@1.63.0 >/dev/null 2>&1'
mkdir -p "$OUT" "$ROOT/frontend/public/landing"
docker run --rm --ipc=host -e PW="$PW" -e EP_HOST="$EPH" -e OUT=/out -v "$OUT":/out -v "$PWMOD/node_modules":/w/node_modules \
  -v "$ROOT/scripts/landing-video/record.mjs":/w/record.mjs -w /w mcr.microsoft.com/playwright:v1.63.0-noble node record.mjs
docker run --rm -v "$OUT":/out -v "$ROOT/frontend/public/landing":/dest -v "$ROOT/scripts/landing-video/encode.sh":/out/encode.sh:ro debian:trixie-slim bash /out/encode.sh
