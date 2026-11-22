cd "$(dirname "$0")"; export SEED_DEFAULT_PASSWORD='Edupilot#Seed-2026'
for ph in a c b; do echo "== $ph $(date +%H:%M:%S) $(uptime | sed 's/.*load/load/')"; timeout 900 python3 d0107.py $ph; done > d0107.log 2>&1; echo FIN >> d0107.log
