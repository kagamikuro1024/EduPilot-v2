#!/bin/bash
# SmkS4 master: chạy lần lượt (mỗi lượt giữ khoá ≤ 8 phút, chờ khoá tối đa 3 giờ)
cd ~/Documents/TA_Agent_qc2/docs/regression/evidence-9.1
run(){ spec=$1; cls=$2; tag=$3; grp=$4; bash scripts/r91-bump.sh >/dev/null 2>&1 </dev/null; QC_TAG=$tag PWL_MAX=470 PWL_WAIT=10800 /tmp/smks/pwlock.sh bash run8.sh "$spec" "$cls" "$grp" > /tmp/smks/q-$spec-$tag.out 2>&1 </dev/null; echo "$(date +%T) done $spec $tag" >> /tmp/smks/queue.log; sleep 20; }
run qc-uiux91-ru1-real "" w TC-UX18-4
run qc-uiux91-r91sv-real 761987 w TC-UX13-6
run qc-uiux91-ru1-real "" x TC-UX12-2
run qc-uiux91-rw4-real QR91A w TC-UX12-9/10
run qc-uiux91-rn4-real QR91H w UX16-14
echo "$(date +%T) MASTER1 END" >> /tmp/smks/queue.log
