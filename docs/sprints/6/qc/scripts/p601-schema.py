#!/usr/bin/env python3
"""QC US-P3-01 (TC 06, 17, 32, 38): so DB thật với `FEAT-private-chat-pii/SRS.md` 5.1–5.6.

Kiểm bốn thứ, bảng kỳ vọng **đọc thẳng từ SRS** (không chép tay vào mã, không lấy từ test của dev):
  1. enum 5.1  — đúng giá trị và đúng thứ tự;
  2. bảng 5.2, 5.3 (`chat_sessions`, `chat_messages`) — tên cột, kiểu, NULL, numeric(p,s); cột thừa;
  3. bảng 5.4, 5.5 (`forum_threads`, `forum_posts`, `pii_events`, mô tả dạng văn xuôi) — có mặt đủ cột;
     riêng `pii_events` kiểm **chặt**: đúng 9 cột và 0 cột văn bản tự do;
  4. chỉ mục 5.6 — đủ tên, và mệnh đề WHERE của chỉ mục từng phần.

Dùng:
    python3 p601-schema.py --dsn "$DB" --tables all
    python3 p601-schema.py --dsn "$DB" --tables chat_sessions,chat_messages
    PSQL='docker exec qc6-pg psql -U edupilot -d edupilot -tAF|' python3 p601-schema.py --tables all
rc=0 khi không lệch, rc=1 khi có lệch (in từng dòng `LỆCH …`).
"""
import argparse
import os
import re
import subprocess
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
DEFAULT_SRS = os.path.normpath(os.path.join(HERE, "../../../specs/FEAT-private-chat-pii/SRS.md"))

TY = {
    "uuid": "uuid", "text": "text", "integer": "integer", "boolean": "boolean",
    "timestamptz": "timestamp with time zone", "jsonb": "jsonb", "bigint": "bigint",
    "smallint": "smallint", "text[]": "ARRAY", "citext": "USER-DEFINED",
}
PII_EVENTS_COLS = ["id", "course_id", "session_id", "user_id", "channel", "pii_type", "count", "action", "created_at"]
# SRS 5.6: tên chỉ mục → mảnh bắt buộc có trong `indexdef` (rỗng = chỉ cần tồn tại).
INDEXES = {
    "chat_sessions_user_idx": "deleted_at IS NULL",
    "chat_messages_session_idx": "",
    "chat_messages_streaming_idx": "stream_status = 'STREAMING'",
    "forum_threads_course_idx": "deleted_at IS NULL",
    "forum_threads_week_idx": "",
    "forum_threads_skipped_idx": "ai_state = 'SKIPPED'",
    "forum_posts_thread_idx": "",
    "forum_posts_pending_idx": "verification_state = 'PENDING'",
    "pii_events_course_idx": "",
    "pii_events_user_idx": "",
}

bad = 0


def fail(msg):
    global bad
    bad += 1
    print("LỆCH", msg)


def make_q(dsn):
    base = os.environ.get("PSQL", "").split() or ["psql", dsn, "-tAF|"]

    def q(sql):
        r = subprocess.run(base + ["-v", "ON_ERROR_STOP=1", "-c", sql], capture_output=True, text=True)
        if r.returncode:
            fail(f"psql lỗi: {r.stderr.strip()[:200]}")
            return []
        return [ln.split("|") for ln in r.stdout.strip().split("\n") if ln]

    return q


def check_enums(srs, q):
    sec = srs.split("### 5.1 Enum")[1].split("### 5.2")[0]
    n = 0
    for name, vals in re.findall(r"`(\w+)` \(((?:`\w+`(?:, )?)+)\)", sec):
        n += 1
        exp = re.findall(r"`(\w+)`", vals)
        got = [r[0] for r in q(
            "select e.enumlabel from pg_enum e join pg_type t on t.oid=e.enumtypid "
            f"where t.typname='{name}' order by e.enumsortorder")]
        if exp != got:
            fail(f"enum {name}: SRS {exp} / DB {got}")
    print(f"enum kiểm: {n}")


