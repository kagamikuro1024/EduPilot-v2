#!/usr/bin/env python3
"""E1 — đo tường lửa PII của Threads (US-P3-07, SRS FEAT-private-chat-pii 9.3).

Gọi POST /courses/{id}/threads/precheck bằng token SINH VIÊN của lớp seed (chỉ đọc, không ghi); `allowed=false` ⇒ chặn.
Mã thoát: 0 đạt, 1 không đạt chỉ số, 2 lỗi (API không tới, 401, bộ dữ liệu sai, token sai vai).
Không tham số ngẫu nhiên: cùng dữ liệu seed + cùng bộ dữ liệu ⇒ cùng TP/FP/FN/TN. Không in nội dung mẫu `test`.

  python benchmarks/eval_pii.py --validate [--strict-synthetic]
  python benchmarks/eval_pii.py --min-recall 0.95 --max-false-block 0.05
Biến môi trường: API_URL (mặc định https://localhost/api/v1), SEED_DEFAULT_PASSWORD, E1_TOKEN (dùng token có sẵn thay vì đăng nhập).
"""
import argparse
import base64
import hashlib
import json
import os
import re
import ssl
import sys
import time
import urllib.error
import urllib.request
from collections import Counter, defaultdict
from pathlib import Path

HERE = Path(__file__).resolve().parent
PII_DIR = HERE / "pii"
STRATA_POS = ["S1", "S2", "S3", "S4", "S5", "S6", "S7", "S8"]
STRATA_NEG = ["N1", "N2", "N3", "N4", "N5"]
EXPECTED_COUNTS = {"S1": 12, "S2": 8, "S3": 8, "S4": 6, "S5": 20, "S6": 38, "S7": 4, "S8": 4, "N1": 50, "N2": 15, "N3": 15, "N4": 10, "N5": 10}
PII_TYPES = {"MSSV", "EMAIL", "PHONE", "CCCD", "NAME", "PERSONAL_QUESTION"}
SLOT = re.compile(r"\{\{SV(\d{2})\.(name|name\.noaccent|name\.reversed|mssv|mssv\.spaced|mssv\.dotted|email)\}\}")
# Hằng số ngưỡng của máy chủ ở thời điểm đo (SRS mục 3 / 4.8) — ghi vào báo cáo để tái lập.
CONSTANTS = {"PII_PERSONAL_SIM_HIGH": 0.78, "PII_PERSONAL_SIM_LOW": 0.55, "RAG_SIM_FLOOR": 0.25, "RAG_SIM_CEIL": 0.65, "THREAD_SIMILAR_MIN": 0.75}
CLASS_CODE = "761987"  # lớp 1 của seed


class Fail(Exception):
    """Lỗi → mã thoát 2."""


def fold(s):
    import unicodedata
    s = s.replace("đ", "d").replace("Đ", "D")
    return "".join(c for c in unicodedata.normalize("NFD", s) if not unicodedata.combining(c))


def load_dataset(path):
    try:
        raw = Path(path).read_bytes()
    except OSError as e:
        raise Fail(f"không đọc được bộ dữ liệu {path}: {e}")
    items = []
    for n, line in enumerate(raw.decode("utf-8").splitlines(), 1):
        if not line.strip():
            continue
        try:
            items.append(json.loads(line))
        except json.JSONDecodeError as e:
            raise Fail(f"{path}:{n}: JSON sai: {e}")
    return items, hashlib.sha256(raw).hexdigest()


def cap_runs(text):
    """Các cụm ≥ 2 từ viết hoa chữ đầu liền nhau (tên riêng khả dĩ) trong văn bản đã điền ô."""
    import unicodedata
    cur = []
    for w in re.findall(r"[^\W\d_]+|[^\w\s]", unicodedata.normalize("NFC", text)):
        if len(w) > 1 and w[0].isalpha() and w[0].isupper() and w[1:].islower():
            cur.append(w)
            continue
        if len(cur) >= 2:
            yield cur
        cur = []
    if len(cur) >= 2:
        yield cur


