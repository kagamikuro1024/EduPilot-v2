#!/usr/bin/env bash
# usage: run-rc.sh <outdir> <rcfile> <mult|-> url-path...   (lhci autorun = collect + assert, như CI; không có startServer: máy chủ :3330 + gateway giả :3332 chạy sẵn)
set -u
H=/Users/kuro/Documents/TA_Agent_qc2/docs/regression/evidence-5/lh
export NODE_PATH=$(ls -d /Users/kuro/checkouts/pq/frontend/node_modules/.pnpm/puppeteer-core@*/node_modules | head -1)
export CHROME_PATH="/Users/kuro/Library/Caches/ms-playwright/chromium-1255/chrome-mac-arm64/Google Chrome for Testing.app/Contents/MacOS/Google Chrome for Testing"
out=$1; rc=$2; mult=$3; shift 3
mkdir -p $H/$out && cd $H/$out && rm -rf .lighthouseci .lighthouseci-devtools
args=(autorun --config=$H/$rc --collect.numberOfRuns=${NRUNS:-5}); [ "$mult" != - ] && args+=(--collect.settings.throttling.cpuSlowdownMultiplier=$mult)
for p in "$@"; do args+=(--collect.url=http://localhost:3330$p); done
echo "uptime: $(uptime)"; /Users/kuro/checkouts/pq/frontend/node_modules/.bin/lhci "${args[@]}"; echo "rc=$?"; echo "uptime: $(uptime)"
