#!/bin/bash
# chạy tuần tự sau q9_07b
cd "$(dirname "$0")"; L=../logs
python3 -u q9_07b.py > $L/q9_07b.log 2>&1
python3 -u q9_08.py > $L/q9_08.log 2>&1
python3 -u runstub.py q9p09.py QD9P1 > $L/q9p09.log 2>&1
python3 -u runstub.py q9p10.py QD9P1 > $L/q9p10.log 2>&1
python3 -u runstub.py q9sweep.py QD9S sv.gioi sv.kha sv.nguyco sv04 sv05 sv06 sv07 > $L/q9sweep.log 2>&1
echo done > $L/chain.done
