#!/bin/bash
cd "$(dirname "$0")"; L=../logs
while [ ! -f $L/chain3.done ]; do sleep 5; done
python3 -u q9_07a.py > $L/q9_07a2.log 2>&1
echo done > $L/chain4.done
