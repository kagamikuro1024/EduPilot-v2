#!/bin/bash
cd ~/Documents/TA_Agent_qc2/docs/regression/evidence-4/scripts
while pgrep -f "s4r-queue7.sh" >/dev/null; do sleep 5; done
bash s4n-pw.sh s4r13 desktop s4r-join-e2e
