#!/bin/bash
# guard.sh <mốc_swap_MB> — chạy song song với lô: dừng xcodebuild nếu swap vượt mốc > 1024 MB hoặc free < 20 % (luật PM 26/11).
SW0="$1"
while pgrep -f "xcodebuild test" >/dev/null; do
  F=$(memory_pressure | awk '/free percentage/{gsub("%","",$5);print $5}'); S=$(sysctl -n vm.swapusage | sed -E 's/.*used = ([0-9.]+)M.*/\1/')
  D=$(python3 -c "print(int($S-$SW0))")
  echo "$(date +%T) free=$F% swap+${D}MB" >> "$(dirname "$0")/.cache/guard.log"
  if [ "$F" -lt 20 ] || [ "$D" -gt 1024 ]; then echo "GUARD free=$F% swap+${D}MB — dừng"; pkill -f "xcodebuild test"; pkill -f Qc2UITests-Runner; exit 1; fi
  sleep 15
done
