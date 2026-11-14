#!/usr/bin/env python3
"""Chụp nháp, chấm lại và tính số đo của kiểm chứng «học từ lần sửa» (kiem-chung.md §2–§4). Chỉ thư viện chuẩn; điểm tính bằng decimal.Decimal.

  measure.py snap <nhãn> --dir <thư mục báo cáo>       chụp `drafts-<nhãn>.json` (b0r1 | b0r2 | b1r1 | b1r2 | bt-b-after): chỉ mã, round, guidance_version, status, criteria[], ai_scores[]; KHÔNG văn bản
  measure.py regrade <nhãn> [--limit N] [--bt bt-a]                 POST …/submissions/{id}/grade cho tập đo (BT-A kc021–kc060 + BT-B), chờ xong (round + 1, giữ nguyên bản lưu ý hiện hành)
  measure.py report --dir <thư mục báo cáo>             đọc labels.csv + drafts-*.json → metrics.json, report.md, report.svg + kết luận ĐẠT | KHÔNG ĐẠT | HARNESS_LỖI
  measure.py selftest                                   kiểm công thức trên dữ liệu tổng hợp
Biến: API_URL, SEED_DEFAULT_PASSWORD, KC_CLASS (KC106), KC_DATA. Thoát 1 khi HARNESS_LỖI hoặc KHÔNG ĐẠT, 0 khi ĐẠT.
"""
import argparse, csv, json, sys, time
from datetime import datetime, timezone
from decimal import Decimal as D
from pathlib import Path
from common import DATA, call, die, find_assignment, find_course, list_all, login, measure_keys

STATUSES = ["NONE", "RUNNING", "DRAFT", "REVIEWED", "PUBLISHED", "FAILED"]
Q, HALF = D("0.25"), D("0.5")


# ---- thu thập qua API ---------------------------------------------------------------------------------------------------
def queue(token, course, asg):
    out = {}
    for st in STATUSES:
        for it in list_all(f"/courses/{course}/submissions?assignment_id={asg}&match_status=MATCHED&status={st}", token):
            if it.get("student") and it.get("result"):
                out[it["student"]["student_code"]] = it
    return out


def key_of(code):
    return "kc" + code[-3:]


def detail(token, course, rid):
    return call("GET", f"/courses/{course}/grading-results/{rid}", token)[1]


def snap(token, course):
    items = []
    for bt in ("bt-a", "bt-b"):
        asg = find_assignment(token, course, bt)["id"]
        for code, it in sorted(queue(token, course, asg).items()):
            v = detail(token, course, it["result"]["id"])
            crit = v["criteria"]
            items.append({"bt": bt, "key": key_of(code), "round": v["result"]["round"], "guidance_version": (v.get("guidance") or {}).get("version", 0), "status": v["result"]["status"],
                          "run_state": v["result"]["run_state"], "criteria": [{"id": c["id"], "score": c["score"]} for c in crit], "ai_scores": [{"id": c["id"], "score": c["ai_score"]} for c in crit]})
    return items


def regrade(token, course, tag, limit, only=None):
    asg = {bt: find_assignment(token, course, bt)["id"] for bt in (only or ("bt-a", "bt-b"))}

    def current(bt):
        """submission id → (result id, round, run_state) hiện hành; mỗi lượt chấm lại là MỘT dòng kết quả mới (id mới)."""
        out = {}
        for code, it in queue(token, course, asg[bt]).items():
            if key_of(code) in set(measure_keys(bt)):
                r = detail(token, course, it["result"]["id"])["result"]
                out[it["id"]] = (r["round"], r["run_state"])
        return out

    before = {bt: current(bt) for bt in asg}
    todo = [(bt, sid) for bt in asg for sid in before[bt]][: limit or None]
    for _, sid in todo:
        call("POST", f"/courses/{course}/submissions/{sid}/grade", token, key=f"kc-regrade-{tag}-{sid}")
    end, t0 = time.time() + 3600, time.time()
    pending = list(todo)
    while pending and time.time() < end:
        time.sleep(5)
        now = {bt: current(bt) for bt in asg}
        pending = [(bt, sid) for bt, sid in pending if not (now[bt][sid][0] > before[bt][sid][0] and now[bt][sid][1] in ("DONE", "FAILED"))]
    if pending:
        die(f"Còn {len(pending)} bài chưa chấm xong sau 3600 s")
    print(f"regrade {tag}: {len(todo)} bài trong {int(time.time() - t0)} s")