def free_names(text, roster, names, benign):
    """Cụm viết hoa không thuộc roster seed / danh sách bịa / danh sách từ thường viết hoa (Hà Nội, Hoa Kỳ…): tên tự do ngoài dữ liệu mô phỏng (D44).
    Tên được coi là có mặt khi một đoạn cuối của cụm (bỏ từ đầu câu viết hoa như "Bạn") có mọi âm tiết nằm trong cùng một tên đã biết, không phân biệt dấu / thứ tự."""
    sets = [set(fold(n).lower().split()) for n in names]
    bad = []
    for run in cap_runs(fill(text, roster)):
        phrase = " ".join(run)
        if phrase in benign:
            continue
        toks = [fold(w).lower() for w in run]
        if not any(len(toks[i:]) >= 2 and any(set(toks[i:]) <= st for st in sets) for i in range(len(toks) - 1)):
            bad.append(phrase)
    return bad


def validate(items, roster, outside, strict, benign=frozenset()):
    errs = []
    ids = Counter(i.get("id") for i in items)
    errs += [f"trùng id: {k}" for k, v in ids.items() if v > 1]
    if len(items) != 200:
        errs.append(f"cần đúng 200 mẫu, có {len(items)}")
    for it in items:
        i = it.get("id", "?")
        for k in ("id", "split", "stratum", "text", "expect_block", "pii_types", "channel_expected"):
            if k not in it:
                errs.append(f"{i}: thiếu trường {k}")
        if it.get("split") not in ("dev", "test"):
            errs.append(f"{i}: split lạ {it.get('split')!r}")
        if it.get("stratum") not in EXPECTED_COUNTS:
            errs.append(f"{i}: nhóm lạ {it.get('stratum')!r}")
        if not isinstance(it.get("expect_block"), bool):
            errs.append(f"{i}: expect_block phải là bool")
        if it.get("channel_expected") != ("PRIVATE" if it.get("expect_block") else "PUBLIC"):
            errs.append(f"{i}: channel_expected không khớp expect_block")
        bad = set(it.get("pii_types", [])) - PII_TYPES
        if bad:
            errs.append(f"{i}: nhãn pii_types lạ {sorted(bad)}")
        if it.get("expect_block") != (it.get("stratum", "")[:1] == "S"):
            errs.append(f"{i}: nhóm {it.get('stratum')} không khớp expect_block")
        if (it.get("expect_block") and not it.get("pii_types")) or (not it.get("expect_block") and it.get("pii_types")):
            errs.append(f"{i}: pii_types không khớp expect_block")
        for m in SLOT.finditer(it.get("text", "")):
            if int(m.group(1)) not in roster:
                errs.append(f"{i}: ô điền {m.group(0)} không có trong roster seed")
        if "{{" in SLOT.sub("", it.get("text", "")):
            errs.append(f"{i}: ô điền sai cú pháp")
        if strict:
            for n in it.get("outside_names", []):
                if n not in outside:
                    errs.append(f"{i}: tên {n!r} không nằm trong roster seed hoặc danh sách bịa outside_roster_names")
                if n not in it.get("text", ""):
                    errs.append(f"{i}: outside_names {n!r} không xuất hiện trong text")
            for phrase in free_names(it.get("text", ""), roster, outside, benign):
                errs.append(f"{i}: cụm viết hoa {phrase!r} không thuộc roster seed / danh sách bịa / danh sách từ thường viết hoa (tên tự do ngoài dữ liệu mô phỏng)")
            if it.get("stratum") == "S7" and not it.get("outside_names"):
                errs.append(f"{i}: mẫu S7 phải khai outside_names")
    cnt = Counter(i.get("stratum") for i in items)
    for s, want in EXPECTED_COUNTS.items():
        if cnt[s] != want:
            errs.append(f"nhóm {s}: cần {want} mẫu, có {cnt[s]}")
    pos = sum(1 for i in items if i.get("expect_block") is True)
    if pos != 100 or len(items) - pos != 100:
        errs.append(f"cần 100 dương + 100 âm, có {pos} + {len(items) - pos}")
    dev = sum(1 for i in items if i.get("split") == "dev")
    if dev != 60 or len(items) - dev != 140:
        errs.append(f"cần dev 60 / test 140, có {dev} / {len(items) - dev}")
    for s in EXPECTED_COUNTS:
        if not any(i.get("stratum") == s and i.get("split") == "dev" for i in items):
            errs.append(f"nhóm {s}: không có mẫu dev (phân tầng sai)")
    if errs:
        raise Fail("bộ dữ liệu sai:\n  " + "\n  ".join(errs[:40]))
    return pos, len(items) - pos, dev, len(items) - dev


