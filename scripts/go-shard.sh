#!/usr/bin/env bash
# Chia gói Go cho dòng `rest-N` của CI theo THỜI GIAN (LPT: gói nặng nhất vào dòng đang nhẹ nhất), không theo thứ tự tên (DEF-R7-01: chia chẵn / lẻ dồn
# grade, crm, exam, grading, integration... vào một dòng). Trọng số = giây của gói ở lượt CI thật (backend-go/ci-shard-weights.txt); gói mới / chưa có số lấy 5 s.
# Dùng: scripts/go-shard.sh <dòng 1..N> <N>   (stdin: danh sách gói đã lọc, mỗi dòng một gói đủ đường dẫn module)
set -euo pipefail
idx=$1 n=$2
w="$(dirname "$0")/../backend-go/ci-shard-weights.txt"
awk -v idx="$idx" -v n="$n" -v w="$w" '
  BEGIN { while ((getline l < w) > 0) { split(l, a, " "); wt[a[1]] = a[2] } }
  { p = $0; k = p; sub(/^github.com\/edupilot\/backend-go\//, "", k); pk[NR] = p; ww[NR] = (k in wt) ? wt[k] : 5 }
  END {
    for (i = 1; i <= NR; i++) ord[i] = i
    for (i = 2; i <= NR; i++) { x = ord[i]; j = i - 1; while (j >= 1 && (ww[ord[j]] < ww[x] || (ww[ord[j]] == ww[x] && pk[ord[j]] > pk[x]))) { ord[j+1] = ord[j]; j-- } ord[j+1] = x }
    for (b = 1; b <= n; b++) sum[b] = 0
    for (i = 1; i <= NR; i++) { m = 1; for (b = 2; b <= n; b++) if (sum[b] < sum[m]) m = b; sum[m] += ww[ord[i]]; if (m == idx) print pk[ord[i]] }
  }'
