#!/bin/bash
# S9R: một lượt = một khoá; chạy song song các người chờ để tăng cơ hội giành khoá
cd ~/Documents/TA_Agent_qc2/docs/regression/evidence-9.1
run(){ spec=$1; cls=$2; [ "$cls" = "-" ] && cls=""; bash scripts/r91-bump.sh >/dev/null 2>&1 </dev/null; tag=$3; grp=$4; QC_TAG=$tag PWL_MAX=470 PWL_WAIT=14400 /tmp/smks/pwlock.sh bash run8.sh "$spec" "$cls" "$grp" > /tmp/smks/q-$spec-$tag.out 2>&1 </dev/null; echo "$(date +%T) done $spec $tag" >> /tmp/smks/queue.log; }
run "$@"
