# gộp logs/s8core-results.jsonl (bản ghi cuối mỗi TC thắng) → part-s8core.tsv; chỉ xuất TC có trong tc-list.txt
import json, re
E = "/Users/kuro/Documents/TA_Agent_qc2/docs/regression/evidence-8"
ids = [l.strip() for l in open(E + "/tc-list.txt") if re.match(r"TC-P6-0[1-4]-", l.strip())]
last = {}
for l in open(E + "/logs/s8core-results.jsonl"):
    d = json.loads(l); last[d["tc"]] = d
out = [f"{t}\t{last[t]['v']}\t{last[t]['note']}" for t in ids if t in last]
open(E + "/part-s8core.tsv", "w").write("\n".join(out) + "\n")
from collections import Counter
c = Counter(last[t]["v"] if last[t]["v"] != "K" else "K" for t in ids if t in last)
print(len(out), "/", len(ids), dict(c), "chưa:", [t for t in ids if t not in last][:200])
