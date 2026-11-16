#!/usr/bin/env bash
cd /Users/kuro/Documents/TA_Agent_qc2/docs/regression/evidence-5/lh
while pgrep -f 'lhci autorun' >/dev/null; do sleep 10; done
export CHROME_PATH="/Users/kuro/Library/Caches/ms-playwright/chromium-1255/chrome-mac-arm64/Google Chrome for Testing.app/Contents/MacOS/Google Chrome for Testing"
echo "== take devtools"; THROT=devtools NODE_TLS_REJECT_UNAUTHORIZED=0 node take_lh.mjs nav 3 2>&1 | tail -4 > take_navd.out; cat take_navd.out
M=$(cd /Users/kuro/checkouts/pq/frontend && node lighthouse-multiplier.cjs 2>multiplier.err); echo "multiplier=$M"; cat /Users/kuro/checkouts/pq/frontend/multiplier.err 2>/dev/null
echo "== simulate autorun (CI config, 9 URL x5)"; NRUNS=5 ./run-rc.sh sim5 rc-lighthouserc.json $M > sim5.log 2>&1; tail -4 sim5.log
echo "== TBT 12x (7 URL x5)"; ./run.sh tbt12 5 simulate 12 / /chat /threads /inbox /gradebook /settings/llm /dev/ui > tbt12.log 2>&1; tail -3 tbt12.log
echo CHAIN-DONE
