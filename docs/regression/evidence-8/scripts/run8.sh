cd $HOME/Documents/TA_Agent_qc2; E=docs/regression/evidence-8; L=$E/logs
export GW=https://localhost:1143 NODE_TLS_REJECT_UNAUTHORIZED=0 PSQL="docker exec -i ep-r-postgres-1 psql -U edupilot -d edupilot -tAc"
python3 docs/sprints/8/qc/scripts/p6-hand-calc.py > $L/hand-calc.log 2>&1; echo rc=$? >> $L/hand-calc.log
python3 docs/sprints/8/qc/scripts/p6-hand-calc.py A B C sv04 sv05 sv06 sv13 sv09 sv20 > $L/hand-calc-steps.log 2>&1; echo rc=$? >> $L/hand-calc-steps.log
bash $E/scripts/p6-rbac.sh > $L/rbac-own.log 2>&1; echo rc=$? >> $L/rbac-own.log
source /tmp/ep-r-lib.sh; T=$(tok teacher); C1=$(docker exec ep-r-postgres-1 psql -U edupilot -d edupilot -Atc "select id from courses where class_code='761987'"); curl -sk "$B/courses/$C1/gradebook/preview?limit=100" -H "authorization: Bearer $T" > /tmp/pv.json; python3 docs/sprints/8/qc/scripts/p6-preview-vs-csv.py --by-account > $L/preview-vs-csv.log 2>&1; echo rc=$? >> $L/preview-vs-csv.log
for r in 1 2; do :; done
cd backend-go; export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true
timeout 280 go test -count=1 -timeout 270s ./internal/grade -run 'TestEngineExpected30|TestEngineCases|TestNoFloatInGrade|TestForbiddenImports|TestConfirmOnlyFromHandler' -v > ../$L/go-grade-core.log 2>&1; echo rc=$? >> ../$L/go-grade-core.log
echo DONE > ../$L/run8.done
