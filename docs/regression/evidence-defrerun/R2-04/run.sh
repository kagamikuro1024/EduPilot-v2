cd ~/Documents/TA_Agent_qc2/backend-go
export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true
D=~/Documents/TA_Agent_qc2/docs/regression/evidence-defrerun/R2-04
( time go test -count=1 ./internal/contract ) > $D/notag.log 2>&1; echo "rc=$?" >> $D/notag.log
sleep 30
( time go test -count=1 -tags testroutes ./internal/contract ) > $D/tag.log 2>&1; echo "rc=$?" >> $D/tag.log
echo done > $D/done
