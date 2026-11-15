export QC_API=https://localhost:1143/api/v1 SEED_DEFAULT_PASSWORD='Edupilot#Seed-2026' NODE_TLS_REJECT_UNAUTHORIZED=0
cd "$(dirname "$0")"
for f in as_f as_e as_i2 as_j2; do echo "=== $f"; timeout 900 python3 $f.py > ../$f.log 2>&1; echo rc=$?; done
