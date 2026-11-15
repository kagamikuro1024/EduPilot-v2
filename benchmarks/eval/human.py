#!/usr/bin/env python3
"""Gói soát nhãn của chủ dự án cho E1 (P-11-3) và E2 (P-11-4). Không gọi stack, không gọi LLM.

  human.py pack     → <OUT>/e1-soat-nhan.csv (57 mẫu phân tầng: 4 / nhóm dương S1–S8, 5 / nhóm âm N1–N5; KHÔNG hiện nhãn máy và nhóm,
                       thứ tự xáo cố định — chủ dự án gán mù, tránh neo) + <OUT>/e2-soat-nhan.csv (24 mục E2-005 … E2-120, đúng bảng benchmarks/e2/README.md)
  human.py status   tiến độ E1 / E2 / E3 từ bản đã điền ở benchmarks/eval/human/ (e1_soat.csv, e2_soat.csv, e3_scores.csv)
Mẫu chọn theo vị trí trong nhóm (rải đều), KHÔNG theo kết quả chạy. Bản đã điền đọc ở `lib.human_review`.
"""
import argparse
import csv
import json
import random
import sys

import lib

sys.path.insert(0, str(lib.ROOT / "benchmarks"))
import eval_pii  # noqa: E402  — dùng đúng hàm điền ô của bộ đo E1

E1_DATA, E2_DATA = lib.ROOT / "benchmarks/pii/e1_dataset.jsonl", lib.ROOT / "benchmarks/e2/golden_e2.jsonl"


def spread(xs, k):
    return [xs[round(i * (len(xs) - 1) / (k - 1))] for i in range(k)] if len(xs) > k else xs


def pack():
    roster = {r["n"]: r for r in json.loads((lib.ROOT / "benchmarks/pii/roster_seed.json").read_text(encoding="utf-8"))}
    items = [json.loads(x) for x in E1_DATA.read_text(encoding="utf-8").splitlines() if x.strip()]
    by = {}
    for it in items:
        by.setdefault(it["stratum"], []).append(it)
    pick = [it for s in sorted(by) for it in spread(by[s], 4 if s.startswith("S") else 5)]
    random.Random(1011).shuffle(pick)
    p1 = lib.OUT / "e1-soat-nhan.csv"
    with p1.open("w", newline="", encoding="utf-8") as f:  # như e3-cham-tay.csv; bản điền đọc bằng utf-8-sig nên có BOM cũng được
        w = csv.writer(f)
        w.writerow(["stt", "ma", "van_ban", "nhan_chu_du_an (CHẶN / CHO QUA)", "ghi_chu"])
        for i, it in enumerate(pick, 1):
            w.writerow([i, f"M{i:02d}", eval_pii.fill(it["text"], roster), "", ""])
    lib.save_raw("e1-soat-map.json", {"note": "mã M.. của e1-soat-nhan.csv → id mẫu (id lộ nhóm nên không in trong gói)", "map": {f"M{i:02d}": it["id"] for i, it in enumerate(pick, 1)}})
    gold = [json.loads(x) for x in E2_DATA.read_text(encoding="utf-8").splitlines() if x.strip()]
    rows = [g for g in gold if int(g["id"].split("-")[1]) % 5 == 0]
    p2 = lib.OUT / "e2-soat-nhan.csv"
    with p2.open("w", newline="", encoding="utf-8") as f:
        w = csv.writer(f)
        w.writerow(["stt", "id", "cau_hoi", "tra_loi_duoc", "phai_chua (moi nhom it nhat 1 bien the)", "nguon", "trang", "bang_chung",
                    "ket_luan (Đúng / Sai nhãn / Câu hỏi mơ hồ)", "ghi_chu"])
        for i, g in enumerate(rows, 1):
            w.writerow([i, g["id"], g["question"], "có" if g["answerable"] else "không", " | ".join(" / ".join(alt) for alt in g.get("must_contain") or []),
                        g.get("source", ""), g.get("page", ""), g.get("evidence", ""), "", ""])
    print(f"→ {p1} ({len(pick)} mẫu: {sum(it['expect_block'] for it in pick)} dương, {sum(not it['expect_block'] for it in pick)} âm)\n→ {p2} ({len(rows)} mục)")


def status():
    for kind in ("e1", "e2"):
        r = lib.human_review(kind)
        print(kind.upper(), r["status"] if r else "chưa có bản điền", "" if not r else {k: v for k, v in r.items() if k != "status"})
    p = lib.HERE / "human/e3_scores.csv"
    n = 0
    if p.exists():
        with p.open(encoding="utf-8-sig") as f:
            n = sum(1 for row in csv.reader(f) if len(row) > 6 and row[1].startswith("kc") and all(x.strip() for x in row[3:7]))
    print(f"E3 đã chấm {n}/60 bài")
    print("E5: python3 benchmarks/eval/e5.py report (đọc gen-stats từ stack ep-qc)")


if __name__ == "__main__":
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("cmd", choices=["pack", "status"])
    {"pack": pack, "status": status}[ap.parse_args().cmd]()
