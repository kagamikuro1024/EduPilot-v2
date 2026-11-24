#!/usr/bin/env python3
"""E3 — chấm tự luận (US-P10-10 AC5). Bộ 60 bài BT-A của KC106 (benchmarks/guidance/kc, sinh một lần qua 9router, có sha256 dưới đây).

  e3.py setup                 nạp lớp KC106 + bài nộp bằng benchmarks/guidance/kc/load.py (API thật), TẮT cờ GRADING_GUIDANCE (đo chấm trần, không lưu ý)
  e3.py grade --tag P1r1 --first     «Chấm tất cả» BT-A (vòng 1), chờ xong, chụp điểm AI từng tiêu chí → raw/e3-<tag>.json
  e3.py grade --tag P1r2             chấm lại cả 60 bài (round + 1), chờ xong, chụp
  e3.py report                đọc raw/e3-*.json (+ benchmarks/eval/human/e3_scores.csv nếu chủ dự án đã chấm) → e3.json / e3.md / e3.svg + gói chấm tay

Chuẩn là điểm NGƯỜI chấm. Chưa có → `WAITING_HUMAN` (Q-E-18: điểm LLM không làm chuẩn — vòng tròn). Số tạm chỉ là «độ ổn định», không phải «độ chính xác».
"""
import argparse
import csv
import hashlib
import json
import math
import os
import random
import subprocess
import sys
import time

import lib
from lib import Fail

os.environ.setdefault("API_URL", lib.API)
KC = lib.ROOT / "benchmarks/guidance/kc"
sys.path.insert(0, str(KC))
HUMAN = lib.HERE / "human/e3_scores.csv"
N = 60


def kc():
    import common  # noqa: PLC0415 — đọc API_URL / mật khẩu lúc nạp, sau khi env đã đặt
    import measure  # noqa: PLC0415
    return common, measure


def dataset_digest():
    h = hashlib.sha256()
    for p in sorted((KC / "data/bt-a").glob("*.txt")):
        h.update(p.name.encode() + b"\0" + p.read_bytes())
    return h.hexdigest()


def setup():
    r = subprocess.run([sys.executable, str(KC / "load.py")], env={**os.environ, "API_URL": lib.API})
    if r.returncode:
        raise Fail(f"load.py thoát {r.returncode}")
    common, _ = kc()
    tok, _ = common.login("teacher")
    course = common.find_course(tok)
    _, flags = common.call("GET", f"/courses/{course}/guidance-flags", tok)
    cur = next(f for f in flags["items"] if f["feature"] == "GRADING_GUIDANCE")
    if cur["enabled"]:
        common.call("PUT", f"/courses/{course}/guidance-flags/GRADING_GUIDANCE", tok, {"enabled": False, "version": cur["version"]})
    print("E3: lớp KC106 sẵn sàng, cờ GRADING_GUIDANCE tắt")


def snapshot(common, measure, tok, course, asg):
    out = {}
    for code, it in measure.queue(tok, course, asg).items():
        v = measure.detail(tok, course, it["result"]["id"])
        r = v["result"]
        out[measure.key_of(code)] = {"sid": it["id"], "round": r["round"], "run_state": r["run_state"], "status": r["status"],
                                     "ai": [c["ai_score"] for c in v["criteria"]]}
    return out