def fill(text, roster):
    def rep(m):
        r = roster[int(m.group(1))]
        name = r["name"]
        words = name.split()
        return {"name": name, "name.noaccent": fold(name), "name.reversed": " ".join(reversed(words)), "mssv": r["code"],
                "mssv.spaced": r["code"][:4] + " " + r["code"][4:], "mssv.dotted": r["code"][:4] + "." + r["code"][4:], "email": r["email"]}[m.group(2)]
    return SLOT.sub(rep, text)


def jwt_role(token):
    try:
        p = token.split(".")[1]
        return json.loads(base64.urlsafe_b64decode(p + "=" * (-len(p) % 4))).get("role")
    except Exception:
        return None


def ctx_for(url):
    c = ssl.create_default_context()
    if os.environ.get("E1_INSECURE", "1") == "1" and url.startswith("https://localhost"):
        c.check_hostname, c.verify_mode = False, ssl.CERT_NONE  # stack dev dùng chứng chỉ tự ký
    return c


def call(api, method, path, token=None, body=None):
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(api + path, data=data, method=method, headers={"Content-Type": "application/json", **({"Authorization": f"Bearer {token}"} if token else {})})
    try:
        with urllib.request.urlopen(req, timeout=15, context=ctx_for(api)) as r:
            return r.status, json.loads(r.read() or b"null"), dict(r.headers)
    except urllib.error.HTTPError as e:
        try:
            payload = json.loads(e.read() or b"null")
        except Exception:
            payload = None
        return e.code, payload, dict(e.headers)
    except (urllib.error.URLError, OSError) as e:
        raise Fail(f"không gọi được API {api}: {e}")


def login(api):
    tok = os.environ.get("E1_TOKEN")
    if tok:
        return tok
    pw = os.environ.get("SEED_DEFAULT_PASSWORD")
    if not pw:
        raise Fail("thiếu SEED_DEFAULT_PASSWORD (hoặc E1_TOKEN) để đăng nhập tài khoản Sinh viên seed")
    st, body, _ = call(api, "POST", "/auth/login", body={"email": os.environ.get("E1_EMAIL", "sv.kha@edupilot.local"), "password": pw})
    if st != 200 or not body or "access_token" not in body:
        raise Fail(f"đăng nhập Sinh viên seed thất bại (HTTP {st})")
    return body["access_token"]


def course_id(api, token):
    st, body, _ = call(api, "GET", "/me/courses?limit=100", token)
    if st == 401:
        raise Fail("token hết hạn / không hợp lệ (401)")
    if st != 200:
        raise Fail(f"không lấy được danh sách lớp (HTTP {st})")
    for it in body.get("items", []):
        if it["course"]["class_code"] == CLASS_CODE:
            return it["course"]["id"]
    raise Fail(f"tài khoản Sinh viên không thuộc lớp seed {CLASS_CODE}")


def precheck(api, token, cid, text, pace):
    for _ in range(6):
        st, body, hdr = call(api, "POST", f"/courses/{cid}/threads/precheck", token, {"body": text})
        if st == 200:
            time.sleep(pace)
            return body
        if st == 401:
            raise Fail("token hết hạn (401) giữa chừng")
        if st == 429:  # giới hạn 60 yêu cầu / phút / người: chờ rồi gọi lại (không tính vào kết quả)
            time.sleep(float(hdr.get("Retry-After", 5)) + 0.5)
            continue
        raise Fail(f"precheck trả HTTP {st}")
    raise Fail("precheck bị giới hạn tốc độ liên tục")


def metrics(c):
    tp, fp, fn, tn = c["tp"], c["fp"], c["fn"], c["tn"]
    prec = tp / (tp + fp) if tp + fp else 1.0
    rec = tp / (tp + fn) if tp + fn else 1.0
    return {"recall": round(rec, 4), "precision": round(prec, 4), "f1": round(2 * prec * rec / (prec + rec), 4) if prec + rec else 0.0,
            "false_block": round(fp / (fp + tn), 4) if fp + tn else 0.0}


