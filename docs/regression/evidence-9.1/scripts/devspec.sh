#!/bin/bash
# devspec.sh <tag> <spec...> : bộ playwright dev (bản build:gate sẵn có trong frontend/.next), log -> logs/devspec-<tag>.log
source ~/Documents/TA_Agent_qc2/docs/regression/env/ep-r-lib.sh
cd $REPO/frontend
T=$1; shift
timeout 420 pnpm exec playwright test "$@" --project=desktop --workers=2 --reporter=list 2>&1 | grep -v -e NO_COLOR -e trace-warnings -e WebServer > $REPO/docs/regression/evidence-9.1/logs/devspec-$T.log
tail -5 $REPO/docs/regression/evidence-9.1/logs/devspec-$T.log
