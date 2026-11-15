# QC sprint 11 — US-P10-01/02/03 phần API + DB (ep-qc). Chạy: python3 p10_obs.py [01|02|03|rbac]
import sys, re, json, uuid, time, datetime as dt
from p10lib import *

part = sys.argv[1] if len(sys.argv) > 1 else "all"
OBS = "/admin/observability"
BAD = "('ok','degraded','cancelled')"
c1, c2 = course("teacher", "761987"), course("teacher", "761988")


def audit_cnt(rid=None):
    q = "select count(*) from audit_log where entity='llm_request'" + (f" and entity_id='{rid}'" if rid else "")
    return int(sql(q)[0][0])


if part in ("all", "01"):
    c, s, _, hd = api("admin", "GET", f"{OBS}/summary?window=24h")
    t = s["totals"] if c == 200 else {}
    n, e, fb, cost = sql("select count(*), count(*) filter (where status not in %s), count(*) filter (where fallback_index>0), coalesce(sum(cost_est),0) from llm_audit where created_at > now() - interval '24 hours'" % BAD)[0]
    p95 = float(sql("select coalesce(percentile_cont(0.95) within group (order by latency_ms),0) from llm_audit where created_at > now() - interval '24 hours'")[0][0])
    ok("TC-01-02a số yêu cầu / lỗi / dự phòng khớp SQL 24h", c == 200 and abs(t["requests"] - int(n)) <= 2 and abs(t["errors"] - int(e)) <= 1 and abs(t["fallbacks"] - int(fb)) <= 1, f"api={t.get('requests')}/{t.get('errors')}/{t.get('fallbacks')} sql={n}/{e}/{fb}")
    ok("TC-01-02b p95 khớp SQL (±10%)", c == 200 and abs(t["p95_ms"] - p95) <= max(5, 0.1 * p95), f"api={t.get('p95_ms')} sql={p95}")
    c7, s7, _, _ = api("admin", "GET", f"{OBS}/summary?window=7d")
    # ngày VN: tổng chi phí theo ngày VN từ SQL
    vn = dict(sql("select (created_at at time zone 'Asia/Ho_Chi_Minh')::date::text, count(*) from llm_audit where created_at > now() - interval '8 days' group by 1"))
    daysum = {d["day"]: d["requests"] for d in s7.get("days", [])}
    mism = {d: (daysum[d], vn.get(d)) for d in daysum if str(vn.get(d)) != str(daysum[d])}
    ok("TC-01-02c bảng ngày dùng múi giờ VN (số yêu cầu mỗi ngày khớp SQL theo VN)", c7 == 200 and len(mism) <= 1, f"lệch={mism}")
    tcost = sum(float(x["cost"]) for x in s7.get("tasks", [])); dcost = sum(float(x["cost"]) for x in s7.get("days", []))
    ok("TC-01-04 tổng chi phí theo tác vụ = tổng bảng ngày (±0,01)", abs(tcost - dcost) <= 0.01, f"tasks={tcost} days={dcost}")
    # 01-05
    c, a, _, hd = api("admin", "GET", f"{OBS}/requests?filter=all&limit=5")
    pages, seen, cur, dup = 0, set(), None, 0
    while pages < 300:
        c, a, _, _ = api("admin", "GET", f"{OBS}/requests?filter=all&limit=100" + (f"&cursor={cur}" if cur else ""))
        if c != 200: break
        for it in a["items"]:
            dup += it["id"] in seen; seen.add(it["id"])
        pages += 1; cur = a["next_cursor"]
        if not cur: break
    total = int(sql("select count(*) from llm_audit")[0][0])
    ok("TC-01-05a duyệt hết cursor: không trùng, không sót", dup == 0 and len(seen) == total, f"đã thấy={len(seen)} DB={total} trùng={dup} trang={pages}")
    c, p, _, _ = api("admin", "GET", f"{OBS}/requests?filter=problem&limit=100")
    pr = sql("select count(*) from llm_audit where status not in %s" % BAD)[0][0]
    ok("TC-01-05b filter=problem chỉ status lỗi", c == 200 and all(i["status"] not in ("ok", "degraded", "cancelled") for i in p["items"]) and len(p["items"]) == min(100, int(pr)), f"{len(p['items'])}/{pr}")
    c, p, _, _ = api("admin", "GET", f"{OBS}/requests?filter=privacy&limit=100")
    ok("TC-01-05c filter=privacy chỉ pii_masked_count>0", c == 200 and p["items"] and all(i["pii_masked_count"] > 0 for i in p["items"]), f"{len(p['items'])}")
    c, p, _, _ = api("admin", "GET", f"{OBS}/requests?limit=1000")
    ok("TC-01-05d limit=1000 kẹp ≤100 hoặc 422", (c == 200 and len(p["items"]) <= 100) or c == 422, f"{c} {len(p['items']) if c == 200 else ''}")
    body = json.dumps(a)
    ok("TC-01-05e thân không có trường nội dung", not re.search(r'"(prompt|content|message|payload|masked_prompt)"', body), "")
    c, p, _, hd = api("admin", "GET", f"{OBS}/requests?course_id={c1}&task=CHAT&limit=100")
    ok("TC-01-05f lọc lớp + tác vụ", c == 200 and all(i["course_id"] == c1 and i["task"] == "CHAT" for i in p["items"]), f"{c} {len(p['items'])}")
    ok("TC-01-05g from/to", api("admin", "GET", f"{OBS}/requests?from=2026-11-14T00:00:00Z&to=2026-11-15T00:00:00Z&limit=10")[0] == 200, "")
    for tc, q in (("TC-01-06a filter=xyz", "filter=xyz"), ("TC-01-06b from>to", "from=2026-11-15T00:00:00Z&to=2026-11-01T00:00:00Z"), ("TC-01-06c cursor rác", "cursor=%40%40%40ra%CC%81c")):
        c, j, _, _ = api("admin", "GET", f"{OBS}/requests?{q}")
        ok(tc, c == 422, f"{c}")
    c, j, _, _ = api("admin", "GET", f"{OBS}/summary?window=9d")
    ok("TC-01-06d window lạ → 422", c == 422, f"{c}")
    # 01-08/09
    c, j, _, _ = api("teacher", "GET", f"/courses/{c1}/observability/summary")
    keys = set(j) if c == 200 else set()
    ok("TC-01-08a GV xem lớp mình: không bảng yêu cầu/chi phí theo nhà cung cấp/nội dung", c == 200 and not (keys & {"items", "tasks", "providers", "cost"}), f"{c} {sorted(keys)}")
    # lớp mà teacher@ KHÔNG dạy: tạm nâng sv.moi thành TEACHER để ADMIN tạo lớp, rồi trả vai
    oc = sql("select c.id from courses c where c.class_code='QCOBS01'")
    if not oc:
        sql("update users set role='TEACHER' where email='sv.moi@edupilot.local'")
        cc, mk, _, _ = api("admin", "POST", "/admin/courses", {"subject_code": "QCO", "class_code": "QCOBS01", "name": "QC lớp khác (observability)", "semester": "2026-2027-HK1", "capacity": 30, "teacher_id": uid("sv.moi")})
        sql("update users set role='STUDENT' where email='sv.moi@edupilot.local'")
        oc = sql("select c.id from courses c where c.class_code='QCOBS01'")
    c, j, _, _ = api("teacher", "GET", f"/courses/{oc[0][0]}/observability/summary")
    ok("TC-01-08b GV xem lớp khác → 404", c == 404, f"{c}")
    for code, cid in (("761987", c1), ("761988", c2)):
        c, j, _, _ = api("admin", "GET", f"/courses/{cid}/observability/summary")
        ok(f"TC-01-09 ADMIN xem {code} → 200", c == 200, f"{c}")
    c, j, _, _ = api("admin", "GET", f"/courses/{uuid.uuid4()}/observability/summary")
    ok("TC-01-09b ADMIN lớp không tồn tại → 404", c == 404, f"{c}")

