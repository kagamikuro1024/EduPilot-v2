#!/bin/bash
cd ~/Documents/TA_Agent_qc2/docs/regression/evidence-4/scripts
while [ ! -f ../logs/s4n-queue6.done ]; do sleep 5; done
bash s4n-pw.sh s4c3 desktop ui-s4c -g "P210-32"; sleep 35
echo queue7-done > ../logs/s4n-queue7.done
