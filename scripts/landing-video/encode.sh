#!/bin/bash
# Mã hoá video giới thiệu (TL-1 A): khung JPEG + frames.ffconcat → mp4 H.264, webm VP9, poster WebP. Chạy TRONG container (run.sh):
#   docker run --rm -v $OUT:/out -v $DEST:/dest debian:trixie-slim bash /out/encode.sh
set -euo pipefail
apt-get update -qq >/dev/null && apt-get install -y -qq ffmpeg jq >/dev/null 2>&1
cd /out
if [ -z "${STILLS_ONLY:-}" ]; then
N=gioi-thieu-v1
ffmpeg -hide_banner -loglevel error -y -f concat -safe 0 -i frames.ffconcat -fps_mode cfr -r 30 \
  -c:v libx264 -preset slow -tune stillimage -crf 28 -maxrate 700k -bufsize 1400k -pix_fmt yuv420p -movflags +faststart -an /dest/$N.mp4
ffmpeg -hide_banner -loglevel error -y -i /dest/$N.mp4 -c:v libvpx-vp9 -b:v 0 -crf 46 -row-mt 1 -deadline good -cpu-used 2 -an /dest/$N.webm
ffmpeg -hide_banner -loglevel error -y -ss "$(jq -r .poster meta.json)" -i /dest/$N.mp4 -frames:v 1 -vf scale=960:-2 -c:v libwebp -quality 68 /dest/$N.webp
# clip lặp màn đầu (TL: ≤ 1 MB mỗi định dạng) + poster ≤ 60 KB = khung đầu của clip
C=gioi-thieu-clip-v1
ffmpeg -hide_banner -loglevel error -y -f concat -safe 0 -i clip.ffconcat -fps_mode cfr -r 30 \
  -c:v libx264 -preset slow -tune stillimage -crf 32 -maxrate 450k -bufsize 900k -pix_fmt yuv420p -movflags +faststart -an /dest/$C.mp4
ffmpeg -hide_banner -loglevel error -y -i /dest/$C.mp4 -c:v libvpx-vp9 -b:v 0 -crf 48 -row-mt 1 -deadline good -cpu-used 2 -an /dest/$C.webm
ffmpeg -hide_banner -loglevel error -y -i /dest/$C.mp4 -frames:v 1 -vf scale=1280:-2 -c:v libwebp -quality 48 /dest/$C.webp
ffmpeg -hide_banner -loglevel error -y -i /dest/$C.mp4 -frames:v 1 -vf scale=640:-2 -c:v libwebp -quality 38 /dest/$C-640.webp # poster điện thoại (nhỏ để về sớm, là LCP)
fi
# ba ảnh khung ghim: 1600 px (máy tính, ảnh gấp đôi) và 800 px (điện thoại)
for k in chat review gradebook; do
  ffmpeg -hide_banner -loglevel error -y -i story-$k.png -vf scale=1600:-2 -c:v libwebp -quality 70 /dest/story-$k-v1.webp
  ffmpeg -hide_banner -loglevel error -y -i story-$k.png -vf scale=800:-2 -c:v libwebp -quality 72 /dest/story-$k-v1-800.webp
done
ls -l /dest/story-*-v1*.webp
[ -n "${STILLS_ONLY:-}" ] && exit 0
ls -l /dest/$N.* /dest/$C.*
ffprobe -v error -show_entries format=duration -of csv=p=0 /dest/$C.mp4
ffprobe -v error -show_entries format=duration,size,bit_rate -of compact=p=0 /dest/$N.mp4
