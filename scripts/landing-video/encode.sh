#!/bin/bash
# Mã hoá video giới thiệu (TL-1 A): khung JPEG + frames.ffconcat → mp4 H.264, webm VP9, poster WebP. Chạy TRONG container (run.sh):
#   docker run --rm -v $OUT:/out -v $DEST:/dest debian:trixie-slim bash /out/encode.sh
set -euo pipefail
apt-get update -qq >/dev/null && apt-get install -y -qq ffmpeg jq >/dev/null 2>&1
cd /out
N=gioi-thieu-v1
ffmpeg -hide_banner -loglevel error -y -f concat -safe 0 -i frames.ffconcat -fps_mode cfr -r 30 \
  -c:v libx264 -preset slow -tune stillimage -crf 28 -maxrate 700k -bufsize 1400k -pix_fmt yuv420p -movflags +faststart -an /dest/$N.mp4
ffmpeg -hide_banner -loglevel error -y -i /dest/$N.mp4 -c:v libvpx-vp9 -b:v 0 -crf 46 -row-mt 1 -deadline good -cpu-used 2 -an /dest/$N.webm
ffmpeg -hide_banner -loglevel error -y -ss "$(jq -r .poster meta.json)" -i /dest/$N.mp4 -frames:v 1 -vf scale=960:-2 -c:v libwebp -quality 68 /dest/$N.webp
ls -l /dest/$N.*
ffprobe -v error -show_entries format=duration,size,bit_rate -of compact=p=0 /dest/$N.mp4
