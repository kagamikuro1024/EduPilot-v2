#!/bin/bash
# s8core: chạy các Go test mà tc-US-P6-01..04 nêu tên, theo từng gói; log -v vào evidence-8/logs/s8core-go-<gói>.log
cd $HOME/Documents/TA_Agent_qc2; E=docs/regression/evidence-8; L=$E/logs
export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true
names(){ f=$(grep -lE "func $1\(" -r backend-go/internal/$2 --include=*_test.go 2>/dev/null); [ -n "$f" ] && echo $1; }
for pkg in grade grading store integration; do
  N=$(grep -ohE 'Test[A-Za-z0-9_]+' docs/sprints/8/qc/tc-US-P6-0[1-4].md | sort -u | while read t; do names $t $pkg; done | tr '\n' '|' | sed 's/|$//')
  echo "== $pkg: $N" >> $L/s8core-go-names.log
  (cd backend-go && timeout 290 go test -count=1 -timeout 280s -tags integration -v -run "^($N)\$" ./internal/$pkg > ../$L/s8core-go-$pkg.log 2>&1; echo "rc=$?" >> ../$L/s8core-go-$pkg.log)
done
echo DONE > $L/s8core-go.done
