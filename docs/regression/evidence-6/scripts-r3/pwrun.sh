#!/bin/bash
# usage: pwrun.sh script.mjs cases...  (đợi khoá Playwright chung, chạy, nhả khoá)
for i in $(seq 1 120); do mkdir /tmp/qc3-pw-lock 2>/dev/null && break; sleep 10; done
[ -d /tmp/qc3-pw-lock ] || exit 9
cd /tmp/qc3e/ui; NODE_TLS_REJECT_UNAUTHORIZED=0 timeout 500 node "$@"; rc=$?
rmdir /tmp/qc3-pw-lock; exit $rc
