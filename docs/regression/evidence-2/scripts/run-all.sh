#!/usr/bin/env bash
# QC sprint 2 — chạy tuần tự các story rồi cổng. Mỗi story ghi nhật ký vào $QC_OUT_DIR (mặc định /tmp/epqc-run-<giờ>).
#   bash docs/sprints/2/qc/scripts/run-all.sh [01 02 … 07 gate]      (mặc định: 01 02 03 04 05 06 07 gate)
# Thứ tự có lý do: 02 trước (đếm bảng/index cần DB CHƯA qua chế độ test — script pg02 tự dựng DB sạch), 03–05 dùng route thử, 07 chạm toàn stack, gate cuối cùng (có xoá volume).
# KHÔNG chạy trong phase 1 (chưa có code). Kết thúc: in bảng PASS/FAIL/MANUAL từng story; rc≠0 nếu có FAIL.
here=$(cd "$(dirname "$0")" && pwd)
out=${QC_OUT_DIR:-/tmp/epqc-run-$(date +%H%M%S)}; mkdir -p "$out"
list=${*:-01 02 03 04 05 06 07 gate}
rc=0
for s in $list; do
  if [ "$s" = gate ]; then f=$here/gate-pg.sh; else f=$here/pg$s.sh; fi
  [ -f "$f" ] || { echo "?? thiếu $f"; rc=1; continue; }
  echo "######## $s — $(date +%T)"; bash "$f" 2>&1 | tee "$out/$s.log" | grep -E '^(-- |TỔNG|FAIL:|MANUAL:)'
  [ "${PIPESTATUS[0]}" = 0 ] || rc=1
done
echo "nhật ký: $out"; exit $rc
