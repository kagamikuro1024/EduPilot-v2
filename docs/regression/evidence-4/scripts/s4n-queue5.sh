#!/bin/bash
cd ~/Documents/TA_Agent_qc2/docs/regression/evidence-4/scripts
while [ ! -f ../logs/s4n-queue4.done ]; do sleep 5; done
bash s4n-pw.sh s5nav2 desktop smkt3-nav smkt3-p20231 s4n-dbg; sleep 35
echo queue5-done > ../logs/s4n-queue5.done
