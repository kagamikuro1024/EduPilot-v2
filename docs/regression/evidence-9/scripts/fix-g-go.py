#!/usr/bin/env python3
"""Vòng 2 cho hàng -G: (a) tên test thiếu ở gói nêu trong TC → tìm gói thật bằng grep, chạy với -tags integration,testroutes; (b) gói contract/httpapi cần -tags testroutes. Ghi logs/g-go-fix.jsonl"""
import json, os, re, subprocess, sys, time, collections
ROOT = os.path.expanduser("~/Documents/TA_Agent_qc2"); BE = f"{ROOT}/backend-go"; OUT = f"{ROOT}/docs/regression/evidence-9/logs"
env = dict(os.environ, DOCKER_HOST=f"unix://{os.path.expanduser('~')}/.colima/default/docker.sock", TESTCONTAINERS_RYUK_DISABLED="true")
rows = collections.defaultdict(list)
for l in open(f"{OUT}/g-go.jsonl"):
    r = json.loads(l); rows[r["tc"]].append(r)
def find_pkg(n):
    p = subprocess.run(["grep", "-rl", f"func {n}(", "--include=*_test.go", "."], cwd=BE, capture_output=True, text=True).stdout.split()
    return sorted({os.path.dirname(x) for x in p})
res = open(f"{OUT}/g-go-fix.jsonl", "a")
def run(tc, pkg, names, tags, race=False):
    argv = ["go", "test", "-count=1", "-v"] + (["-race"] if race else []) + ["-tags", tags, pkg, "-run", "|".join(f"^{n}$" if "/" not in n else n for n in names)]
    t = time.time(); p = subprocess.run(argv, cwd=BE, env=env, capture_output=True, text=True, timeout=900); out = p.stdout + p.stderr
    miss = [n for n in names if not re.search(rf"--- PASS: {re.escape(n)}\b", out)]; skip = [n for n in names if re.search(rf"--- SKIP: {re.escape(n)}\b", out)]
    open(f"{OUT}/{re.sub(r'[^A-Za-z0-9]+','_',tc)}.fix.log", "a").write(f"$ {' '.join(argv)}\n{out[-8000:]}\nrc={p.returncode}\n")
    r = dict(tc=tc, pkg=pkg, names=names, tags=tags, rc=p.returncode, missing=miss, skipped=skip, secs=int(time.time() - t)); res.write(json.dumps(r, ensure_ascii=False) + "\n"); res.flush(); print(r)
for tc, rs in rows.items():
    for r in rs:
        if "contract" in r["cmd"] and r["rc"]:
            run(tc, "./internal/contract/...", [], "testroutes,integration") if False else None
        for n in r["missing"]:
            pk = find_pkg(n.split("/")[0])
            if not pk: res.write(json.dumps(dict(tc=tc, names=[n], absent=True)) + "\n"); res.flush(); print("ABSENT", tc, n); continue
            for p in pk: run(tc, p, [n.split("/")[0]], "integration,testroutes")
