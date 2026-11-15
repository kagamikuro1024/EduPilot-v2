export QC_API=https://localhost:1143/api/v1 SEED_DEFAULT_PASSWORD='Edupilot#Seed-2026' NODE_TLS_REJECT_UNAUTHORIZED=0
cd "$(dirname "$0")"
for f in as_e as_f as_h as_k as_i as_j2; do echo "=== $f"; timeout 700 python3 $f.py > ../$f.log 2>&1; echo rc=$?; done
