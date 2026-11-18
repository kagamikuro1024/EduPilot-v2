#!/bin/bash
cd "$(dirname "$0")"; L=../logs
while [ ! -s $L/q9_08c.log ] || ! grep -q "^RESULT" $L/q9_08c.log; do sleep 5; done
python3 -u runstub.py q9p11.py QD9T sv.gioi sv.kha sv04 sv05 sv06 sv07 sv08 sv09 sv10 sv11 sv12 sv13 sv14 sv15 sv16 sv18 sv19 sv20 sv22 sv24 sv25 sv27 sv29 sv30 > $L/q9p11.log 2>&1
python3 -u q9_07d.py > $L/q9_07d.log 2>&1
echo done > $L/chain3.done
