#!/bin/bash
# S4X: DEF-R9.5-17 (TC-N02-02/03) trên ep-r bằng bản sao seed (seed-qc.mjs: chỉ đổi 3 truy vấn /admin/courses sang q=<mã lớp>, né DEF-R9.5-26) rồi check-*-seed.
export API_URL=https://localhost:1143/api/v1 MAILPIT_URL=http://localhost:8725 SEED_DEFAULT_PASSWORD='Edupilot#Seed-2026' COMPOSE_PROJECT_NAME=ep-r NODE_TLS_REJECT_UNAUTHORIZED=0
export SEED_ADMIN_CMD="docker exec -e ADMIN_PASSWORD=Edupilot#Seed-2026 ep-r-gateway-1 /gateway admin create"
export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock
R=~/Documents/TA_Agent_qc2; D=$R/docs/regression/evidence-defrerun/R9.5-17; cd $R
clrl(){ docker exec ep-r-redis-1 sh -c "(redis-cli --scan --pattern 'ep:rl:auth*'; redis-cli --scan --pattern 'ep:rl:ip:*') | xargs -r redis-cli del" > /dev/null; }
cnt(){ docker exec -i ep-r-postgres-1 psql -U edupilot -d edupilot -At -F ' ' -c "select table_name, (xpath('/row/c/text()', query_to_xml('select count(*) as c from public.'||quote_ident(table_name), false, true, '')))[1]::text from information_schema.tables where table_schema='public' and table_type='BASE TABLE' order by 1" > $1; }
clrl; cnt $D/n1-before.txt
for i in 1 2; do
  t=$(date +%s); /tmp/qc2-pw.sh bash -c "(while sleep 1; do docker exec ep-r-redis-1 sh -c \"(redis-cli --scan --pattern 'ep:rl:auth*'; redis-cli --scan --pattern 'ep:rl:ip:*') | xargs -r redis-cli del\" >/dev/null 2>&1; done) & L=\$!; node $D/seed-qc.mjs; r=\$?; kill \$L; exit \$r" > $D/seed-qc-run$i.log 2>&1; echo "rc=$? $(( $(date +%s) - t )) s" >> $D/seed-qc-run$i.log; tail -3 $D/seed-qc-run$i.log
  cnt $D/n1-after-run$i.txt
done
for f in scripts/check-{assignments,chat,crm,docs,exam,grade,knowledge}-seed.mjs; do clrl; /tmp/qc2-pw.sh node $f > $D/check-$(basename $f .mjs).log 2>&1; echo "$f $?" | tee -a $D/check-rc.txt; done
diff $D/n1-after-run1.txt $D/n1-after-run2.txt > $D/diff-run1-run2.txt; echo "diff rc=$? lines=$(wc -l < $D/diff-run1-run2.txt)"
cp /tmp/s4x-seed-t0.bak $R/benchmarks/reports/seed-t0.json
