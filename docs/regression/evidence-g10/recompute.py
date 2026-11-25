#!/usr/bin/env python3
"""QC tính lại số đo TC-G10-10 từ dữ liệu thô (labels.csv + drafts-*.json), KHÔNG dùng benchmarks/guidance/kc/measure.py. So với out/guidance-before-after.json (sai số cho phép <= 0,005)."""
import csv, json
from decimal import Decimal as D
from pathlib import Path
H = Path(__file__).parent
lab = {(r["bt"], r["code"]): [D(r[f"c{i}"]) for i in range(1, 5)] for r in csv.DictReader(open(H / "labels.csv"))}
def snap(n):
    d = {}
    for it in json.load(open(H / "drafts" / f"drafts-{n}.json"))["items"]:
        if it["bt"] == "bt-a" and it["key"] <= "kc020":
            continue
        d[(it["bt"], it["key"])] = [D(str(x["score"])) for x in it["ai_scores"]]
    return d
S = {n: snap(n) for n in ("b0r1", "b0r2", "b1r1", "b1r2", "bt-b-after")}
A = sorted(k for k in lab if k[0] == "bt-a")
assert len(A) == 40 and all(k in S[n] for n in ("b0r1", "b0r2", "b1r1", "b1r2") for k in A)
def mae(run, keys):
    return sum(sum(abs(a - b) for a, b in zip(S[run][k], lab[k])) / 4 for k in keys) / len(keys)
def maet(run, keys):
    return sum(abs(sum(S[run][k]) - sum(lab[k])) for k in keys) / len(keys)
def bias(run, keys):
    return sum(sum(S[run][k]) - sum(lab[k]) for k in keys) / len(keys)
def avg(a, b):
    return (a + b) / 2
out = {}
for tag, (r1, r2) in {"b0": ("b0r1", "b0r2"), "b1": ("b1r1", "b1r2")}.items():
    out[tag] = {"mae_total": avg(maet(r1, A), maet(r2, A)), "mae_criterion": avg(mae(r1, A), mae(r2, A)), "bias_total": avg(bias(r1, A), bias(r2, A))}
def fixcount(run):
    return sum(1 for k in A for a, b in zip(S[run][k], lab[k]) if abs(a - b) >= D("0.5"))
out["fix_b0"] = avg(D(fixcount("b0r1")), D(fixcount("b0r2")))
out["fix_b1"] = avg(D(fixcount("b1r1")), D(fixcount("b1r2")))
drop = (out["b0"]["mae_total"] - out["b1"]["mae_total"]) / out["b0"]["mae_total"]
# nhiễu: tiêu chí đúng (|Δ| <= 0,25) ở b0r1 nhưng sai ở b0r2
Q = D("0.25")
noise = sum(1 for k in A for a, b, l in zip(S["b0r1"][k], S["b0r2"][k], lab[k]) if abs(a - l) <= Q and abs(b - l) > Q)
# vốn đúng nay sai: |Δ| <= 0,25 ở B0 (lượt 1) nhưng > 0,25 ở B1 (lượt 1)
wr = sum(1 for k in A for a, b, l in zip(S["b0r1"][k], S["b1r1"][k], lab[k]) if abs(a - l) <= Q and abs(b - l) > Q)
wr_avg = sum(1 for k in A for a0, a1, b0, b1, l in zip(S["b0r1"][k], S["b0r2"][k], S["b1r1"][k], S["b1r2"][k], lab[k]) if abs(avg(a0, a1) - l) <= Q and abs(avg(b0, b1) - l) > Q)
out.update(drop=drop, noise=noise, was_right_now_wrong_r1=wr, was_right_now_wrong_avg=wr_avg, thr=D("1.5") * noise)
m = json.load(open(H / "out" / "guidance-before-after.json"))["metrics"]
print("QC tính lại:", {k: (str(v) if not isinstance(v, dict) else {a: f"{b:.4f}" for a, b in v.items()}) for k, v in out.items()})
print("run.py:", {k: m[k] for k in ("noise_items", "was_right_now_wrong", "fix_b0", "fix_b1", "better", "worse")}, {t: {a: m[t][a][:7] for a in ("mae_total", "mae_criterion", "bias_total")} for t in ("b0", "b1")})
for t in ("b0", "b1"):
    for a in ("mae_total", "mae_criterion", "bias_total"):
        d = abs(out[t][a] - D(m[t][a]))
        print(f"{t}.{a}: QC {out[t][a]:.4f} vs run.py {D(m[t][a]):.4f} chênh {d:.5f} {'OK' if d <= D('0.005') else 'LỆCH'}")
# số tiêu chí phải sửa tay / tốt lên - xấu đi: tính trên điểm AI TRUNG BÌNH của hai lượt (đúng như ghi chú «Giới hạn» của báo cáo)
def av(a, b, k):
    return [avg(x, y) for x, y in zip(S[a][k], S[b][k])]
fx0 = sum(1 for k in A for a, l in zip(av("b0r1", "b0r2", k), lab[k]) if abs(a - l) >= D("0.5"))
fx1 = sum(1 for k in A for a, l in zip(av("b1r1", "b1r2", k), lab[k]) if abs(a - l) >= D("0.5"))
better = sum(1 for k in A if abs(sum(av("b0r1", "b0r2", k)) - sum(lab[k])) - abs(sum(av("b1r1", "b1r2", k)) - sum(lab[k])) >= Q)
worse = sum(1 for k in A if abs(sum(av("b1r1", "b1r2", k)) - sum(lab[k])) - abs(sum(av("b0r1", "b0r2", k)) - sum(lab[k])) >= Q)
print(f"trung bình hai lượt: sửa tay B0={fx0} B1={fx1} (run.py 74 / 62); tốt lên={better} xấu đi={worse} (run.py 37 / 2); vốn đúng nay sai={wr_avg} (run.py 9); nhiễu={noise} (run.py 5)")
print(f"ngưỡng 1,5 x nhiễu = {D('1.5')*noise}; vốn đúng nay sai {wr_avg} > {D('1.5')*noise} -> tiêu chí 3 {'ĐẠT' if wr_avg <= D('1.5')*noise else 'KHÔNG'}; MAE giảm {drop:.1%} (>= 20%: {'đạt' if drop >= D('0.2') else 'KHÔNG'}); |lệch B1| {abs(out['b1']['bias_total']):.4f} <= |lệch B0| + 0,20 = {abs(out['b0']['bias_total'])+D('0.2'):.4f}: {'đạt' if abs(out['b1']['bias_total']) <= abs(out['b0']['bias_total'])+D('0.2') else 'KHÔNG'}")
