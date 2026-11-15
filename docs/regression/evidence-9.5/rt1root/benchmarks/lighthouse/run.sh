#!/usr/bin/env bash
# US-NFR-07 AC4: Lighthouse (mô phỏng + devtools) trên BẢN BUILD PRODUCTION của frontend ở stack đo ep-*, đăng nhập thật bằng tài khoản T1.
#   LH_ORIGIN=https://localhost:843 SEED_DEFAULT_PASSWORD=… LHCI=<đường dẫn lhci> bash benchmarks/lighthouse/run.sh
# Chạy từ thư mục gốc repo; dùng Chrome của máy (CHROME_PATH). Kết quả thô: benchmarks/.lighthouse-raw/<mode>/*.json → `node benchmarks/lighthouse/summarize.mjs`.
set -euo pipefail
: "${LH_ORIGIN:?}" "${SEED_DEFAULT_PASSWORD:?}"
case "$LH_ORIGIN" in https://localhost:443|https://localhost) echo "Từ chối: stack demo" >&2; exit 1;; esac
LHCI=${LHCI:-lhci}
export CHROME_PATH=${CHROME_PATH:-"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"} NODE_TLS_REJECT_UNAUTHORIZED=0
ASSIGN=${LH_ASSIGNMENT_PATH:-}
URLS=(/ /chat /threads /inbox /gradebook /me /settings/llm $ASSIGN)
for mode in simulate devtools; do
  rm -rf "benchmarks/.lighthouse-raw/$mode"
  args=(); for u in "${URLS[@]}"; do args+=(--url="$LH_ORIGIN$u"); done
  "$LHCI" collect --config=benchmarks/lighthouse/lighthouserc.ep.json "${args[@]}" --settings.throttlingMethod=$mode
  mkdir -p "benchmarks/.lighthouse-raw/$mode" && cp .lighthouseci/lhr-*.json "benchmarks/.lighthouse-raw/$mode/" && rm -rf .lighthouseci
done
