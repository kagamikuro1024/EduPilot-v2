#!/bin/bash
# usage: pwrun.sh <cmd...>  — đợi khoá Playwright toàn máy (/tmp/qc3-pw-lock), chạy, nhả khoá
for i in $(seq 1 180); do mkdir /tmp/qc3-pw-lock 2>/dev/null && break; sleep 10; done
[ -d /tmp/qc3-pw-lock ] || exit 9
NODE_TLS_REJECT_UNAUTHORIZED=0 timeout ${PWTIMEOUT:-1500} "$@"; rc=$?
rmdir /tmp/qc3-pw-lock; exit $rc
