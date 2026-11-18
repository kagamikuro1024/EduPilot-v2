#!/bin/bash
# queue.sh <file>: mỗi dòng "<spec> <lớp|-> <tag> <grep|->"; mỗi lượt giữ khoá ≤ 7,5 phút, nghỉ 35 s giữa các lượt
cd ~/Documents/TA_Agent_qc2/docs/regression/evidence-9.1
while IFS=' ' read -r spec cls tag grp; do
  [ -z "$spec" ] && continue
  [ "$cls" = "-" ] && cls=""
  [ "$grp" = "-" ] && grp=""
  QC_TAG=$tag PWL_MAX=450 /tmp/smks/pwlock.sh bash run7.sh "$spec" "$cls" "$grp" > /tmp/smks/q-$spec-$tag.out 2>&1
  echo "$(date +%T) done $spec $tag" >> /tmp/smks/queue.log
  sleep 35
done < "$1"
echo "$(date +%T) QUEUE END $1" >> /tmp/smks/queue.log
