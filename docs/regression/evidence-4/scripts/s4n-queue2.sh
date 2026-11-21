#!/bin/bash
cd ~/Documents/TA_Agent_qc2/docs/regression/evidence-4/scripts
while [ ! -f ../logs/s4n-queue.done ]; do sleep 5; done
bash s4n-pw.sh s5nav desktop smkt3-nav smkt3-p20231; sleep 35
bash s4n-pw.sh s6mock desktop smkt3-mock smkt3-g21; sleep 35
bash s4n-pw.sh s7bell desktop smkt3-bell; sleep 35
bash s4n-pw.sh s8jc desktop smkt3-journey s4n-caddy; sleep 35
bash s4n-pw.sh s9aud1 desktop r4ui-audit -g "student|ta$"; sleep 35
bash s4n-pw.sh s9aud2 desktop r4ui-audit -g "teacher|admin"; sleep 35
echo queue2-done > ../logs/s4n-queue2.done