# ---- số đo (thuần, không mạng) ------------------------------------------------------------------------------------------
def num(x):
    return D(x)


def scores(item, field):
    return [num(c["score"]) for c in item[field]]


def mean(xs):
    xs = list(xs)
    return sum(xs, D(0)) / len(xs) if xs else D(0)


def aggregate(ai, lab):
    """ai, lab: key → [4 điểm]. Trả MAE tiêu chí, MAE tổng, lệch TB có dấu (tổng và từng tiêu chí) — Δ = AI − giảng viên."""
    d = {k: [a - b for a, b in zip(ai[k], lab[k])] for k in lab}
    n = len(lab[next(iter(lab))])
    return {"n": len(d), "mae_criterion": mean(abs(x) for v in d.values() for x in v), "mae_total": mean(abs(sum(v, D(0))) for v in d.values()),
            "bias_total": mean(sum(v, D(0)) for v in d.values()), "bias_criterion": [mean(v[j] for v in d.values()) for j in range(n)]}


def avg(ai1, ai2):
    return {k: [(a + b) / 2 for a, b in zip(ai1[k], ai2[k])] for k in ai1}


def pick(items, bt, keys, field="ai_scores"):
    return {i["key"]: scores(i, field) for i in items if i["bt"] == bt and i["key"] in keys}


