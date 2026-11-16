#!/usr/bin/env bash
# usage: visual-docker.sh <log> [playwright args...]  — visual.spec.ts trong image mcr.microsoft.com/playwright:v1.63.0-noble (khớp @playwright/test 1.63.0 của pnpm-lock), checkout pq2 (cài --frozen-lockfile + build:gate), KHÔNG --update-snapshots.
export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock
log=$1; shift
docker run --rm --ipc=host -v /Users/kuro/checkouts/pq2:/work -w /work/frontend -e PATH=/work/.shim:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin mcr.microsoft.com/playwright:v1.63.0-noble pnpm exec playwright test visual.spec.ts --workers=2 --reporter=list --output=/tmp/pwout "$@" > $log 2>&1
echo rc=$? >> $log; tail -25 $log
