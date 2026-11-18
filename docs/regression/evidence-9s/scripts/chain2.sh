#!/bin/bash
cd "$(dirname "$0")"; L=../logs
python3 -u q9_07c.py > $L/q9_07c.log 2>&1
python3 -u q9_08.py > $L/q9_08b.log 2>&1
python3 -u runstub.py q9p11.py QD9P1 > $L/q9p11.log 2>&1
echo done > $L/chain2.done