def compute(snaps, labels, order=None):
    """snaps: nhãn → danh sách mục; labels: bt → key → [4]. Trả (metrics, harness_errors)."""
    errs, m = [], {}
    ka, kb = set(labels.get("bt-a", {})), set(labels.get("bt-b", {}))
    for lab in ("b0r1", "b0r2", "b1r1", "b1r2", "bt-b-after"):
        if lab not in snaps:
            errs.append(f"thiếu ảnh chụp {lab}")
    if errs:
        return m, errs
    for lab, s in snaps.items():
        for bt, keys in (("bt-a", ka), ("bt-b", kb)):
            if lab == "bt-b-after" and bt == "bt-a":
                continue
            got = {i["key"]: i for i in s if i["bt"] == bt}
            miss = sorted(keys - set(got))
            if miss:
                errs.append(f"{lab}: {bt} thiếu {len(miss)} bài ({', '.join(miss[:3])}…)")
            for k in keys & set(got):
                i = got[k]
                if i["run_state"] != "DONE" or len(i["ai_scores"]) != 4 or any(c["score"] is None for c in i["ai_scores"]):
                    errs.append(f"{lab}: {bt} {k} chưa chấm xong (run_state={i['run_state']})")
    # A3 cách ly: BT-B giữ guidance_version 0 ở MỌI ảnh chụp; A-đo ở b0 là 0, ở b1 ≥ 1
    for lab, s in snaps.items():
        bad = [i["key"] for i in s if i["bt"] == "bt-b" and i["guidance_version"] != 0]
        if bad:
            errs.append(f"{lab}: BT-B có guidance_version ≠ 0 ({', '.join(bad[:3])}…) — lưu ý rò sang bài đối chứng")
    for lab, want in (("b0r1", lambda v: v == 0), ("b0r2", lambda v: v == 0), ("b1r1", lambda v: v >= 1), ("b1r2", lambda v: v >= 1)):
        bad = [i["key"] for i in snaps[lab] if i["bt"] == "bt-a" and i["key"] in ka and not want(i["guidance_version"])]
        if bad:
            errs.append(f"{lab}: BT-A đo có guidance_version không đúng kỳ vọng ({', '.join(bad[:3])}…)")
    # A2 tập học đã duyệt được miễn: ai_scores giữ nguyên, không chấm lại
    if order:
        b0 = {i["key"]: i for i in snaps["b0r1"] if i["bt"] == "bt-a"}
        b1 = {i["key"]: i for i in snaps["b1r1"] if i["bt"] == "bt-a"}
        for k in order["bt-a"]["learn"]:
            if k in b0 and k in b1 and (b0[k]["ai_scores"] != b1[k]["ai_scores"] or b0[k]["round"] != b1[k]["round"]):
                errs.append(f"bài học {k} bị chấm lại hoặc đổi ai_scores sau khi áp dụng lưu ý (round {b0[k]['round']} → {b1[k]['round']})")
    if errs:
        return m, errs
    la, lb = labels["bt-a"], labels["bt-b"]
    a = {lab: pick(snaps[lab], "bt-a", ka) for lab in ("b0r1", "b0r2", "b1r1", "b1r2")}
    b = {lab: pick(snaps[lab], "bt-b", kb) for lab in ("b0r1", "b0r2", "bt-b-after")}
    ag = {lab: aggregate(a[lab], la) for lab in a}
    bg = {lab: aggregate(b[lab], lb) for lab in b}
    eps = max(abs(ag["b0r1"]["mae_total"] - ag["b0r2"]["mae_total"]), abs(bg["b0r1"]["mae_total"] - bg["b0r2"]["mae_total"]))

    def side(x, y):
        return {"mae_criterion": mean([x["mae_criterion"], y["mae_criterion"]]), "mae_total": mean([x["mae_total"], y["mae_total"]]), "bias_total": mean([x["bias_total"], y["bias_total"]]),
                "bias_criterion": [mean([p, q]) for p, q in zip(x["bias_criterion"], y["bias_criterion"])]}
    b0, b1 = side(ag["b0r1"], ag["b0r2"]), side(ag["b1r1"], ag["b1r2"])
    m0, m1 = avg(a["b0r1"], a["b0r2"]), avg(a["b1r1"], a["b1r2"])
    wrong = sum(1 for k in la for j in range(4) if abs(m0[k][j] - la[k][j]) <= Q and abs(m1[k][j] - la[k][j]) > Q)
    noise = sum(1 for k in la for j in range(4) if abs(a["b0r1"][k][j] - la[k][j]) <= Q and abs(a["b0r2"][k][j] - la[k][j]) > Q)
    tot = lambda ai, k: abs(sum(ai[k], D(0)) - sum(la[k], D(0)))
    better = sum(1 for k in la if tot(m0, k) - tot(m1, k) >= Q)
    worse = sum(1 for k in la if tot(m1, k) - tot(m0, k) >= Q)
    fix0 = sum(1 for k in la for j in range(4) if abs(m0[k][j] - la[k][j]) >= HALF)
    fix1 = sum(1 for k in la for j in range(4) if abs(m1[k][j] - la[k][j]) >= HALF)
    iso = abs(bg["bt-b-after"]["mae_total"] - bg["b0r2"]["mae_total"])
    if iso > eps:
        errs.append(f"BT-B: MAE tổng sau bước 7 lệch {iso} > ε {eps}")
    checks = {
        "B1_mae_total": b1["mae_total"] <= D("0.8") * b0["mae_total"] and (b0["mae_total"] - b1["mae_total"]) > 2 * eps,
        "B2_bias": abs(b1["bias_total"]) <= abs(b0["bias_total"]) + D("0.20"),
        "B3_was_right_now_wrong": wrong <= (D("1.5") * noise if noise else 4),
        "B4_fix_count": fix1 < fix0,
    }
    m = {"n_a": len(la), "n_b": len(lb), "b0": b0, "b1": b1, "b0_runs": {k: ag[k] for k in ("b0r1", "b0r2")}, "b1_runs": {k: ag[k] for k in ("b1r1", "b1r2")}, "bt_b": bg, "epsilon": eps, "noise_items": noise,
         "was_right_now_wrong": wrong, "better": better, "worse": worse, "fix_b0": fix0, "fix_b1": fix1, "fix_total": len(la) * 4, "bt_b_isolation_diff": iso, "checks": checks,
         "verdict": "HARNESS_LỖI" if errs else ("ĐẠT" if all(checks.values()) else "KHÔNG ĐẠT")}
    return m, errs


# ---- đọc / ghi ---------------------------------------------------------------------------------------------------------
def read_labels(path):
    out = {"bt-a": {}, "bt-b": {}}
    for r in csv.DictReader(open(path, encoding="utf8")):
        out[r["bt"]][r["code"]] = [D(r[f"c{i}"]) for i in range(1, 5)]
    return out


def jsonable(x):
    if isinstance(x, dict):
        return {k: jsonable(v) for k, v in x.items()}
    if isinstance(x, (list, tuple)):
        return [jsonable(v) for v in x]
    return str(x) if isinstance(x, D) else x


