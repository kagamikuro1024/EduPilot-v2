#!/bin/bash
# TC-P5-06-17: Lighthouse mobile (mặc định của CLI) trên /me của ep-r, SV qc3me03 (một lớp, đã Đã hiểu), cookie ep_rt qua --extra-headers; hai chế độ throttling × 3 lượt
LH=/Users/kuro/.npm/_npx/0f94ee7615faf582/node_modules/lighthouse/cli/index.js
export CHROME_PATH="/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" NODE_TLS_REJECT_UNAUTHORIZED=0
O=${1:-/tmp/qc3e/lh-me}; mkdir -p $O; B=https://localhost:1143; PW='Edupilot#Seed-2026'
for mode in simulate devtools; do for i in 1 2 3; do
  hdr=$(curl -sk -D - -o $O/login.json -X POST $B/api/v1/auth/login -H 'content-type: application/json' -H "Origin: $B" -d "{\"email\":\"qc3me03@edupilot.local\",\"password\":\"$PW\"}")
  RT=$(echo "$hdr" | tr -d '\r' | grep -i '^set-cookie: ep_rt=' | sed 's/^[Ss]et-[Cc]ookie: //; s/;.*//')
  f=$O/$mode-$i.json
  node $LH "$B/me" --only-categories=performance,accessibility --throttling-method=$mode --output=json --output-path=$f --quiet --chrome-flags="--headless=new --no-sandbox --ignore-certificate-errors" --extra-headers="{\"Cookie\":\"$RT\"}" >/dev/null 2>$f.err; echo "$mode $i rc=$?"
done; done
python3 - <<PY
import json,glob,statistics as st
rows={}
for f in sorted(glob.glob("$O/*-*.json")):
    if 'login' in f: continue
    l=json.load(open(f)); a=l["audits"]; m='simulate' if '/simulate' in f else 'devtools'
    rows.setdefault(m,[]).append((a["largest-contentful-paint"]["numericValue"],a["total-blocking-time"]["numericValue"],a["cumulative-layout-shift"]["numericValue"],next(i["transferSize"] for i in a["resource-summary"]["details"]["items"] if i["resourceType"]=="script")/1024,l["categories"]["accessibility"]["score"],l["finalDisplayedUrl"],l["configSettings"]["formFactor"]))
for k,v in rows.items(): print(k,len(v),"LCP",[round(x[0]) for x in v],"TBT",[round(x[1]) for x in v],"CLS",[round(x[2],4) for x in v],"scriptKB",[round(x[3]) for x in v],"a11y",[x[4] for x in v],v[0][5],v[0][6])
PY
