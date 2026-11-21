#!/usr/bin/env python3
"""E1 — tường lửa PII (US-P10-10 AC3) bằng `benchmarks/eval_pii.py` (precheck của Threads, token Sinh viên) + quét rò rỉ payload LLM.

  e1.py run --tag fake-a     một lượt eval_pii (200 mẫu) → raw/e1-<tag>/e1.json (+ ghi cấu hình nhúng lúc chạy)
  e1.py leak                 quét `llm_request_payloads` (bản đã che, sau mọi lượt E2–E5) tìm tên / MSSV / email roster seed → raw/e1-leak.json (chỉ đếm)
  e1.py report               → e1.json / e1.md / e1.svg
"""
import argparse
import json
import os
import re
import unicodedata
import subprocess
import sys

import lib
from lib import Fail

ROSTER = lib.ROOT / "benchmarks/pii/roster_seed.json"


def run(tag):
    out = lib.RAW / f"e1-{tag}"
    out.mkdir(parents=True, exist_ok=True)
    prov = json.loads((lib.RAW / "providers.json").read_text()) if (lib.RAW / "providers.json").exists() else {}
    t0 = lib.now_iso()
    r = subprocess.run([sys.executable, str(lib.ROOT / "benchmarks/eval_pii.py"), "--api", lib.API, "--out-dir", str(out)], env={**os.environ, "E1_INSECURE": "1"})
    if r.returncode == 2 or not (out / "e1.json").exists():
        raise Fail(f"eval_pii.py lỗi (mã {r.returncode})")
    lib.save_raw(f"e1-{tag}/run.json", {"tag": tag, "embedding": prov.get("embedding", "giả (fake)"), "window": [t0, lib.now_iso()], "exit": r.returncode})


def leak():
    roster = json.loads(ROSTER.read_text(encoding="utf-8"))
    def fold(s):
        s = s.replace("đ", "d").replace("Đ", "D")
        return "".join(c for c in unicodedata.normalize("NFD", s) if not unicodedata.combining(c)).lower()
    kinds = {"name": [r["name"] for r in roster], "mssv": [r["code"] for r in roster], "email": [r["email"] for r in roster]}
    probes = sum(len(v) for v in kinds.values())
    cols = ", ".join(f"count(*) filter (where f ~ {lib.lit('|'.join(re.escape(fold(v)) for v in vs))}) as {k}" for k, vs in kinds.items())
    row = lib.sql(f"select count(*) as rows, {cols} from (select lower(vn_fold(masked_prompt)) as f from llm_request_payloads) s")[0]
    hits = {k: row[k] for k in kinds}  # số payload trúng ít nhất một mẫu của loại đó
    by_task = lib.sql("select a.task, count(*) as n from llm_request_payloads p join llm_audit a on a.id = p.audit_id group by a.task order by 2 desc")
    lib.save_raw("e1-leak.json", {"at": lib.now_iso(), "payload_rows": row["rows"], "by_task": by_task, "hits": hits, "probes": probes})
    print(f"payload: {row['rows']} dòng, trúng {hits}")


