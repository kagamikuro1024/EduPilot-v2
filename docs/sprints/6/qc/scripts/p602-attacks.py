#!/usr/bin/env python3
"""QC US-P3-02 (TC 07, 16, 26–28) + US-P3-03 (tệp chuỗi cấm): bộ tấn công dữ liệu cá nhân.

Bộ mẫu do QC soạn (không lấy từ test của dev), họ tên và MSSV lấy **thẳng từ DB** của lớp đang kiểm:
  --set names             họ tên roster × {có dấu, không dấu, HOA, đảo, rút}            → phải chặn (NAME)
  --set mssv/email/phone/cccd                                                           → phải chặn
  --set negatives-number  cổng, năm, IP, CVE, hex, kích thước khoá, dãy 10 số không '0' → phải cho qua
  --set negatives-name    một âm tiết, từ thường trùng tên, địa danh, viết tắt          → phải cho qua
  --set redact            tính idempotent của `Redact` và "văn bản sạch giữ nguyên byte"
  --set all               tất cả

Dùng:
    python3 p602-attacks.py --gw "$GW" --course "$C1" --token "$SVA_RAW" --dsn "$DB" --set all
    python3 p602-attacks.py --dsn "$DB" --course "$C1" --build-roster /tmp/roster.txt   # tệp chuỗi cấm cho p603
    python3 p602-attacks.py --selftest                                                   # kiểm logic sinh biến thể
rc=0 khi mọi mẫu đúng nhãn, rc=1 khi có mẫu sai (in bảng sai sót).
"""
import argparse
import json
import os
import re
import ssl
import subprocess
import sys
import unicodedata
import urllib.request

HIDDEN = "[đã ẩn]"


def vn_fold(s):
    s = unicodedata.normalize("NFD", s)
    s = "".join(c for c in s if not unicodedata.combining(c))
    return s.replace("đ", "d").replace("Đ", "D")


def variants(full_name):
    """Năm biến thể theo SRS 4.2.3: nguyên, không dấu, HOA, đảo (tên trước họ), rút (họ + tên)."""
    parts = full_name.split()
    out = [full_name, vn_fold(full_name).lower(), full_name.upper()]
    if len(parts) >= 2:
        out.append(" ".join([parts[-1]] + parts[:-1]))          # Khải Bùi Thanh
        out.append(f"{parts[-1]} {parts[0]}")                    # Khải Bùi
        out.append(f"{parts[0]} {parts[-1]}")                    # Bùi Khải
    seen, uniq = set(), []
    for v in out:
        if v not in seen:
            seen.add(v)
            uniq.append(v)
    return uniq


NEG_NUMBER = [
    "Mở cổng 8080 trên máy chủ thử nghiệm",
    "Đề thi năm 2022 có câu về hàm băm",
    "IP nội bộ 192.168.1.10 không ra Internet",
    "CVE-2021-44228 ảnh hưởng thư viện nào?",
    "Dãy 1234567890 có ý nghĩa gì trong ví dụ này?",
    "Khoá RSA 2048 bit còn an toàn không?",
    "Bài 2022 trong giáo trình nói về AES",
    "Chuỗi hash 4f3c2b1a9d8e7f6a5b4c3d2e là SHA-1 hay MD5?",
]
NEG_NAME = [
    "Khải",
    "An",
    "anh hiểu sai chỗ này rồi",
    "Hoa Kỳ dùng chuẩn mã hoá nào?",
    "Cần minh chứng cho luận điểm này",
    "MIT công bố bài báo về mật mã hậu lượng tử",
    "Mai thi lại được không ạ?",
]
POS_FIXED = {
    "mssv": [("Bài lab 3 của 20221234 nộp chưa?", "MSSV"),
             ("Mã B21DCAT123 có trong danh sách không?", "MSSV"),
             ("MSSV 123456 là ai ạ", "MSSV"),
             ("mã số sinh viên 1234567890123 ạ", "MSSV"),
             ("mssv: abc123xyz", "MSSV")],
    "email": [("Liên hệ nam.nt@edupilot.local giúp em", "EMAIL"),
              ("Gửi vào abc.xyz+tag@gmail.com nhé", "EMAIL"),
              ("Mail trường là sv@cntt.hust.edu.vn", "EMAIL")],
    "phone": [("Số của bạn ấy là 0912345678", "PHONE"),
              ("Gọi +84 912 345 678 giúp em", "PHONE"),
              ("SĐT 0912.345.678", "PHONE"),
              ("Liên hệ 09 1234 5678", "PHONE")],
    "cccd": [("CCCD 001203004567 của em", "CCCD"),
             ("Căn cước 001 203 004 567", "CCCD")],
}
CLEAN_TEXT = 'So sánh AES-GCM với ChaCha20 🙂\nDùng `[[ -f "$f" ]]` để kiểm tệp'