def svg(m):
    rows = [("MAE tổng", m["b0"]["mae_total"], m["b1"]["mae_total"]), ("MAE tiêu chí", m["b0"]["mae_criterion"], m["b1"]["mae_criterion"]),
            ("|lệch TB| tổng", abs(m["b0"]["bias_total"]), abs(m["b1"]["bias_total"])), ("BT-B MAE tổng", m["bt_b"]["b0r2"]["mae_total"], m["bt_b"]["bt-b-after"]["mae_total"])]
    top = max([max(a, b) for _, a, b in rows] + [D("0.01")])
    out = ['<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 640 %d" font-family="sans-serif" font-size="13">' % (60 + 70 * len(rows)),
           '<text x="10" y="22" font-size="15" fill="currentColor">Trước (v0) / sau (v1): điểm AI − giảng viên, càng thấp càng tốt (BT-B: trước / sau bước 7)</text>']
    for n, (name, a, b) in enumerate(rows):
        y = 50 + n * 70
        out.append(f'<text x="10" y="{y + 14}" fill="currentColor">{name}</text>')
        for off, v, col, lab in ((0, a, "var(--muted)", "trước"), (24, b, "var(--accent)", "sau")):
            w = int(float(v / top) * 360)
            out.append(f'<rect x="150" y="{y + off}" width="{max(w, 1)}" height="20" fill="{col}"/><text x="{156 + w}" y="{y + off + 15}" fill="currentColor">{lab} {v:.3f}</text>')
    return "\n".join(out) + "\n</svg>\n"


def md(m, errs):
    f = lambda x: f"{x:.3f}"
    if errs:
        return "# Kiểm chứng «học từ lần sửa» — HARNESS_LỖI\n\n" + "\n".join(f"- {e}" for e in errs) + "\n"
    c = m["checks"]
    ok = lambda b: "đạt" if b else "KHÔNG"
    return f"""# Kiểm chứng «học từ lần sửa» — {m['verdict']}

Nguồn nhãn: `giaovien` (agent `claude-fable-5-2`, effort medium), chấm mù. Bằng chứng kỹ thuật cho harness, không thay nhãn người thật.

| Chỉ số (BT-A đo, n = {m['n_a']}) | B0 (v0, TB 2 lần) | B1 (v1, TB 2 lần) |
| --- | --- | --- |
| MAE tổng | {f(m['b0']['mae_total'])} | {f(m['b1']['mae_total'])} |
| MAE tiêu chí | {f(m['b0']['mae_criterion'])} | {f(m['b1']['mae_criterion'])} |
| Lệch TB có dấu, tổng (AI − GV) | {f(m['b0']['bias_total'])} | {f(m['b1']['bias_total'])} |
| Lệch TB theo tiêu chí | {', '.join(f(x) for x in m['b0']['bias_criterion'])} | {', '.join(f(x) for x in m['b1']['bias_criterion'])} |
| Tiêu chí phải sửa tay (Δ tuyệt đối ≥ 0,5) / {m['fix_total']} | {m['fix_b0']} | {m['fix_b1']} |

Nền nhiễu ε = {f(m['epsilon'])}. Vốn đúng nay sai: {m['was_right_now_wrong']} (nhiễu tự nhiên: {m['noise_items']}). Bài tốt lên / xấu đi (≥ 0,25): {m['better']} / {m['worse']}.

BT-B (đối chứng, n = {m['n_b']}): MAE tổng b0r1 {f(m['bt_b']['b0r1']['mae_total'])}, b0r2 {f(m['bt_b']['b0r2']['mae_total'])}, sau bước 7 {f(m['bt_b']['bt-b-after']['mae_total'])}; lệch cách ly {f(m['bt_b_isolation_diff'])}.

## Tiêu chí chất lượng (SRS §9)
1. MAE tổng B1 ≤ 0,8 × B0 và mức giảm > 2ε: {ok(c['B1_mae_total'])}
2. |lệch TB B1| ≤ |lệch TB B0| + 0,20: {ok(c['B2_bias'])}
3. Vốn đúng nay sai ≤ 1,5 × nhiễu (hoặc ≤ 4 nếu nhiễu = 0): {ok(c['B3_was_right_now_wrong'])}
4. Số tiêu chí phải sửa tay B1 < B0: {ok(c['B4_fix_count'])}

## Giới hạn
Bài mô phỏng; một model chấm, một model sinh; một giảng viên là agent; một bài tập học; n = {m['n_a']}. Số tốt lên / xấu đi, vốn đúng nay sai và số sửa tay tính trên điểm AI trung bình của hai lượt chấm.
"""