def grade(tag, first):
    common, measure = kc()
    tok, _ = common.login("teacher")
    course = common.find_course(tok)
    asg = common.find_assignment(tok, course, "bt-a")["id"]
    before = snapshot(common, measure, tok, course, asg)
    t0 = lib.now_iso()
    if first:
        st, b = lib.call("POST", f"/courses/{course}/assignments/{asg}/grade-all", tok, {}, idem=f"e3-gradeall-{tag}")
        print(f"grade-all → {st} {b}")
        target = {k: 0 for k in before}
    else:
        target = {k: v["round"] for k, v in before.items()}
        for k, v in sorted(before.items()):
            st, b = lib.call("POST", f"/courses/{course}/submissions/{v['sid']}/grade", common.login("teacher")[0], {}, idem=f"e3-{tag}-{v['sid']}")
            if st not in (200, 202):
                print(f"  {k}: chấm lại → {st} {(b or {}).get('code')}")
    end = time.time() + 5400
    while time.time() < end:
        time.sleep(15)
        tok, _ = common.login("teacher")
        now = snapshot(common, measure, tok, course, asg)
        done = [k for k, v in now.items() if v["round"] > target.get(k, 0) and v["run_state"] in ("DONE", "FAILED")]
        print(f"  {tag}: {len(done)}/{len(now)} xong")
        if len(done) >= len(now) >= N:
            break
    else:
        raise Fail(f"{tag}: chưa xong sau 90 phút")
    t1 = lib.now_iso()
    lib.save_raw(f"e3-{tag}.json", {"tag": tag, "course_id": course, "window": [t0, t1], "items": now,
                                     "audit": lib.audit(t0, t1, task="GRADING", course=course)})
    print(f"→ raw/e3-{tag}.json")


def retry_failed(tag, gap=20):
    """Bài FAILED của lượt `tag` (nhà cung cấp quá tải → cầu dao mở): chấm lại TỪNG bài một, chờ xong mới sang bài sau; gộp vào raw/e3-<tag>.json, ghi số lần thử lại."""
    common, measure = kc()
    raw = lib.load_raw(f"e3-{tag}.json")
    tok, _ = common.login("teacher")
    course, asg = raw["course_id"], common.find_assignment(tok, raw["course_id"], "bt-a")["id"]
    failed = sorted(k for k, v in raw["items"].items() if v["run_state"] == "FAILED")
    t0 = lib.now_iso()
    print(f"{tag}: chấm lại tuần tự {len(failed)} bài FAILED")
    for k in failed:
        v = raw["items"][k]
        lib.call("POST", f"/courses/{course}/submissions/{v['sid']}/grade", common.login("teacher")[0], {}, idem=f"e3-{tag}-retry-{v['sid']}")
        end = time.time() + 600
        while time.time() < end:
            time.sleep(5)
            it = measure.queue(common.login("teacher")[0], course, asg).get(f"KC106{k[2:]}")
            r = measure.detail(common.login("teacher")[0], course, it["result"]["id"])
            if r["result"]["round"] > v["round"] and r["result"]["run_state"] in ("DONE", "FAILED"):
                raw["items"][k] = {"sid": v["sid"], "round": r["result"]["round"], "run_state": r["result"]["run_state"], "status": r["result"]["status"],
                                   "ai": [c["ai_score"] for c in r["criteria"]], "retried": True}
                break
        print(f"  {k}: {raw['items'][k]['run_state']}")
        lib.save_raw(f"e3-{tag}.json", raw)
        time.sleep(gap)
    t1 = lib.now_iso()
    raw["window_retry"] = [t0, t1]
    raw["audit"] = (raw.get("audit") or []) + lib.audit(t0, t1, task="GRADING", course=course)
    lib.save_raw(f"e3-{tag}.json", raw)


# ---- số (thuần) ---------------------------------------------------------------------------------------------------------------
def qwk(a, b, lo=0.0, hi=10.0, step=0.5):
    """Cohen kappa trọng số bậc hai trên điểm tổng làm tròn về bậc 0,5 (21 hạng 0…10)."""
    k = int(round((hi - lo) / step)) + 1
    ia = [min(k - 1, max(0, int(round((x - lo) / step)))) for x in a]
    ib = [min(k - 1, max(0, int(round((x - lo) / step)))) for x in b]
    n = len(ia)
    obs = [[0] * k for _ in range(k)]
    for x, y in zip(ia, ib):
        obs[x][y] += 1
    ha, hb = [sum(r) for r in obs], [sum(obs[i][j] for i in range(k)) for j in range(k)]
    num = sum(((i - j) ** 2) * obs[i][j] for i in range(k) for j in range(k))
    den = sum(((i - j) ** 2) * ha[i] * hb[j] / n for i in range(k) for j in range(k))
    return round(1 - num / den, 4) if den else None


