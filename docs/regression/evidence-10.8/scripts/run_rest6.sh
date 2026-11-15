export QC_API=https://localhost:1143/api/v1 SEED_DEFAULT_PASSWORD='Edupilot#Seed-2026' NODE_TLS_REJECT_UNAUTHORIZED=0
cd "$(dirname "$0")"
timeout 400 python3 as_arch.py > ../as_arch.log 2>&1
timeout 500 python3 as_d5.py > ../as_d5.log 2>&1
