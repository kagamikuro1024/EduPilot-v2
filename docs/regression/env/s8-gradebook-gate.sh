#!/bin/bash
cd ~/Documents/TA_Agent_qc2/frontend
L=~/Documents/TA_Agent_qc2/docs/regression/evidence-8/logs
git -C .. rev-parse --short HEAD > $L/seed26-head.txt
(time pnpm build:gate) > $L/seed26-build.log 2>&1; echo "build rc=$?" >> $L/seed26-build.log
pnpm exec playwright test e2e/gradebook.spec.ts --list 2>&1 | tail -3 > $L/seed27-list.txt
pnpm exec playwright test e2e/gradebook.spec.ts --list 2>&1 | grep -c "›" >> $L/seed27-list.txt
pnpm exec playwright test e2e/gradebook.spec.ts -g 'lop2 demo step 6' --output=$HOME/Documents/TA_Agent_qc2/docs/regression/evidence-8/pwout17 > $L/seed17.log 2>&1; echo "rc=$?" >> $L/seed17.log
pnpm exec playwright test e2e/gradebook.spec.ts --workers=2 --output=$HOME/Documents/TA_Agent_qc2/docs/regression/evidence-8/pwout26 > $L/seed26.log 2>&1; echo "rc=$?" >> $L/seed26.log
echo DONE >> $L/seed26.log
