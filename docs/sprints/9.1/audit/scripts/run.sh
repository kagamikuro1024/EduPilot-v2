#!/bin/bash
# Chạy tiếp tới khi xong (trình duyệt đôi khi chết giữa chừng): bash run.sh <vai>...
cd "$(dirname "$0")"
for i in 1 2 3 4 5 6; do node shoot.mjs "$@" && exit 0; echo "lần $i lỗi, chạy tiếp"; sleep 3; done
