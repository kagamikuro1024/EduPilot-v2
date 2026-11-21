#!/bin/bash
cd ~/Documents/TA_Agent_qc2/docs/regression/evidence-4
bash scripts/s4n-pw.sh s4x5 desktop r4ui-audit.spec.ts -g "· (student|ta)$"
bash scripts/s4n-pw.sh s4x6 desktop r4ui-audit.spec.ts -g "· (teacher|admin)$"