def report():
    runs = {}
    for d in sorted(lib.RAW.glob("e1-*/")):
        if (d / "e1.json").exists():
            runs[d.name[3:]] = {**json.loads((d / "e1.json").read_text()), "run": json.loads((d / "run.json").read_text())}
    if not runs:
        raise Fail("chưa có raw/e1-*/e1.json")
    lk = lib.load_raw("e1-leak.json")
    fakes = [t for t in runs if t.startswith("fake")]
    repro = None
    if len(fakes) >= 2:
        repro = {"tags": fakes[:2], "identical_counts": runs[fakes[0]]["counts"] == runs[fakes[1]]["counts"], "tolerance": "bằng nhau tuyệt đối với nhúng giả (Q-S11-13)"}
    main_tag = "real" if "real" in runs else fakes[0]
    m = runs[main_tag]
    leak_total = sum(lk["hits"].values()) if lk else None
    obj = {"meta": lib.meta({"stack": "ep-qc", "dataset": "benchmarks/pii/e1_dataset.jsonl (200)", "runs": {t: r["run"]["embedding"] for t, r in runs.items()}}, runs=len(runs)),
           "status": "OK", "labels": (lib.human_review("e1") or {}).get("status", "WAITING_HUMAN"), "human_review": lib.human_review("e1"),
           "labels_note": "200 nhãn do agent viết (benchmarks/pii/LABELING.md); chủ dự án gán mù ≥ 50 mẫu phân tầng (P-11-3): gói e1-soat-nhan.csv → benchmarks/eval/human/e1_soat.csv.",
           "main_run": main_tag, "channel": {**m["metrics"], **m["counts"]}, "targets": {"f1_channel": ">= 0.90 (mục tiêu)", "payload_leak": "= 0 (mục tiêu)"},
           "by_pii_type_recall": m["by_pii_type"], "by_stratum": m["by_stratum"], "channel_confusion": m["channel_confusion"],
           "payload_leak": lk and {"rows_scanned": lk["payload_rows"], "hits": lk["hits"], "total": leak_total, "by_task": lk["by_task"]},
           "runs": {t: {"metrics": r["metrics"], "counts": r["counts"], "embedding": r["run"]["embedding"]} for t, r in runs.items()}, "reproducibility": repro}
    lib.svg_bars("e1.svg", [(t, [r["recall"]]) for t, r in sorted(m["by_pii_type"].items())] + [("kênh F1", [m["metrics"]["f1"]]), ("chặn nhầm", [m["metrics"]["false_block"]])],
                 ["recall / F1 / tỉ lệ"], f"E1 — tường lửa PII ({runs[main_tag]['run']['embedding'][:40]})", "tỉ lệ", ymax=1.0)
    f = lambda x: f"{x:.3f}".replace(".", ",")  # noqa: E731
    c = obj["channel"]
    L = ["# E1 — tường lửa PII của Threads", "", f"Trạng thái: **OK** · nhãn: **{obj['labels']}** (nhãn máy; chủ dự án gán mù ≥ 50 mẫu — P-11-3, `HUONG-DAN-CHU-DU-AN.md`). Stack `ep-qc` (sha bản gateway không ghi lúc đo); báo cáo dựng bằng bộ đo @ `{obj['meta']['commit']}`, {obj['meta']['timestamp'][:10]}.", "",
         f"Lượt chính `{main_tag}` ({runs[main_tag]['run']['embedding']}): kênh **P {f(c['precision'])} · R {f(c['recall'])} · F1 {f(c['f1'])}** (mục tiêu F1 ≥ 0,90), chặn nhầm {f(c['false_block'])}; TP {c['tp']} · FP {c['fp']} · FN {c['fn']} · TN {c['tn']}.",
         f"Rò rỉ payload LLM: **{leak_total if leak_total is not None else 'chưa quét'}** lần trúng tên / MSSV / email roster trên {lk['payload_rows'] if lk else '–'} payload đã che (mục tiêu = 0).", "",
         "![](e1.svg)", "", "## Theo lượt", "", "| lượt | nhúng | P | R | F1 | chặn nhầm | TP | FP | FN | TN |", "| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |"]
    L += [f"| {t} | {r['embedding']} | {f(r['metrics']['precision'])} | {f(r['metrics']['recall'])} | {f(r['metrics']['f1'])} | {f(r['metrics']['false_block'])} | {r['counts']['tp']} | {r['counts']['fp']} | {r['counts']['fn']} | {r['counts']['tn']} |" for t, r in obj["runs"].items()]
    L += ["", "## Recall theo loại PII (lượt chính)", "", "| loại | recall | TP | FN |", "| --- | --- | --- | --- |"]
    L += [f"| {t} | {f(v['recall'])} | {v['tp']} | {v['fn']} |" for t, v in sorted(m["by_pii_type"].items())]
    L += ["", "Precision theo loại không định nghĩa được: mẫu âm không mang loại PII, tường lửa chỉ trả chặn / không chặn (precision chỉ có ở mức kênh).", "",
          "## Theo nhóm", "", "| nhóm | TP | FN | FP | TN | recall / chặn nhầm |", "| --- | --- | --- | --- | --- | --- |"]
    L += [f"| {s} | {v['tp']} | {v['fn']} | {v['fp']} | {v['tn']} | {f(v.get('recall', v.get('false_block')))} |" for s, v in m["by_stratum"].items()]
    if lk:
        L += ["", "## Rò rỉ payload", "", f"Quét {lk['payload_rows']} dòng `llm_request_payloads` (bản đã che, gồm mọi lời gọi của E2–E5) với {lk['probes']} mẫu (tên, MSSV, email của 13 sinh viên roster seed; so không dấu): " + ", ".join(f"{k} {v}" for k, v in lk["hits"].items()) + ".",
              "Theo tác vụ: " + ", ".join(f"{x['task']} {x['n']}" for x in lk["by_task"]) + "."]
    if repro:
        L += ["", f"Tái lập (AC11): hai lượt nhúng giả `{repro['tags'][0]}`, `{repro['tags'][1]}` → TP/FP/FN/TN **{'bằng nhau' if repro['identical_counts'] else 'KHÁC nhau'}** (sai số cho phép: {repro['tolerance']})."]
    L += ["", "## Giới hạn", "", "- Dữ liệu mô phỏng (D44): 200 câu do agent soạn; chưa kiểm chứng trên bài đăng thật của sinh viên. Nhãn là nhãn máy (`WAITING_HUMAN`).",
          "- Tường lửa không có NER (D46): tên người ngoài roster không kèm tín hiệu khác (S7) không được bảo vệ.",
          "- Rò rỉ chỉ quét những tác vụ có ghi `llm_request_payloads` trong cửa sổ đo và chỉ với roster seed; precheck Threads không gọi LLM sinh chữ (0 lời gọi)."]
    lib.write_report("e1", obj, L)


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    sp = ap.add_subparsers(dest="cmd", required=True)
    r = sp.add_parser("run")
    r.add_argument("--tag", required=True)
    sp.add_parser("leak")
    sp.add_parser("report")
    a = ap.parse_args()
    try:
        if a.cmd != "report":
            lib.refuse_demo()
            if not lib.ready():
                lib.die("Gateway chưa chạy")
        {"run": lambda: run(a.tag), "leak": leak, "report": report}[a.cmd]()
    except Fail as e:
        lib.die(f"LỖI: {e}", 2)


if __name__ == "__main__":
    main()
