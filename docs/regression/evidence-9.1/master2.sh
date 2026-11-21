#!/bin/bash
cd ~/Documents/TA_Agent_qc2/docs/regression/evidence-9.1
QC_TAG=w PWL_MAX=470 PWL_WAIT=10800 /tmp/smks/pwlock.sh bash run8.sh qc-uiux91-rx1-real "" "" > /tmp/smks/q-rx1.out 2>&1 </dev/null; echo "$(date +%T) done rx1 w" >> /tmp/smks/queue.log