def psql(dsn, sql):
    base = os.environ.get("PSQL", "").split() or ["psql", dsn, "-tAF|"]
    r = subprocess.run(base + ["-v", "ON_ERROR_STOP=1", "-c", sql], capture_output=True, text=True)
    if r.returncode:
        sys.exit(f"psql lỗi: {r.stderr.strip()[:300]}")
    return [ln.split("|") for ln in r.stdout.strip().split("\n") if ln]


def roster(dsn, course_id):
    return psql(dsn, "select u.full_name, e.student_code_snapshot, u.email from enrollments e "
                     "join users u on u.id = e.user_id "
                     f"where e.course_id = '{course_id}' and e.status = 'ACTIVE' "
                     "and e.role_in_course = 'STUDENT' order by u.full_name")


def post(gw, course, token, body, title=None):
    payload = {"body": body}
    if title:
        payload["title"] = title
    req = urllib.request.Request(
        f"{gw}/api/v1/courses/{course}/threads/precheck",
        data=json.dumps(payload).encode(),
        headers={"Content-Type": "application/json", "Authorization": f"Bearer {token}"},
        method="POST")
    ctx = ssl._create_unverified_context()
    with urllib.request.urlopen(req, context=ctx, timeout=20) as r:
        return r.status, json.loads(r.read())


def build_corpus(sets, rows):
    """→ [(nhãn_bộ, văn_bản, phải_chặn, loại_kỳ_vọng)]"""
    out = []
    if "names" in sets:
        for full_name, _code, _email in rows:
            for v in variants(full_name):
                out.append(("names", f"{v} được mấy điểm lab?", True, "NAME"))
    for key in ("mssv", "email", "phone", "cccd"):
        if key in sets:
            out += [(key, t, True, k) for t, k in POS_FIXED[key]]
    if "mssv" in sets:
        for _n, code, _e in rows:
            out.append(("mssv", f"Mã {code} nộp bài chưa?", True, "MSSV"))
    if "email" in sets:
        for _n, _c, email in rows[:5]:
            out.append(("email", f"Gửi cho {email} giúp em", True, "EMAIL"))
    if "negatives-number" in sets:
        out += [("negatives-number", t, False, None) for t in NEG_NUMBER]
    if "negatives-name" in sets:
        out += [("negatives-name", t, False, None) for t in NEG_NAME]
    return out


def check(resp, text, must_block, kind, forbidden):
    """→ danh sách lỗi của một mẫu."""
    errs = []
    allowed = resp.get("allowed")
    types = {r.get("type") for r in resp.get("reasons") or []}
    red = resp.get("redacted_text", "")
    if must_block:
        if allowed is not False:
            errs.append(f"allowed={allowed} (phải false)")
        if kind and kind not in types:
            errs.append(f"reasons={sorted(types)} (thiếu {kind})")
        if HIDDEN not in red:
            errs.append(f"redacted_text không có {HIDDEN}")
        for raw in forbidden:
            if raw and raw.lower() in red.lower():
                errs.append(f"redacted_text còn chuỗi gốc {raw!r}")
    else:
        if allowed is not True:
            errs.append(f"allowed={allowed} (phải true) reasons={sorted(types)}")
        if red != text:
            errs.append("redacted_text khác body từng byte")
    return errs


