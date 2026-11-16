#!/bin/bash
# usage: pkgrun.sh <outdir> <tags|-> pkg1 pkg2 ...   — go test -race theo từng gói, tuần tự, thử lại tối đa 3 lần nếu đỏ (tải máy)
cd /Users/kuro/Documents/TA_Agent_qc2/backend-go
export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true EP_SKIP_TIMING=1
D=$1; tags=$2; shift 2; mkdir -p $D; S=$D/summary.tsv
t=(); [ "$tags" != "-" ] && t=(-tags "$tags")
for p in "$@"; do
  n=$(echo $p | sed 's#[./]#_#g')
  for try in 1 2 3; do
    t0=$(date +%s)
    timeout 900 go test -race -count=1 -timeout 420s "${t[@]}" -json $p > $D/$n.json 2> $D/$n.err; rc=$?
    [ $rc = 0 ] && break
    echo "$(date +%T) $p rc=$rc try=$try load=$(uptime | sed 's/.*averages*: //')" >> $D/progress.log; sleep 20
  done
  np=$(jq -rs '[.[]|select(.Action=="pass" and .Test!=null)]|length' $D/$n.json 2>/dev/null); nf=$(jq -rs '[.[]|select(.Action=="fail" and .Test!=null)]|length' $D/$n.json 2>/dev/null); ns=$(jq -rs '[.[]|select(.Action=="skip" and .Test!=null)]|length' $D/$n.json 2>/dev/null)
  printf '%s\t%s\ttags=%s\tpass=%s\tfail=%s\tskip=%s\ttry=%s\t%ss\n' "$p" "$rc" "$tags" "$np" "$nf" "$ns" "$try" "$(( $(date +%s)-t0 ))" >> $S
done
echo "$(date +%T) XONG" >> $D/progress.log
