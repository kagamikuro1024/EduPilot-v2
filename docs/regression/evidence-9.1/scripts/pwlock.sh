#!/bin/bash
# usage: pwlock.sh <cmd...> — giữ /tmp/qc2-pw.lock quanh lệnh (chờ tối đa 20 phút, nhả bằng trap EXIT, tối đa 8 phút giữ)
t=0
until mkdir /tmp/qc2-pw.lock 2>/dev/null; do sleep 3; t=$((t+3)); [ $t -ge ${PWL_WAIT:-1200} ] && { echo "KHÔNG LẤY ĐƯỢC KHOÁ"; exit 99; }; done
trap 'rmdir /tmp/qc2-pw.lock 2>/dev/null; echo "$(date +%T) SmkS: PW-UNLOCK" >> /tmp/qc2-global-changes.log' EXIT
echo "$(date +%T) SmkS: PW-LOCK — $*" >> /tmp/qc2-global-changes.log
timeout ${PWL_MAX:-480} "$@"
