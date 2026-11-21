#!/bin/bash
cd ~/Documents/TA_Agent_qc2/docs/regression/evidence-4/scripts
while pgrep -f "s4r-queue8.sh" >/dev/null; do sleep 5; done
bash s4n-pw.sh s4r14 desktop ui-s4c -g "P210-32"
