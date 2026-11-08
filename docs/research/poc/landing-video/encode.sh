#!/usr/bin/env bash
# PoC TL-1 sprint 9.1: mã hoá khung JPEG (page.screencast onFrame) và webm VP8 của Playwright, in kích thước / bitrate.
# Chạy trên VPS: docker run --rm -v $PWD:/w -w /w debian:trixie-slim bash encode.sh
set -euo pipefail
apt-get update -qq >/dev/null && apt-get install -y -qq ffmpeg >/dev/null 2>&1
ffmpeg -hide_banner -version | head -1
X264="-c:v libx264 -preset slow -tune stillimage -pix_fmt yuv420p -movflags +faststart -an"
for crf in 26 28 30; do
  ffmpeg -hide_banner -loglevel error -y -f concat -safe 0 -i frames.ffconcat -fps_mode cfr -r 30 $X264 -crf $crf "x264-crf$crf.mp4"
done
ffmpeg -hide_banner -loglevel error -y -i x264-crf26.mp4 -c:v libvpx-vp9 -b:v 0 -crf 40 -row-mt 1 -deadline good -cpu-used 4 -an vp9-crf40.webm
ffmpeg -hide_banner -loglevel error -y -i pw.webm $X264 -crf 26 fromwebm-crf26.mp4
for f in pw.webm x264-crf*.mp4 vp9-crf40.webm fromwebm-crf26.mp4; do
  printf '%s ' "$f"; ffprobe -v error -show_entries format=duration,size,bit_rate -of compact=p=0 "$f"
done
ffmpeg -hide_banner -loglevel error -y -ss 3 -i x264-crf28.mp4 -frames:v 1 -c:v libwebp -quality 75 poster-1440.webp
ffmpeg -hide_banner -loglevel error -y -ss 3 -i x264-crf28.mp4 -frames:v 1 -vf scale=1280:-2 -c:v libwebp -quality 75 poster-1280.webp
ls -l poster-*.webp
