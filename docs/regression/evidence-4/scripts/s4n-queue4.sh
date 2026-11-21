#!/bin/bash
cd ~/Documents/TA_Agent_qc2/docs/regression/evidence-4/scripts
while [ ! -f ../logs/s4n-queue3.done ]; do sleep 5; done
bash s4n-pw.sh s4d2 desktop ui-s4d; sleep 35
echo queue4-done > ../logs/s4n-queue4.done
