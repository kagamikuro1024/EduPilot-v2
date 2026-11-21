#!/bin/bash
for i in $(seq 1 120); do mkdir /tmp/qc2-pw.lock 2>/dev/null && break; sleep 10; done
echo "$(date +%T) qc2: PW LOCK $*" >> /tmp/qc2-global-changes.log
trap 'rmdir /tmp/qc2-pw.lock 2>/dev/null; echo "$(date +%T) qc2: PW UNLOCK" >> /tmp/qc2-global-changes.log' EXIT
"$@"
