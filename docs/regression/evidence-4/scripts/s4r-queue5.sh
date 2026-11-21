#!/bin/bash
cd ~/Documents/TA_Agent_qc2/docs/regression/evidence-4/scripts
while pgrep -f "s4n-pw.sh s4r5 " >/dev/null; do sleep 5; done
bash s4n-pw.sh s4r9 desktop ui-s4d -g "P211-33|P211-41"; sleep 20
bash s4n-pw.sh s4r10 desktop ui-s4b -g "P209-45"
