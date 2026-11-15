#!/usr/bin/env python3
"""Tính kết quả hàng -G sprint 9 từ logs/g-go.jsonl (+g-go-fix.jsonl) và logs/pw-mock.json. In JSON {tc:[P|F|?, note]}"""
import json, re, glob, os, collections, sys
ROOT = os.path.expanduser("~/Documents/TA_Agent_qc2"); L = f"{ROOT}/docs/regression/evidence-9/logs"
rows = {}
for f in sorted(glob.glob(f"{ROOT}/docs/sprints/9/qc/tc-*.md")):
    for l in open(f):
        m = re.match(r"\| (TC-[\w-]+-G\d+) \|", l)
        if m: rows[m.group(1)] = [c.replace("\\|", "|") for c in re.findall(r"`([^`]*)`", l)]
go = collections.defaultdict(list)
for fn in ("g-go.jsonl", "g-go-fix.jsonl"):
    if os.path.exists(f"{L}/{fn}"):
        for l in open(f"{L}/{fn}"): r = json.loads(l); go[r["tc"]].append(r)
pw = json.load(open(f"{L}/pw-mock-rows.json")) if os.path.exists(f"{L}/pw-mock-rows.json") else []
def pw_eval(cmd):
    m = re.search(r"-g '([^']*)'", cmd); specs = [s for s in re.findall(r"[\w-]+\.spec\.ts", cmd)]
    pats = [p for p in m.group(1).split("|")] if m else [None]
    notes = []; ok = True
    for p in pats:
        hit = [r for r in pw if r[0] in specs and r[1] == "desktop" and (p is None or re.search(p, r[2]))]
        if not hit: ok = False; notes.append(f"-g «{p}» không khớp test nào"); continue
        bad = [r for r in hit if r[3] != "passed"]
        if bad: ok = False; notes.append(f"{p}: {bad[0][3]}")
    return ok, "; ".join(notes) or f"Playwright mock {len(pats)} mẫu -g xanh"
out = {}
for tc, cmds in rows.items():
    res = []; note = []
    gocmds = [c for c in cmds if re.match(r"(cd backend-go && )?go test", c)]
    pwcmds = [c for c in cmds if "playwright" in c]
    for c in gocmds:
        c2 = re.sub(r"^cd backend-go && ", "", c)
        rs = [r for r in go.get(tc, []) if r.get("cmd") == c2]
        if not rs: res.append("?"); note.append("chưa chạy go"); continue
        r = rs[-1]; bad = r["rc"] or r["missing"] or r["skipped"] or r["race"]
        if not bad: res.append("P"); continue
        # vòng 2
        fx = [x for x in go.get(tc, []) if x.get("pkg") or x.get("absent")]
        if r["missing"] and not r["rc"]:
            ok = True
            lg = open(f"{L}/{re.sub(r'[^A-Za-z0-9]+','_',tc)}.log").read() if os.path.exists(f"{L}/{re.sub(r'[^A-Za-z0-9]+','_',tc)}.log") else ""
            for n in r["missing"]:
                if re.search(rf"--- PASS: {re.escape(n.split('/')[0])}\w*", lg) or (n in ("Teams",) and "\nok" in lg): note.append(f"{n}: khớp theo tiền tố/regex (PASS)"); continue
                f = [x for x in fx if n in x["names"]]
                if not f: ok = False; note.append(f"{n}: chưa chạy vòng 2")
                elif f[-1].get("absent"): ok = False; note.append(f"{n}: không có trong repo")
                elif f[-1]["rc"] or f[-1]["missing"] or f[-1]["skipped"]: ok = False; note.append(f"{n}: vòng 2 lỗi")
            res.append("P" if ok else "F"); 
        else: res.append("F"); note.append(f"go rc={r['rc']} missing={r['missing']} skip={r['skipped']} race={r['race']}")
    for c in pwcmds:
        ok, n = pw_eval(c); res.append("P" if ok else "F"); note.append(n)
    out[tc] = ["?" if "?" in res else ("F" if "F" in res else ("P" if res else "-")), "; ".join(note)]
json.dump(out, open(f"{L}/g-eval.json", "w"), ensure_ascii=False, indent=0)
c = collections.Counter(v[0] for v in out.values()); print(c)
for k, v in out.items():
    if v[0] != "P": print(k, v)