def pearson(a, b):
    n = len(a)
    ma, mb = sum(a) / n, sum(b) / n
    sa, sb = math.sqrt(sum((x - ma) ** 2 for x in a)), math.sqrt(sum((y - mb) ** 2 for y in b))
    return round(sum((x - ma) * (y - mb) for x, y in zip(a, b)) / (sa * sb), 4) if sa and sb else None


def ranks(xs):
    order = sorted(range(len(xs)), key=lambda i: xs[i])
    r = [0.0] * len(xs)
    i = 0
    while i < len(order):
        j = i
        while j + 1 < len(order) and xs[order[j + 1]] == xs[order[i]]:
            j += 1
        for m in range(i, j + 1):
            r[order[m]] = (i + j) / 2 + 1
        i = j + 1
    return r


def spearman(a, b):
    return pearson(ranks(a), ranks(b))


def mae(a, b):
    return round(sum(abs(x - y) for x, y in zip(a, b)) / len(a), 4)


def pair(a, b, name_a, name_b):
    keys = sorted(set(a) & set(b))
    xa, xb = [a[k] for k in keys], [b[k] for k in keys]
    return {"a": name_a, "b": name_b, "n": len(keys), "qwk": qwk(xa, xb), "mae": mae(xa, xb) if keys else None, "pearson": pearson(xa, xb) if keys else None}


def totals(raw):
    return {k: sum(float(s) for s in v["ai"]) for k, v in raw["items"].items() if v["run_state"] == "DONE" and v["ai"] and None not in v["ai"]}


def human_pack():
    """Gói chấm tay cho chủ dự án: thứ tự xáo cố định, KHÔNG có điểm AI (tránh neo)."""
    keys = [f"kc{i:03d}" for i in range(1, N + 1)]
    random.Random(1011).shuffle(keys)
    rub = json.loads((KC / "data/rubric.json").read_text(encoding="utf-8"))["rubric"]
    p = lib.OUT / "e3-cham-tay.csv"
    with p.open("w", newline="", encoding="utf-8") as f:
        w = csv.writer(f)
        w.writerow(["stt", "ma_bai", "tep_bai"] + [f"c{i + 1} {c['name']} (0–{c['max_score']})" for i, c in enumerate(rub)] + ["ghi_chu"])
        for i, k in enumerate(keys, 1):
            w.writerow([i, k, f"benchmarks/guidance/kc/data/bt-a/KC106{k[2:]}.txt", "", "", "", "", ""])
    return p.name


def read_human():
    if not HUMAN.exists():
        return None
    out = {}
    with HUMAN.open(encoding="utf-8") as f:
        for row in csv.reader(f):
            if not row or not row[1].startswith("kc"):
                continue
            try:
                out[row[1]] = sum(float(x.replace(",", ".")) for x in row[3:7])
            except ValueError:
                continue  # dòng chưa chấm
    return out


