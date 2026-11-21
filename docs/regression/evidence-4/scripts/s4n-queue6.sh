#!/bin/bash
cd ~/Documents/TA_Agent_qc2/docs/regression/evidence-4/scripts
while [ ! -f ../logs/s4n-queue5.done ]; do sleep 5; done
bash s4n-pw.sh s4b3 desktop ui-s4b -g "P209-45"; sleep 35
echo queue6-done > ../logs/s4n-queue6.done
