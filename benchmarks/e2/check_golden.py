#!/usr/bin/env python3
"""Kiểm tra golden set E2 (chỉ thư viện chuẩn). Dùng: python3 benchmarks/e2/check_golden.py [--policy-md PATH] [--text-dir DIR]

- Cấu trúc, id / câu hỏi duy nhất, số lượng (README).
- Mục `policy`: `evidence` phải có trong `docs/specs/FEAT-gradebook/fixtures/quy-che-lop1.md`; chính `evidence` phải qua `must_contain` của mục đó.
- `--text-dir`: thư mục chứa `<tên PDF>.txt` (văn bản đã trích) để kiểm `evidence` của mục `academic`.
Hàm `norm` và `passes` là định nghĩa chấm tất định; bộ chạy E2 phải import đúng hai hàm này.
"""
import argparse, json, os, re, sys, unicodedata

HERE = os.path.dirname(os.path.abspath(__file__))
DEFAULT_POLICY = os.path.join(HERE, "..", "..", "docs", "specs", "FEAT-gradebook", "fixtures", "quy-che-lop1.md")


def norm(s):
    s = unicodedata.normalize("NFD", s.replace("đ", "d").replace("Đ", "D"))
    s = "".join(c for c in s if unicodedata.category(c) != "Mn")
    s = s.replace("–", "-").replace("—", "-").replace("’", "'").replace("‘", "'").replace("“", '"').replace("”", '"')
    s = re.sub(r"[*_`]", "", s).lower()
    s = re.sub(r"(?<=\d),(?=\d)", ".", s)
    s = re.sub(r"\s+", " ", s).strip()
    return s


def passes(answer, item):
    """Đúng khi MỖI nhóm trong must_contain có ít nhất một biến thể xuất hiện trong câu trả lời đã chuẩn hoá."""
    a = norm(answer)
    return all(any(norm(v) in a for v in group) for group in item["must_contain"])


def evnorm(s):  # so khớp evidence với văn bản nguồn: không dấu thập phân bị đổi, không bỏ dấu tiếng Việt
    s = s.replace("–", "-").replace("—", "-").replace("’", "'")
    s = re.sub(r"\s+", " ", s).lower()
    return re.sub(r"\s*-\s*", "-", s)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--policy-md", default=DEFAULT_POLICY)
    ap.add_argument("--text-dir")
    args = ap.parse_args()
    rows = [json.loads(l) for l in open(os.path.join(HERE, "golden_e2.jsonl"), encoding="utf-8") if l.strip()]
    errs = []
    ids, qs = set(), set()
    cnt = {}
    for r in rows:
        for k in ("id", "set", "topic", "source", "page", "question", "answerable", "must_contain", "evidence"):
            if k not in r:
                errs.append(f"{r.get('id')}: thiếu `{k}`")
        if r["id"] in ids: errs.append(f"{r['id']}: trùng id")
        if r["question"] in qs: errs.append(f"{r['id']}: trùng câu hỏi")
        ids.add(r["id"]); qs.add(r["question"])
        cnt[(r["set"], r["answerable"])] = cnt.get((r["set"], r["answerable"]), 0) + 1
        if r["answerable"]:
            if not r["must_contain"] or not all(g and all(isinstance(v, str) and v == norm(v) for v in g) for g in r["must_contain"]):
                errs.append(f"{r['id']}: must_contain rỗng hoặc biến thể chưa ở dạng chuẩn hoá")
            if not r["evidence"] or not r["source"] or not r["page"]:
                errs.append(f"{r['id']}: thiếu evidence / source / page")
        elif r["must_contain"] or r["evidence"] or r["source"]:
            errs.append(f"{r['id']}: answerable=false nhưng có must_contain / evidence / source")
    want = {("policy", True): 15, ("policy", False): 5, ("academic", True): 80, ("academic", False): 20}
    if cnt != want:
        errs.append(f"số lượng {cnt} != {want}")
    pol = evnorm(open(args.policy_md, encoding="utf-8").read())
    for r in rows:
        if r["set"] == "policy" and r["answerable"]:
            if evnorm(r["evidence"]) not in pol:
                errs.append(f"{r['id']}: evidence không có trong quy chế")
            if not passes(r["evidence"], r):
                errs.append(f"{r['id']}: evidence không qua must_contain của chính mục (nhãn có thể sai)")
    if args.text_dir:
        for r in rows:
            if r["set"] == "academic" and r["answerable"]:
                p = os.path.join(args.text_dir, r["source"] + ".txt")
                if not os.path.exists(p):
                    errs.append(f"{r['id']}: thiếu {p}"); continue
                if evnorm(r["evidence"]) not in evnorm(open(p, encoding="utf-8").read()):
                    errs.append(f"{r['id']}: evidence không có trong {r['source']}")
    for e in errs:
        print("LỖI", e)
    print(f"{len(rows)} mục", {f"{k[0]}/{'answerable' if k[1] else 'unanswerable'}": v for k, v in sorted(cnt.items())}, "OK" if not errs else f"{len(errs)} lỗi")
    return 1 if errs else 0


if __name__ == "__main__":
    sys.exit(main())
