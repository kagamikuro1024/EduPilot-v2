#!/usr/bin/env bash
# usage: run.sh <outdir-name> <runs> <preset: simulate|devtools|mobile> <cpuMult|-> url-path...
# Chạy lhci collect trên bản dựng cổng đang phục vụ ở :3330 (gateway giả lighthouse-api :3332). Số liệu ra lh/<outdir>/lhr-*.json
set -u
H=/Users/kuro/Documents/TA_Agent_qc2/docs/regression/evidence-5/lh
LHCI=/Users/kuro/checkouts/pq/frontend/node_modules/.bin/lhci
export NODE_PATH=/Users/kuro/checkouts/pq/frontend/node_modules/.pnpm/puppeteer-core@24.43.1_bare-buffer@3.7.2_supports-color@7.2.0/node_modules
export CHROME_PATH="/Users/kuro/Library/Caches/ms-playwright/chromium-1255/chrome-mac-arm64/Google Chrome for Testing.app/Contents/MacOS/Google Chrome for Testing"
out=$1; runs=$2; preset=$3; mult=$4; shift 4
mkdir -p $H/$out && cd $H/$out && rm -rf .lighthouseci
args=(collect --puppeteerScript=../lh-auth-qc.cjs -n $runs --puppeteerLaunchOptions.args=--no-sandbox --puppeteerLaunchOptions.args=--headless=new "--settings.chromeFlags=--no-sandbox --headless=new" --settings.onlyCategories=performance)
[ "$preset" = devtools ] && args+=(--settings.throttlingMethod=devtools)
[ "$preset" = mobile ] && args+=(--settings.formFactor=mobile)
[ "$mult" != - ] && args+=(--settings.throttling.cpuSlowdownMultiplier=$mult)
for p in "$@"; do args+=(--url=http://localhost:3330$p); done
echo "uptime: $(uptime)"; "$LHCI" "${args[@]}"
echo "rc=$?"; echo "uptime: $(uptime)"
