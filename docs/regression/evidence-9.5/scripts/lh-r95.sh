#!/bin/bash
# N07-09: Lighthouse (lighthouse CLI, không có lhci) cho / và /assignments/[id] trên frontend production của stack README ep-r95 (https://localhost:1343), SV T1 t1sv0001, cookie ep_rt qua --extra-headers.
LH=/Users/kuro/.npm/_npx/0f94ee7615faf582/node_modules/lighthouse/cli/index.js
export CHROME_PATH="/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" NODE_TLS_REJECT_UNAUTHORIZED=0
O=${1:-/tmp/qc2-r95b/lh}; mkdir -p $O; B=https://localhost:1343; PW='Edupilot#Seed-2026'
hdr=$(curl -sk -D - -o /tmp/qc2-r95b/lh-login.json -X POST $B/api/v1/auth/login -H 'content-type: application/json' -H 'Origin: https://localhost:1343' -d "{\"email\":\"t1sv0001@edupilot.local\",\"password\":\"$PW\"}")
RT=$(echo "$hdr" | tr -d '\r' | grep -i '^set-cookie: ep_rt=' | sed 's/^[Ss]et-[Cc]ookie: //; s/;.*//')
TOK=$(jq -r .access_token /tmp/qc2-r95b/lh-login.json)
curl -sk -o /dev/null -w 'ack %{http_code}\n' -X POST $B/api/v1/me/tracking-notice/ack -H "Authorization: Bearer $TOK" -H 'Origin: https://localhost:1343'
CID=$(curl -sk $B/api/v1/me/courses -H "Authorization: Bearer $TOK" | jq -r '.items[0].course.id')
AID=$(curl -sk "$B/api/v1/courses/$CID/assignments?limit=5" -H "Authorization: Bearer $TOK" | jq -r '.items[0].id')
echo "rt=${RT:0:12}… course=$CID assignment=$AID"
for mode in simulate devtools; do for u in / /assignments/$AID; do for i in 1 2 3; do
  f=$O/$mode-$(echo $u | tr '/' '_')-$i.json
  # cookie ep_rt mới cho mỗi lượt (refresh cookie xoay vòng)
  hdr=$(curl -sk -D - -o /tmp/qc2-r95b/lh-login.json -X POST $B/api/v1/auth/login -H 'content-type: application/json' -H 'Origin: https://localhost:1343' -d "{\"email\":\"t1sv0001@edupilot.local\",\"password\":\"$PW\"}"); RT=$(echo "$hdr" | tr -d '\r' | grep -i '^set-cookie: ep_rt=' | sed 's/^[Ss]et-[Cc]ookie: //; s/;.*//')
  node $LH "$B$u" --only-categories=performance --throttling-method=$mode --output=json --output-path=$f --quiet --chrome-flags="--headless=new --no-sandbox --ignore-certificate-errors" --extra-headers="{\"Cookie\":\"$RT\"}" >/dev/null 2>$f.err; echo "$mode $u $i rc=$?"
done; done; done
python3 - <<PY
import json,glob,statistics as st,re
rows={}
for f in sorted(glob.glob("$O/*.json")):
    l=json.load(open(f)); a=l["audits"]; m=re.match(r".*/(simulate|devtools)-(.*)-\d.json",f)
    k=(m.group(1), l["finalDisplayedUrl"].replace("https://localhost:1343","").replace("$AID","[id]"))
    rows.setdefault(k,[]).append((a["largest-contentful-paint"]["numericValue"],a["total-blocking-time"]["numericValue"],a["cumulative-layout-shift"]["numericValue"],next(i["transferSize"] for i in a["resource-summary"]["details"]["items"] if i["resourceType"]=="script")/1024))
for k,v in rows.items(): print(k,len(v),"LCP",round(st.median(x[0] for x in v)),"TBT",round(st.median(x[1] for x in v)),"CLS",round(st.median(x[2] for x in v),3),"script KB",round(st.median(x[3] for x in v)))
PY
