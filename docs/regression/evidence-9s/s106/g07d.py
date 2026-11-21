import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from g07c import regress, W
from gdlib import *
def run():
    stub("allow"); redis_del("ep:llm:cb:*")
    r0 = regress(["run", "--course", CG0, "--limit", "6"]); rep = json.loads(r0.stdout); print("lần đo", rep)
    mae = float(rep["mae"]); bias = float(rep["bias"]); out = {}
    for tag, b in (("a MAE +12%", {"n": 6, "mae": f"{mae/1.12:.2f}", "bias": rep["bias"], "old_mae": rep["old_mae"], "worse": 0, "skipped": 0}), ("b MAE +8%", {"n": 6, "mae": f"{mae/1.08:.2f}", "bias": rep["bias"], "old_mae": rep["old_mae"], "worse": 0, "skipped": 0}),
                   ("e lệch TB xấu thêm 0,25", {"n": 6, "mae": rep["mae"], "bias": f"{bias-0.25:.2f}", "old_mae": rep["old_mae"], "worse": 0, "skipped": 0}), ("f worse=5 → (ca xấu đi)", {"n": 6, "mae": rep["mae"], "bias": rep["bias"], "old_mae": rep["old_mae"], "worse": 0, "skipped": 0})):
        json.dump(b, open(W + "baseline_t.json", "w")); r = regress(["run", "--course", CG0, "--limit", "6", "--baseline", "/bl.json"], extra=("-v", W + "baseline_t.json:/bl.json:ro")); out[tag] = (r.returncode, r.stdout.strip().splitlines()[-1][:120], b["mae"])
        print(tag, out[tag])
    json.dump(out, open(W + "g07d.json", "w"), ensure_ascii=False)
with window(): run()
