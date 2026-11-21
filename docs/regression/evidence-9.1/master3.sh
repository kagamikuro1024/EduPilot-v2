#!/bin/bash
cd ~/Documents/TA_Agent_qc2/docs/regression/evidence-9.1
run(){ spec=$1; cls=$2; tag=$3; grp=$4; bash scripts/r91-bump.sh >/dev/null 2>&1 </dev/null; QC_TAG=$tag PWL_MAX=470 PWL_WAIT=10800 /tmp/smks/pwlock.sh bash run8.sh "$spec" "$cls" "$grp" > /tmp/smks/q-$spec-$tag.out 2>&1 </dev/null; echo "$(date +%T) done $spec $tag" >> /tmp/smks/queue.log; sleep 15; }
run qc-uiux91-ru1-real "" y TC-UX18-4
run qc-uiux91-r91sv-real 761987 y TC-UX13-6
echo "$(date +%T) MASTER3 END" >> /tmp/smks/queue.log