def report():
    raws = {p.stem[3:]: lib.load_raw(p.name) for p in sorted(lib.RAW.glob("e3-*.json"))}
    if not raws:
        raise Fail("chưa có raw/e3-*.json")
    prov = json.loads((lib.RAW / "providers.json").read_text()) if (lib.RAW / "providers.json").exists() else {}
    tot = {t: totals(r) for t, r in raws.items()}
    profile = {json.loads(x)["key"]: json.loads(x)["total"] for x in (KC / "data/profile.jsonl").read_text().splitlines() if '"bt-a"' in x}
    sim = {}
    with (KC / "data/labels.csv").open(encoding="utf-8") as f:
        for row in csv.DictReader(f):
            if row["bt"] == "bt-a":
                sim[row["code"]] = float(row["total"])
    human = read_human()
    tags = sorted(tot)
    stab = []
    if "P1r1" in tot and "P1r2" in tot:
        stab.append({**pair(tot["P1r1"], tot["P1r2"], "P1r1", "P1r2"), "kind": "test–retest (cùng model, 2 vòng)"})
    for t in tags:
        if t.startswith("P2") and "P1r1" in tot:
            stab.append({**pair(tot["P1r1"], tot[t], "P1r1", t), "kind": "đồng thuận giữa hai model"})
    design = [{"tag": t, "n": len(set(tot[t]) & set(profile)), "spearman": spearman([tot[t][k] for k in sorted(set(tot[t]) & set(profile))], [profile[k] for k in sorted(set(tot[t]) & set(profile))])} for t in tags]
    appendix = [{**pair(tot[t], sim, t, "nhãn GV mô phỏng KC106"), "kind": "PHỤ LỤC — nhãn do agent đóng vai giảng viên, KHÔNG phải người chấm"} for t in tags]
    vs_human = None
    if human and len(human) >= N:
        vs_human = [{**pair(tot[t], human, t, "người chấm"), "kind": "độ chính xác so với người"} for t in tags]
    status = "OK" if vs_human else "WAITING_HUMAN"
    pack = human_pack()
    calls = {t: {s: sum(1 for a in r.get("audit") or [] if a["status"] == s) for s in sorted({a["status"] for a in r.get("audit") or []})} for t, r in raws.items()}
    lat = {t: lib.pct([a["latency_ms"] for a in r.get("audit") or [] if a["status"] == "ok"]) for t, r in raws.items()}
    obj = {"meta": lib.meta({"stack": "ep-qc", "providers": prov, "dataset": "benchmarks/guidance/kc/data/bt-a (60 bài)", "dataset_sha256": dataset_digest(), "guidance_flag": "tắt"}),
           "status": status, "reason": None if vs_human else "Chưa có điểm người chấm ≥ 60 bài (Q-E-18). Điểm LLM không được dùng làm chuẩn (vòng tròn). Gói chấm tay: " + pack + ".",
           "human_vs_ai": vs_human, "stability": stab, "ai_vs_design_tier": design, "appendix_simulated_teacher": appendix,
           "n_scored": {t: len(v) for t, v in tot.items()}, "llm_calls": calls, "latency_ms": lat,
           "unavailable": {t: r["retry_note"] for t, r in raws.items() if r.get("retry_note")}}
    grp = [(s["kind"].split(" (")[0] + f" {s['a']}–{s['b']}", [s["qwk"], s["pearson"]]) for s in stab] + [(f"tầng thiết kế {d['tag']}", [None, d["spearman"]]) for d in design]
    if grp:
        lib.svg_bars("e3.svg", grp, ["QWK", "Pearson / Spearman"], "E3 — độ ổn định của điểm AI (KHÔNG phải độ chính xác)", "hệ số", ymax=1.0)
    lib.write_report("e3", obj, md(obj, prov, pack))


def f(x, d=3):
    return "–" if x is None else f"{x:.{d}f}".replace(".", ",")


