#!/bin/bash
# queue3.sh <file>: như queue2.sh nhưng gọi run8.sh (config qc2g); mỗi dòng "<spec> <lớp|-> <tag> <grep|->"
cd ~/Documents/TA_Agent_qc2/docs/regression/evidence-9.1
while IFS=' ' read -r spec cls tag grp; do
  [ -z "$spec" ] && continue
  [ "$cls" = "-" ] && cls=""; [ "$grp" = "-" ] && grp=""
  bash scripts/r91-bump.sh >/dev/null 2>&1 < /dev/null
  QC_TAG=$tag PWL_MAX=450 /tmp/smks/pwlock.sh bash run8.sh "$spec" "$cls" "$grp" > /tmp/smks/q-$spec-$tag.out 2>&1 < /dev/null
  echo "$(date +%T) done $spec $tag" >> /tmp/smks/queue.log
  sleep 35
done < "$1"
echo "$(date +%T) QUEUE END $1" >> /tmp/smks/queue.log
