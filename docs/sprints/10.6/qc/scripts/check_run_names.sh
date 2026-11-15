#!/usr/bin/env bash
# Cổng «-run không rỗng» (P-106-7): mọi tên Test* nêu trong TC 10.6 phải có hàm thật; với các tên đã có, chạy -v và đòi `--- PASS`.
# Dùng: bash docs/sprints/10.6/qc/scripts/check_run_names.sh [tên ...]   (không tham số: lấy mọi tên trong tc-*.md)
set -u
cd "$(dirname "$0")/../../../../../backend-go" || exit 2
if [ $# -gt 0 ]; then names="$*"; else names=$(grep -ohE "Test[A-Z][A-Za-z0-9_]+" ../docs/sprints/10.6/qc/tc-*.md | sort -u); fi
missing=0
for t in $names; do
  pkg=$(grep -rlE "func $t\(" --include=*_test.go . | head -1)
  if [ -z "$pkg" ]; then echo "THIẾU   $t"; missing=$((missing + 1)); continue; fi
  tags=""; case "$pkg" in ./internal/contract/*) tags="-tags testroutes";; esac
  out=$(go test $tags "$(dirname "$pkg")" -run "^$t\$" -count=1 -v 2>&1)
  if echo "$out" | grep -q -- "--- PASS: $t "; then echo "PASS    $t"; else echo "KHÔNG PASS $t ($pkg)"; missing=$((missing + 1)); fi
done
echo "tổng không đạt: $missing"; [ "$missing" -eq 0 ]