def md(o, prov, pack):
    L = ["# E3 — chấm tự luận", "", f"Trạng thái: **{o['status']}**. Stack `ep-qc` (sha bản gateway không ghi lúc đo); báo cáo dựng bằng bộ đo @ `{o['meta']['commit']}`, {o['meta']['timestamp'][:10]}. "
         f"Bộ bài: 60 bài BT-A của KC106 (sha256 `{o['meta']['config']['dataset_sha256'][:16]}…`), rubric 4 tiêu chí × 2,5, cờ lưu ý chấm tắt.", ""]
    if o["status"] == "WAITING_HUMAN":
        L += [f"**Chưa có điểm người chấm.** {o['reason']} Khi chủ dự án điền `benchmarks/eval/human/e3_scores.csv` (cùng cột với `{pack}`), chạy `make eval-report`: QWK / MAE / Pearson tính lại ngay, không gọi LLM.", ""]
    else:
        L += ["## So với người chấm", "", "| model | n | QWK | MAE | Pearson |", "| --- | --- | --- | --- | --- |"]
        L += [f"| {s['a']} | {s['n']} | {f(s['qwk'])} | {f(s['mae'])} | {f(s['pearson'])} |" for s in o["human_vs_ai"]]
        L.append("")
    L += ["## Độ ổn định (không phải độ chính xác)", "", "| phép so | n | QWK | MAE | Pearson |", "| --- | --- | --- | --- | --- |"]
    L += [f"| {s['kind']}: {s['a']} – {s['b']} | {s['n']} | {f(s['qwk'])} | {f(s['mae'])} | {f(s['pearson'])} |" for s in o["stability"]]
    L += ["", "| model | n | Spearman điểm AI ~ hồ sơ thiết kế lúc sinh bài |", "| --- | --- | --- |"]
    L += [f"| {d['tag']} | {d['n']} | {f(d['spearman'])} |" for d in o["ai_vs_design_tier"]]
    L += ["", "![](e3.svg)", "", "Lượt chấm (llm_audit, task GRADING; độ trễ chỉ tính lời gọi `ok`): " + "; ".join(f"{t}: " + ", ".join(f"{s} {n}" for s, n in o['llm_calls'][t].items()) + f" — p50 {o['latency_ms'][t].get('p50')} ms, p95 {o['latency_ms'][t].get('p95')} ms" for t in o["llm_calls"]) + ".",
          "Số bài chấm xong / 60: " + ", ".join(f"{t} {n}" for t, n in o["n_scored"].items()) + ". Lỗi `NETWORK` / `BAD_RESPONSE` / `SERVER` từ 9router làm cầu dao mở (`circuit_open`); P1 tự thử lại tới đủ 60 bài.",
          *[f"**{t} không đủ bài (AC6):** {note}" for t, note in o["unavailable"].items()], "",
          "Model: " + ", ".join(f"{k} = `{v.get('model')}`" for k, v in prov.items() if k.startswith("P")) + " (qua 9router).", "",
          "## Phụ lục — nhãn giảng viên MÔ PHỎNG (KC106, 40 bài kc021–kc060)", "",
          "Nhãn do agent đóng vai giảng viên tạo ở sprint 10.6 (`benchmarks/guidance/kc/data/labels.csv`). **Không phải điểm người chấm**, không thay AC5; chỉ để đối chiếu.", "",
          "| model | n | QWK | MAE | Pearson |", "| --- | --- | --- | --- | --- |"]
    L += [f"| {s['a']} | {s['n']} | {f(s['qwk'])} | {f(s['mae'])} | {f(s['pearson'])} |" for s in o["appendix_simulated_teacher"]]
    L += ["", "## Giới hạn", "",
          "- Dữ liệu mô phỏng (D44): 60 bài do `ag/gemini-3.8-flash-medium` sinh theo hồ sơ chất lượng; chưa kiểm chứng trên bài làm thật của sinh viên. Bài sinh và model chấm P1 cùng họ gemini-3.8-flash.",
          "- E3 chấm tay: **chủ dự án** (chưa thực hiện). Chưa có người chấm độc lập thứ hai nên chưa có mốc QWK người–người.",
          "- QWK tính trên điểm tổng làm tròn về bậc 0,5 (21 hạng). Spearman với «hồ sơ thiết kế» đo việc model tách được tầng chất lượng do người viết đề đặt, không phải điểm chấm.",
          "- Đính chính 9.5 (P-11-26): mẫu S7 của R3 (AI 10,0 so với nhãn giảng viên 8,00) là một điểm lệch lớn của chấm BT03, ghi lại ở đây; R3 «8,0 vs 4,5» là hàng seed bị tính nhầm, không phải AI bị dắt.",
          "- Số đo qua 9router (thuê bao): độ trễ gồm cả proxy; mỗi bài 2 lượt gọi (chấm hai lượt)."]
    return L


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    sp = ap.add_subparsers(dest="cmd", required=True)
    sp.add_parser("setup")
    g = sp.add_parser("grade")
    g.add_argument("--tag", required=True)
    g.add_argument("--first", action="store_true")
    g.add_argument("--retry-failed", action="store_true", help="chỉ chấm lại tuần tự các bài FAILED của lượt --tag")
    sp.add_parser("report")
    a = ap.parse_args()
    try:
        if a.cmd != "report":
            lib.refuse_demo()
            if not lib.ready():
                lib.die("Gateway chưa chạy")
        {"setup": setup, "report": report}.get(a.cmd, lambda: retry_failed(a.tag) if a.retry_failed else grade(a.tag, a.first))()
    except Fail as e:
        lib.die(f"LỖI: {e}", 2)


if __name__ == "__main__":
    main()