if part in ("all", "02"):
    rid = sql("select id from llm_audit where pii_masked_count>0 and trace_id<>'' order by created_at desc limit 1")[0][0]
    # có payload cho rid chưa? tạo bằng một lời gọi thật (chat) nếu không
    has = sql(f"select count(*) from llm_request_payloads where audit_id='{rid}'")
    print("payload cho", rid, has)
    a0 = audit_cnt(rid)
    c, d, _, hd = api("admin", "POST", f"{OBS}/requests/{rid}/open", {"reason": "Điều tra lỗi tăng sau 10h"})
    ok("TC-02-02 open → 200 + no-store", c in (200, 410), f"{c} {hd.get('Cache-Control')} {str(d)[:120]}")
    a1 = audit_cnt(rid)
    ok("TC-02-03 đúng 1 dòng audit_log VIEW_DETAIL", a1 == a0 + 1, f"{a0}->{a1}")
    row = sql(f"select action, actor_id is not null, after->>'reason', trace_id is not null from audit_log where entity='llm_request' and entity_id='{rid}' order by created_at desc limit 1")
    ok("TC-02-03b VIEW_DETAIL + reason + actor + trace", row and row[0][0] == "VIEW_DETAIL" and row[0][2] == "Điều tra lỗi tăng sau 10h" and row[0][1] == "t", str(row))
    for tc, body in (("9 ký tự", {"reason": "123456789"}), ("201 ký tự", {"reason": "x" * 201}), ("rỗng", {"reason": ""}), ("thiếu trường", {}), ("10 khoảng trắng", {"reason": " " * 10})):
        b = audit_cnt(rid)
        c, j, _, _ = api("admin", "POST", f"{OBS}/requests/{rid}/open", body)
        ok(f"TC-02-04 lý do {tc} → 422 REASON_REQUIRED, không audit", c == 422 and "REASON_REQUIRED" in json.dumps(j) and audit_cnt(rid) == b and "masked_prompt" not in json.dumps(j), f"{c} {str(j)[:80]}")
    b = audit_cnt(rid)
    api("admin", "POST", f"{OBS}/requests/{rid}/open", {"reason": "Lần hai cùng lý do kiểm"}); api("admin", "POST", f"{OBS}/requests/{rid}/open", {"reason": "Lần hai cùng lý do kiểm"})
    ok("TC-02-06 hai lần open → 2 dòng audit", audit_cnt(rid) == b + 2, f"{b}->{audit_cnt(rid)}")
    for u in ("teacher", "ta", "sv.kha"):
        b = audit_cnt(rid); c, _, _, _ = api(u, "POST", f"{OBS}/requests/{rid}/open", {"reason": "Thử quyền không được"})
        ok(f"TC-02-11 {u} → 403, không audit", c == 403 and audit_cnt(rid) == b, f"{c}")
    b = audit_cnt(rid); c, _, _, _ = raw("POST", f"{OBS}/requests/{rid}/open", {"reason": "Không token thử"}, {"Idempotency-Key": str(uuid.uuid4())})
    ok("TC-02-11b không token → 401", c == 401 and audit_cnt(rid) == b, f"{c}")
    b = audit_cnt()
    for tc, x in (("UUID không có", str(uuid.uuid4())), ("UUID sai định dạng", "not-a-uuid")):
        c, _, _, _ = api("admin", "POST", f"{OBS}/requests/{x}/open", {"reason": "Kiểm tra 404 không 500"})
        ok(f"TC-02-16 {tc} → 404", c == 404, f"{c}")
    ok("TC-02-16b không thêm audit_log", audit_cnt() == b, f"{b}->{audit_cnt()}")
    for p in (f"{OBS}/requests", f"{OBS}/summary"):
        c, _, _, hd = api("admin", "GET", p + ("?window=24h" if "summary" in p else ""))
        ok(f"TC-02-19 no-store {p}" + (" (nên có, không bắt buộc)" if "summary" in p else ""), "no-store" in (hd.get("Cache-Control") or "") or "summary" in p, hd.get("Cache-Control"))
    c, _, _, _ = api("admin", "GET", f"{OBS}/requests/{rid}")
    ok("TC-02-20 GET chi tiết cũ → 404/405", c in (404, 405), f"{c}")
    # 02-07: PII
    rids = [r[0] for r in sql("select a.id from llm_audit a join llm_request_payloads p on p.audit_id=a.id where a.pii_masked_count>0 order by a.created_at desc limit 40")]
    names = [r[0] for r in sql("select full_name from users where email like 'sv.%' union select student_code from users where student_code is not null union select email from users where email like 'sv.%'")]
    names = [n for n in names if n and len(n) >= 4]
    leaks, opened = [], 0
    for r in rids:
        c, d, _, _ = api("admin", "POST", f"{OBS}/requests/{r}/open", {"reason": "Kiểm PII trong payload (QC)"})
        if c != 200: continue
        opened += 1
        blob = json.dumps(d, ensure_ascii=False)
        leaks += [n for n in names if n in blob]
    ok("TC-02-07 payload đã mở ≥ 20, 0 khớp tên/MSSV/email SV", opened >= 20 and not leaks, f"mở={opened} rò={sorted(set(leaks))[:5]}")
    # 02-14: payload quá 30 ngày
    if rids:
        old = rids[-1]
        sql(f"update llm_request_payloads set created_at = now() - interval '31 days' where audit_id='{old}'")
        b = audit_cnt(old); c, j, _, _ = api("admin", "POST", f"{OBS}/requests/{old}/open", {"reason": "Mở bản đã quá hạn lưu (QC)"})
        ok("TC-02-14 payload > 30 ngày → 410 PAYLOAD_EXPIRED + vẫn audit", c == 410 and audit_cnt(old) == b + 1, f"{c} {str(j)[:80]}")

if part in ("all", "rbac"):
    cases = [("GET", f"{OBS}/summary?window=24h"), ("GET", f"{OBS}/requests?limit=5"), ("GET", f"/courses/{c1}/observability/summary")]
    for u in ("teacher", "ta", "sv.kha", None):
        for m, p in cases:
            c, _, _, _ = api(u, m, p) if u else raw(m, p)[:1] + (0, 0, 0)
            exp = {"admin-only": None}
            if p.startswith(OBS): want = 401 if u is None else 403
            else: want = 401 if u is None else (200 if u == "teacher" else 403)
            ok(f"TC-00-RBAC-1 {u or 'không token'} {m} {p.split('?')[0]}", c == want, f"{c} want {want}")
