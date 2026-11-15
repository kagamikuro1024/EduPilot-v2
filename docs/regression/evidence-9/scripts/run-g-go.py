#!/usr/bin/env python3
"""Chạy các lệnh `go test` ghi ở hàng -G của tc-US-P7-*.md (sprint 9). Mỗi lệnh chạy 1 lần (cache theo chuỗi lệnh), tuần tự.
Kết quả: logs/g-go.jsonl (mỗi dòng: tc, cmd, rc, named, missing, skipped, race, secs)."""
import re, glob, json, shlex, subprocess, os, time, sys
ROOT = os.path.expanduser("~/Documents/TA_Agent_qc2"); OUT = f"{ROOT}/docs/regression/evidence-9/logs"
env = dict(os.environ, DOCKER_HOST=f"unix://{os.path.expanduser('~')}/.colima/default/docker.sock", TESTCONTAINERS_RYUK_DISABLED="true")
rows = {}
for f in sorted(glob.glob(f"{ROOT}/docs/sprints/9/qc/tc-*.md")):
    for l in open(f):
        m = re.match(r"\| (TC-[\w-]+-G\d+) \|", l)
        if m: rows[m.group(1)] = [c.replace("\\|", "|") for c in re.findall(r"`([^`]*)`", l)]
only = set(sys.argv[1:])
cache = {}; res = open(f"{OUT}/g-go.jsonl", "a")
for tc, cmds in rows.items():
    if only and tc not in only: continue
    for c in cmds:
        c = re.sub(r"^cd backend-go && ", "", c)
        if not c.startswith("go test"): continue
        if c in cache: r = dict(cache[c], tc=tc); res.write(json.dumps(r, ensure_ascii=False) + "\n"); res.flush(); continue
        argv = shlex.split(c); names = []
        if "-run" in argv:
            v = argv[argv.index("-run") + 1]; names = [n.strip("^$") for n in v.split("|")]
        argv = argv[:2] + ["-count=1"] + argv[2:] + ([] if "-v" in argv else ["-v"])
        t = time.time()
        try:
            p = subprocess.run(argv, cwd=f"{ROOT}/backend-go", env=env, capture_output=True, text=True, timeout=900); out = p.stdout + p.stderr; rc = p.returncode
        except subprocess.TimeoutExpired as e:
            out = (e.stdout or b"").decode() if isinstance(e.stdout, bytes) else (e.stdout or ""); rc = 124
        lg = re.sub(r"[^\w]+", "_", tc) + ".log"; open(f"{OUT}/{lg}", "a").write(f"$ {c}\n{out[-20000:]}\nrc={rc}\n")
        missing, skipped = [], []
        for n in names:
            base = n.split("/")[0]
            if re.search(rf"--- SKIP: {re.escape(n)}\b", out): skipped.append(n)
            elif not re.search(rf"--- PASS: {re.escape(n)}\b", out): missing.append(n)
        r = dict(tc=tc, cmd=c, rc=rc, named=len(names), missing=missing, skipped=skipped, race="DATA RACE" in out, secs=int(time.time() - t), nopkg=bool(re.search(r"no Go files|cannot find package|no test files", out)))
        cache[c] = r; res.write(json.dumps(r, ensure_ascii=False) + "\n"); res.flush()
