#!/bin/bash
# DEF-R7-06/07 chạy lại sau 425396a7 — đọc-only trên worktree TA_Agent_qc3p (425396a7)
cd ~/Documents/TA_Agent_qc3p || exit 1
echo "HEAD: $(git log --oneline -1)"; echo "ancestor 7062b009 trong HEAD: $(git merge-base --is-ancestor 7062b009 HEAD && echo có)"
echo "--- TC-E4: shard rỗng N=200 (bước CI: mapfile + guard)"
cd frontend
# mapfile không có trong bash của macOS/engine cục bộ: chạy đúng đoạn bước CI trong debian bash 5 (stdin = đầu ra node)
step='mapfile -t files < <(cat); echo "số phần tử mảng files=${#files[@]}"; if [ "${#files[@]}" -eq 0 ]; then echo "(bước CI) shard rỗng: thoát 0, KHÔNG gọi playwright"; else echo "playwright sẽ chạy ${#files[@]} đối số; phần tử 0=<${files[0]}>"; fi'
for sh in "1 4" "4 4" "150 200" "200 200"; do echo "shard ${sh/ //}:"; node e2e/support/shard.mjs $sh | docker run -i --rm debian:trixie-slim bash -c "$step"; done
echo "--- shard thật 1/4..4/4 (mỗi tệp đúng 1 shard)"
for i in 1 2 3 4; do echo "shard $i/4: $(node e2e/support/shard.mjs $i 4 | wc -l) tệp"; done
for i in 1 2 3 4; do node e2e/support/shard.mjs $i 4; done | sort | uniq -d | wc -l | sed 's/^/tệp trùng giữa các shard: /'
echo "--- scripts/ci-guard.test.mjs"
cd ..; node --test scripts/ci-guard.test.mjs 2>&1 | grep -E "✔|✖|ℹ (tests|pass|fail)"
echo "--- TC-E2: timeout-minutes từng job @HEAD"
python3 - <<'P'
import re
t=open('.github/workflows/ci.yml').read().split('\njobs:')[1]
for m in re.finditer(r'^  ([a-z0-9-]+):\s*\n(.*?)(?=^  [a-z0-9-]+:\s*\n|\Z)',t,re.S|re.M):
    b=m.group(2); n=re.search(r'^    name: (.*)$',b,re.M); to=re.search(r'^    timeout-minutes: (\d+)',b,re.M)
    print(f"{m.group(1):16s} {n.group(1) if n else '':45s} timeout-minutes={to.group(1) if to else 'THIẾU'}")
P
