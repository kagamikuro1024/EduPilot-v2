#!/usr/bin/env bash
# Dựng lại video, clip và ảnh của trang giới thiệu: quay một bản chạy EduPilot (container Playwright) rồi mã hoá (container debian + ffmpeg). Hướng dẫn: README.md.
set -euo pipefail
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
OUT=${OUT:-$ROOT/.landing-out}                   # khung hình trung gian
PWMOD=${PWMOD:-$ROOT/.landing-pw}                # nơi cài playwright@1.63.0 (tự cài lần đầu)
ENV_FILE=${ENV_FILE:-}                           # tệp .env (không bắt buộc); đọc EP_HOST, SEED_DEFAULT_PASSWORD nếu biến chưa đặt
if [ -n "$ENV_FILE" ]; then
  : "${EP_HOST:=$(grep '^EP_HOST=' "$ENV_FILE" | cut -d= -f2- || true)}"
  : "${SEED_DEFAULT_PASSWORD:=$(grep '^SEED_DEFAULT_PASSWORD=' "$ENV_FILE" | cut -d= -f2- || true)}"
fi
[ -n "${EP_BASE_URL:-}${EP_HOST:-}" ] || { echo "Thiếu EP_HOST hoặc EP_BASE_URL (xem README.md)" >&2; exit 1; }
[ -n "${SEED_DEFAULT_PASSWORD:-}" ] || { echo "Thiếu SEED_DEFAULT_PASSWORD (biến môi trường hoặc ENV_FILE)" >&2; exit 1; }
NET=(); [ -n "${EP_BASE_URL:-}" ] && NET=(--network host) # địa chỉ cục bộ (cổng ép bind 127.0.0.1) cần mạng của máy chủ
[ -d "$PWMOD/node_modules/playwright" ] || { mkdir -p "$PWMOD"; docker run --rm -v "$PWMOD":/w -w /w mcr.microsoft.com/playwright:v1.63.0-noble sh -c 'npm init -y >/dev/null && npm i playwright@1.63.0 >/dev/null 2>&1'; }
mkdir -p "$OUT" "$ROOT/frontend/public/landing"
docker run --rm --ipc=host "${NET[@]}" -e PW="$SEED_DEFAULT_PASSWORD" -e EP_HOST="${EP_HOST:-}" -e EP_BASE_URL="${EP_BASE_URL:-}" -e EP_CLASS_CODE="${EP_CLASS_CODE:-761987}" \
  -e EP_STUDENT_EMAIL="${EP_STUDENT_EMAIL:-sv.kha@edupilot.local}" -e EP_TEACHER_EMAIL="${EP_TEACHER_EMAIL:-teacher@edupilot.local}" -e OUT=/out -e STILLS_ONLY="${STILLS_ONLY:-}" \
  -v "$OUT":/out -v "$PWMOD/node_modules":/w/node_modules -v "$ROOT/scripts/landing-video/record.mjs":/w/record.mjs -w /w mcr.microsoft.com/playwright:v1.63.0-noble node record.mjs
docker run --rm -v "$OUT":/out -v "$ROOT/frontend/public/landing":/dest -v "$ROOT/scripts/landing-video/encode.sh":/out/encode.sh:ro -e STILLS_ONLY="${STILLS_ONLY:-}" debian:trixie-slim bash /out/encode.sh
