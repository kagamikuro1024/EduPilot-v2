#!/bin/bash
cd ~/Documents/TA_Agent_qc2/docs/regression/evidence-9.5/scripts
for m in devtools simulate; do MODES=$m PWL_MAX=470 PWL_WAIT=10800 /tmp/smks/pwlock.sh bash lh-smk91.sh > ../rerun/n07-09-lh-$m-smk91.log 2>&1 </dev/null; echo "$(date +%T) LH $m done" >> /tmp/smks/queue.log; sleep 20; done