def run(args, items, roster, digest):
    api = args.api.rstrip("/")
    token = login(api)
    role = jwt_role(token)
    if role != "STUDENT":
        raise Fail(f"script chỉ chạy bằng token Sinh viên (token này có vai {role}); từ chối")
    cid = course_id(api, token)
    new = lambda: {"tp": 0, "fp": 0, "fn": 0, "tn": 0}  # noqa: E731
    total, split, stratum, ptype, chan = new(), defaultdict(new), defaultdict(new), defaultdict(new), Counter()
    dev_misses = []
    for it in items:
        text = fill(it["text"], roster)
        out = precheck(api, token, cid, text, args.pace)
        blocked = not out["allowed"]
        pred_chan = "PRIVATE" if blocked else "PUBLIC"
        chan[f"{it['channel_expected']}->{pred_chan}"] += 1
        k = ("tp" if blocked else "fn") if it["expect_block"] else ("fp" if blocked else "tn")
        for bucket in (total, split[it["split"]], stratum[it["stratum"]]):
            bucket[k] += 1
        for t in it["pii_types"]:
            ptype[t][k] += 1
        if k in ("fn", "fp") and it["split"] == "dev":
            dev_misses.append(it["id"])
    counts = {"tp": total["tp"], "fp": total["fp"], "fn": total["fn"], "tn": total["tn"]}
    overall = metrics(total)
    report = {
        "counts": counts,
        "metrics": overall,
        "dataset_sha256": digest,
        "constants": CONSTANTS,
        "gate": {"min_recall": args.min_recall, "max_false_block": args.max_false_block},
        "by_split": {s: {**dict(c), **metrics(c)} for s, c in sorted(split.items())},
        "by_stratum": {s: {**dict(c), "recall" if s.startswith("S") else "false_block": metrics(c)["recall" if s.startswith("S") else "false_block"]} for s, c in sorted(stratum.items())},
        "by_pii_type": {t: {**dict(c), "recall": metrics(c)["recall"]} for t, c in sorted(ptype.items())},
        "channel_confusion": dict(sorted(chan.items())),
        "dev_misses": dev_misses,  # chỉ mã mẫu của tập dev; tập test không in nội dung / mã
        "s7_leaked": stratum["S7"]["fn"],
    }
    ok = overall["recall"] >= args.min_recall and overall["false_block"] <= args.max_false_block
    report["pass"] = ok
    return report