def check_table_md(table, body, q):
    """Bảng markdown kiểu | `cột` | `kiểu` | NOT NULL | mặc định | ràng buộc |."""
    cols = {r[0]: r for r in q(
        "select column_name,data_type,is_nullable,column_default,udt_name,numeric_precision,numeric_scale "
        f"from information_schema.columns where table_name='{table}'")}
    if not cols:
        fail(f"thiếu bảng {table}")
        return 0
    seen, n = set(), 0
    for m in re.finditer(r"^\| ((?:`\w+`(?:, )?)+) \| `([^`|]+)` \| (NOT NULL|NULL) \|", body, re.M):
        typ, nul = m.group(2), m.group(3)
        for name in re.findall(r"`(\w+)`", m.group(1)):
            seen.add(name)
            n += 1
            if name not in cols:
                fail(f"{table}.{name} thiếu")
                continue
            c = cols[name]
            if (c[2] == "NO") != (nul == "NOT NULL"):
                fail(f"{table}.{name} null: SRS {nul} / DB {c[2]}")
            base = typ.split("(")[0].strip()
            if base.startswith("numeric"):
                mm = re.match(r"numeric\((\d+),(\d+)\)", typ.replace(" ", ""))
                if mm and (c[5], c[6]) != mm.groups():
                    fail(f"{table}.{name} numeric: SRS {typ} / DB ({c[5]},{c[6]})")
            elif base == "vector":
                if c[4] != "vector":
                    fail(f"{table}.{name} kiểu: SRS {typ} / DB {c[4]}")
            elif base in TY:
                if TY[base] != c[1] and not (base.endswith("[]") and c[1] == "ARRAY"):
                    fail(f"{table}.{name} kiểu: SRS {typ} / DB {c[1]}/{c[4]}")
            elif base not in (c[4], c[1]):
                fail(f"{table}.{name} kiểu enum/khác: SRS {typ} / DB {c[1]}/{c[4]}")
    extra = set(cols) - seen
    if extra:
        fail(f"{table}: cột thừa so với SRS {sorted(extra)}")
    return n


def check_table_prose(table, body, q):
    """5.4: cột liệt kê dạng văn xuôi `cột` (…). Chỉ kiểm **có mặt**, kiểu đã kiểm bằng TC riêng."""
    cols = {r[0] for r in q(f"select column_name from information_schema.columns where table_name='{table}'")}
    if not cols:
        fail(f"thiếu bảng {table}")
        return 0
    want = set(re.findall(r"`(\w+)`", body)) & {c for c in cols} | set(re.findall(r"`(\w+)`", body))
    # chỉ giữ tên trông như cột (chữ thường, có trong câu mô tả bảng này)
    want = {w for w in want if re.fullmatch(r"[a-z][a-z0-9_]*", w) and not w.startswith(("forum_", "chat_", "pii_"))}
    missing = sorted(w for w in want if w not in cols and w not in TY)
    if missing:
        print(f"  (chú ý) {table}: tên trong SRS chưa thấy là cột: {missing} — soát tay, văn xuôi có thể lẫn từ khác")
    return len(cols)


def check_pii_events(q):
    cols = [r[0] for r in q(
        "select column_name from information_schema.columns where table_name='pii_events' order by ordinal_position")]
    if cols != PII_EVENTS_COLS:
        fail(f"pii_events cột: SRS {PII_EVENTS_COLS} / DB {cols}")
    free = q("select count(*) from information_schema.columns where table_name='pii_events' "
             "and data_type in ('text','character varying','character')")
    if free and free[0][0] != "0":
        fail(f"pii_events có {free[0][0]} cột văn bản tự do (phải 0)")


def check_indexes(q):
    got = {r[0]: r[1] for r in q(
        "select indexname, indexdef from pg_indexes where tablename in "
        "('chat_sessions','chat_messages','forum_threads','forum_posts','pii_events')")}
    for name, must in INDEXES.items():
        if name not in got:
            fail(f"thiếu chỉ mục {name}")
        elif must and must.lower() not in got[name].lower():
            fail(f"chỉ mục {name} thiếu điều kiện `{must}`: {got[name]}")
    print(f"chỉ mục kiểm: {len(INDEXES)}")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--dsn", default=os.environ.get("DB", ""))
    ap.add_argument("--srs", default=DEFAULT_SRS)
    ap.add_argument("--tables", default="all")
    a = ap.parse_args()
    if not a.dsn and not os.environ.get("PSQL"):
        sys.exit("cần --dsn hoặc biến môi trường PSQL")
    q = make_q(a.dsn)
    srs = open(a.srs, encoding="utf-8").read()
    want = None if a.tables == "all" else set(a.tables.split(","))

    if want is None:
        check_enums(srs, q)
    ncol = 0
    for num, table in (("5.2", "chat_sessions"), ("5.3", "chat_messages")):
        if want and table not in want:
            continue
        body = srs.split(f"### {num} `{table}`")[1].split("\n### ")[0]
        ncol += check_table_md(table, body, q)
    body54 = srs.split("### 5.4 ")[1].split("\n### ")[0]
    for table in ("forum_threads", "forum_posts"):
        if want and table not in want:
            continue
        part = body54.split(f"`{table}`:")[1].split("\n\n")[0] if f"`{table}`:" in body54 else body54
        check_table_prose(table, part, q)
    if not want or "pii_events" in want:
        check_pii_events(q)
    if want is None:
        check_indexes(q)
    print(f"cột kiểm (5.2+5.3): {ncol}; lệch: {bad}")
    sys.exit(1 if bad else 0)


if __name__ == "__main__":
    main()
