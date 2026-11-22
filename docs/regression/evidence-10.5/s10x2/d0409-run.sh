cd "$(dirname "$0")"; export SEED_DEFAULT_PASSWORD='Edupilot#Seed-2026'
for m in always_yes always_no error; do python3 d0409.py $m; done > d0409-modes.log 2>&1
python3 d0409.py fin >> d0409-modes.log 2>&1; echo FIN >> d0409-modes.log