def write_reports(report, out_dir):
    out_dir.mkdir(parents=True, exist_ok=True)
    (out_dir / "e1.json").write_text(json.dumps(report, ensure_ascii=False, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    m, c = report["metrics"], report["counts"]
    lines = [
        "# E1 — tường lửa PII của Threads", "",
        f"Kết quả: **{'ĐẠT' if report['pass'] else 'KHÔNG ĐẠT'}** — recall {m['recall']:.3f} (cần ≥ {report['gate']['min_recall']}), chặn nhầm {m['false_block']:.3f} (cần ≤ {report['gate']['max_false_block']}), precision {m['precision']:.3f}, F1 {m['f1']:.3f}.",
        f"Bộ dữ liệu `e1_dataset.jsonl` sha256 `{report['dataset_sha256'][:16]}…`; TP {c['tp']} · FP {c['fp']} · FN {c['fn']} · TN {c['tn']}. Hằng số: " + ", ".join(f"{k}={v}" for k, v in report["constants"].items()) + ".", "",
        "## Theo tập", "| tập | TP | FN | FP | TN | recall | chặn nhầm |", "| --- | --- | --- | --- | --- | --- | --- |",
    ]
    for s, v in report["by_split"].items():
        lines.append(f"| {s} | {v['tp']} | {v['fn']} | {v['fp']} | {v['tn']} | {v['recall']:.3f} | {v['false_block']:.3f} |")
    lines += ["", "## Theo nhóm", "| nhóm | TP | FN | FP | TN | recall / chặn nhầm |", "| --- | --- | --- | --- | --- | --- |"]
    for s, v in report["by_stratum"].items():
        lines.append(f"| {s} | {v['tp']} | {v['fn']} | {v['fp']} | {v['tn']} | {v.get('recall', v.get('false_block')):.3f} |")
    lines += ["", "## Theo loại PII (recall)", "| loại | recall | FN |", "| --- | --- | --- |"]
    for t, v in report["by_pii_type"].items():
        lines.append(f"| {t} | {v['recall']:.3f} | {v['fn']} |")
    lines += ["", "## Ma trận nhầm lẫn kênh", "| kỳ vọng → dự đoán | số mẫu |", "| --- | --- |"]
    lines += [f"| {k} | {v} |" for k, v in report["channel_confusion"].items()]
    lines += [
        "", "## Giới hạn đã biết",
        f"- **S7 — tên người ngoài roster, không tín hiệu khác:** tường lửa không có NER (D46) nên vùng này không được bảo vệ. {report['s7_leaked']}/4 mẫu S7 lọt; vẫn tính vào chỉ số chung (tối đa làm recall tụt xuống 0,96).",
        "- Bộ dữ liệu là mô phỏng (D44), soạn bởi Dev và soát độc lập bởi QC (≥ 50 mẫu); không đại diện cách viết của sinh viên thật. Luật chỉ được chỉnh trên tập `dev`.",
        "- Nhúng ở stack seed là `fake` (tất định): bước tương đồng ngữ nghĩa không đóng góp; kết quả phản ánh **luật** (regex + roster + mẫu câu).",
        "- Chỉ đo `allowed` của `precheck`; chưa đo chất lượng câu trả lời có / không che (ghi cho luận văn).",
    ]
    (out_dir / "e1.md").write_text("\n".join(lines) + "\n", encoding="utf-8")


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--api", default=os.environ.get("API_URL", "https://localhost/api/v1"))
    ap.add_argument("--dataset", default=str(PII_DIR / "e1_dataset.jsonl"))
    ap.add_argument("--roster", default=str(PII_DIR / "roster_seed.json"))
    ap.add_argument("--outside", default=str(PII_DIR / "outside_roster_names.json"))
    ap.add_argument("--benign", default=str(PII_DIR / "benign_capitalized.json"), help="cụm viết hoa không phải tên người (Hà Nội, Hoa Kỳ…)")
    ap.add_argument("--out-dir", default=str(HERE / "reports"))
    ap.add_argument("--validate", action="store_true", help="chỉ kiểm bộ dữ liệu, không gọi API")
    ap.add_argument("--strict-synthetic", action="store_true", help="từ chối tên không thuộc roster seed / danh sách bịa")
    ap.add_argument("--min-recall", type=float, default=0.95)
    ap.add_argument("--max-false-block", type=float, default=0.05)
    ap.add_argument("--pace", type=float, default=1.05, help="giây giữa hai lần gọi (giới hạn precheck 60 / phút / người)")
    args = ap.parse_args()
    try:
        items, digest = load_dataset(args.dataset)
        roster = {r["n"]: r for r in json.loads(Path(args.roster).read_text(encoding="utf-8"))}
        outside = set(json.loads(Path(args.outside).read_text(encoding="utf-8")))
        outside |= {r["name"] for r in roster.values()}
        benign = frozenset(json.loads(Path(args.benign).read_text(encoding="utf-8"))) if args.strict_synthetic else frozenset()
        pos, neg, dev, test = validate(items, roster, outside, args.strict_synthetic, benign)
        strata = ",".join(f"{s}={n}" for s, n in EXPECTED_COUNTS.items())
        if args.validate:
            print(f"OK {len(items)} items; positives={pos} negatives={neg}; dev={dev} test={test}; strata={strata}")
            return 0
        report = run(args, items, roster, digest)
        write_reports(report, Path(args.out_dir))
        m = report["metrics"]
        if report["pass"]:
            print(f"E1 PASS recall={m['recall']:.3f} false_block={m['false_block']:.3f}")
            return 0
        print(f"E1 FAIL recall={m['recall']:.3f} false_block={m['false_block']:.3f}; mẫu dev sai: {', '.join(report['dev_misses']) or '(không)'}")
        return 1
    except Fail as e:
        print(f"LỖI: {e}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
