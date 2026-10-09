#!/bin/bash
# Khởi động go-judge: chỉ nhận cấu hình qua biến JUDGE_* (container không có biến nào khác). `-enable-debug` KHÔNG bật.
# JUDGE_EXTRA_ARGS trống ở amd64 (seccomp bật); trên colima arm64 đặt `-no-seccomp` ở .env.local (D58: v1.13.0 + seccomp hỏng ở arm64).
set -euo pipefail
if [ "${#JUDGE_TOKEN}" -lt 16 ]; then
  echo "JUDGE_TOKEN phải có ít nhất 16 ký tự" >&2
  exit 1
fi
# go-judge in cấu hình (kể cả AuthToken) và dòng "Attach token auth" ra log lúc khởi động → AC3 (token không xuất hiện ở log):
# lọc mọi dòng nhật ký qua bash (thay NGUYÊN VĂN, không regex nên token có ký tự đặc biệt vẫn đúng). exec giữ go-judge là PID 1.
scrub() {
  local l
  while IFS= read -r l || [ -n "$l" ]; do printf '%s\n' "${l//"$JUDGE_TOKEN"/***}"; done
}
exec > >(scrub) 2>&1
# shellcheck disable=SC2086
exec /opt/go-judge -http-addr=:5050 -parallelism="${JUDGE_PARALLELISM:-2}" -auth-token="$JUDGE_TOKEN" ${JUDGE_EXTRA_ARGS:-}