def run_redact_checks(gw, course, token):
    """AC4: idempotent + văn bản sạch giữ nguyên byte."""
    errs = []
    _, r1 = post(gw, course, token, "Bùi Thanh Khải 20229002 a@b.local 0912345678")
    red = r1.get("redacted_text", "")
    _, r2 = post(gw, course, token, red)
    if r2.get("allowed") is not True or r2.get("redacted_text") != red:
        errs.append(f"Redact không idempotent: {red!r} → {r2.get('redacted_text')!r} allowed={r2.get('allowed')}")
    _, r3 = post(gw, course, token, CLEAN_TEXT)
    if r3.get("redacted_text") != CLEAN_TEXT:
        errs.append("văn bản sạch bị đổi byte")
    if r3.get("allowed") is not True:
        errs.append(f"văn bản sạch bị chặn: {r3.get('reasons')}")
    return errs


def selftest():
    v = variants("Bùi Thanh Khải")
    assert "bui thanh khai" in v, v
    assert "BÙI THANH KHẢI" in v, v
    assert "Khải Bùi Thanh" in v, v
    assert "Khải Bùi" in v, v
    assert "Bùi Khải" in v, v
    assert vn_fold("Đặng Vũ Hoàng") == "Dang Vu Hoang"
    assert variants("An") == ["An", "an", "AN"], variants("An")
    bad = check({"allowed": True, "reasons": [], "redacted_text": "x"}, "x", True, "NAME", [])
    assert bad, "check() phải báo lỗi khi mẫu dương lại được cho qua"
    ok = check({"allowed": False, "reasons": [{"type": "NAME"}], "redacted_text": HIDDEN}, "x", True, "NAME", ["khải"])
    assert ok == [], ok
    print("selftest OK")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--gw")
    ap.add_argument("--course")
    ap.add_argument("--token")
    ap.add_argument("--dsn", default=os.environ.get("DB", ""))
    ap.add_argument("--set", dest="sets", default="all")
    ap.add_argument("--build-roster", metavar="FILE")
    ap.add_argument("--selftest", action="store_true")
    a = ap.parse_args()

    if a.selftest:
        selftest()
        return

    rows = roster(a.dsn, a.course) if a.dsn and a.course else []

    if a.build_roster:
        lines = []
        for full_name, code, email in rows:
            lines += variants(full_name) + [code, email]
        with open(a.build_roster, "w", encoding="utf-8") as f:
            f.write("\n".join(sorted(set(x for x in lines if x))) + "\n")
        print(f"{a.build_roster}: {len(set(lines))} chuỗi cấm từ {len(rows)} sinh viên")
        return

    if not (a.gw and a.course and a.token):
        sys.exit("cần --gw, --course, --token (hoặc dùng --build-roster / --selftest)")

    all_sets = ["names", "mssv", "email", "phone", "cccd", "negatives-number", "negatives-name", "redact"]
    sets = set(all_sets) if a.sets == "all" else set(a.sets.split(","))
    corpus = build_corpus(sets, rows)
    forbidden_by_row = {r[0]: [r[0], r[1], r[2]] for r in rows}

    fails = 0
    for label, text, must_block, kind in corpus:
        raw = []
        if label == "names":
            for name, vals in forbidden_by_row.items():
                if vn_fold(name).lower() in vn_fold(text).lower():
                    raw = [name]
                    break
        elif must_block:
            raw = re.findall(r"[\w.+-]+@[\w.-]+|\b\d{6,13}\b|\b[A-Za-z]\d{2}[A-Za-z]{4}\d{3}\b", text)
        try:
            status, resp = post(a.gw, a.course, a.token, text)
        except Exception as e:                                   # noqa: BLE001 — in nguyên nhân rồi tính là sai
            print(f"SAI [{label}] {text[:60]!r}: gọi API lỗi {e}")
            fails += 1
            continue
        if status != 200:
            print(f"SAI [{label}] {text[:60]!r}: HTTP {status}")
            fails += 1
            continue
        errs = check(resp, text, must_block, kind, raw)
        if errs:
            fails += 1
            print(f"SAI [{label}] {text[:60]!r}: " + "; ".join(errs))

    if "redact" in sets:
        for e in run_redact_checks(a.gw, a.course, a.token):
            fails += 1
            print(f"SAI [redact] {e}")

    print(f"mẫu chạy: {len(corpus)} (+redact); sai: {fails}")
    sys.exit(1 if fails else 0)


if __name__ == "__main__":
    main()
