#!/bin/bash
cd ~/Documents/TA_Agent_qc2/docs/regression/evidence-4/scripts
while [ ! -f ../logs/s4n-queue.done ]; do sleep 5; done
bash s4n-pw.sh s4b2 desktop ui-s4b; sleep 35
bash s4n-pw.sh s4c2 desktop ui-s4c; sleep 35
echo queue3-done > ../logs/s4n-queue3.done
