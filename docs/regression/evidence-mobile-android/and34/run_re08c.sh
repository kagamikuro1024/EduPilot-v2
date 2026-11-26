#!/bin/bash
python3 -c "import lib34 as L, m, time; m.login('sv.qc2'); time.sleep(3)"
echo "== nen"; python3 a11_idle.py 2>&1 | grep -vE '^mem'
echo "== 429 dem TTS (khong CDP)"; python3 a11_re08b.py 2>&1 | grep -vE '^mem'
sleep 50
echo "== 429 nhat ky thay doi"; python3 a11_re08.py 2>&1 | grep -vE '^mem'
