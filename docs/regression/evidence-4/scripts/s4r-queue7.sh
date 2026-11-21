#!/bin/bash
cd ~/Documents/TA_Agent_qc2/docs/regression/evidence-4/scripts
while pgrep -f "s4r-queue6.sh" >/dev/null; do sleep 5; done
bash s4n-pw.sh s4r12 desktop s4r-dbg