def report(d):
    d = Path(d)
    labels = read_labels(d / "labels.csv")
    snaps = {p.stem.removeprefix("drafts-"): json.loads(p.read_text())["items"] for p in d.glob("drafts-*.json")}
    order = json.loads((DATA / "order.json").read_text()) if (DATA / "order.json").exists() else None
    m, errs = compute(snaps, labels, order)
    (d / "metrics.json").write_text(json.dumps(jsonable({"errors": errs, **m}), ensure_ascii=False, indent=2) + "\n")
    (d / "report.md").write_text(md(m, errs), encoding="utf8")
    if m:
        (d / "report.svg").write_text(svg(m), encoding="utf8")
    print(m.get("verdict", "HARNESS_LỖI"))
    for e in errs:
        print("  -", e, file=sys.stderr)
    return 0 if m.get("verdict") == "ĐẠT" else 1


def selftest():
    def item(bt, k, ai, gv=0, rnd=1):
        return {"bt": bt, "key": k, "round": rnd, "guidance_version": gv, "status": "DRAFT", "run_state": "DONE", "criteria": [], "ai_scores": [{"id": f"c{j}", "score": s} for j, s in enumerate(ai)]}
    lab = {"bt-a": {"kc021": [D(2), D(2), D("1.5"), D(2)], "kc022": [D(1), D(1), D(1), D(1)]}, "bt-b": {"kc001": [D(2)] * 4}}
    # v0: AI gắt hơn 1,0 ở tiêu chí đầu; v1: sửa đúng. BT-B không đổi.
    a0 = {"kc021": ["1.00", "2.00", "1.50", "2.00"], "kc022": ["0.00", "1.00", "1.00", "1.00"]}
    a1 = {"kc021": ["2.00", "2.00", "1.50", "2.00"], "kc022": ["1.00", "1.00", "1.00", "1.00"]}
    mk = lambda src, gv, rnd: [item("bt-a", k, v, gv, rnd) for k, v in src.items()] + [item("bt-b", "kc001", ["2.00"] * 4)]
    snaps = {"b0r1": mk(a0, 0, 1), "b0r2": mk(a0, 0, 2), "b1r1": mk(a1, 1, 3), "b1r2": mk(a1, 1, 4), "bt-b-after": [item("bt-b", "kc001", ["2.00"] * 4)]}
    m, errs = compute(snaps, lab)
    assert not errs, errs
    assert m["b0"]["mae_total"] == D(1) and m["b1"]["mae_total"] == D(0) and m["b0"]["bias_total"] == D(-1), m
    assert m["epsilon"] == D(0) and m["fix_b0"] == 2 and m["fix_b1"] == 0 and m["verdict"] == "ĐẠT", m
    snaps["b1r1"][0]["guidance_version"] = 0
    assert any("guidance_version" in e for e in compute(snaps, lab)[1]), "phải bắt bản lưu ý sai"
    snaps["b1r1"][0]["guidance_version"] = 1
    snaps["b0r2"][-1]["guidance_version"] = 1
    assert any("rò sang bài đối chứng" in e for e in compute(snaps, lab)[1]), "phải bắt rò sang BT-B"
    svg(m), md(m, [])
    print("selftest ok")


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("cmd", choices=["snap", "regrade", "report", "selftest"])
    ap.add_argument("label", nargs="?")
    ap.add_argument("--dir")
    ap.add_argument("--limit", type=int, default=0)
    ap.add_argument("--bt", choices=["bt-a", "bt-b"], help="regrade: chỉ bài tập này (bước 7b chỉ chấm lại BT-A; BT-B phải giữ nguyên v0)")
    a = ap.parse_args()
    if a.cmd == "selftest":
        return selftest()
    if a.cmd == "report":
        sys.exit(report(a.dir or die("--dir")))
    if not a.label:
        die("thiếu nhãn")
    token, _ = login("teacher")
    course = find_course(token)
    if a.cmd == "snap":
        d = Path(a.dir or die("--dir"))
        d.mkdir(parents=True, exist_ok=True)
        items = snap(token, course)
        (d / f"drafts-{a.label}.json").write_text(json.dumps({"label": a.label, "taken_at": datetime.now(timezone.utc).isoformat(timespec="seconds"), "items": items}, indent=1) + "\n")
        print(f"snap {a.label}: {len(items)} mục")
    else:
        regrade(token, course, a.label, a.limit, (a.bt,) if a.bt else None)


if __name__ == "__main__":
    main()
