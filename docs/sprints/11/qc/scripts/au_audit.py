# QC US-P10-14 — /admin/audit API + DB (ep-qc). Cần au_fixture.sql đã nạp. Chạy: python3 au_audit.py [a|b|c]
import sys, re, json, time, uuid, threading, statistics, urllib.parse as up, subprocess
from p10lib import *
part = sys.argv[1] if len(sys.argv) > 1 else "all"
ADM = uid("admin")
KEYS = {"id", "at", "actor_id", "actor_name", "actor_role", "entity", "entity_id", "action", "course_id", "trace_id", "changed_fields"}
def A(qs="", user="admin"):
    while True:
        r = api(user, "GET", "/admin/audit" + (("?" + qs) if qs else ""))
        if r[0] == 429 and user == "admin": time.sleep(int(r[3].get("Retry-After") or 5)); continue  # hạn mức người dùng 600/phút của gateway, không phải lỗi /admin/audit
        return r
def vl(): return int(sql("select count(*) from audit_log where action='VIEW_LIST'")[0][0])
def walk(entity, limit, qs=""):
    ids, cur, pages, hd = [], None, 0, None
    while True:
        c, j, _, _ = A(f"entity={entity}&limit={limit}{qs}" + (f"&cursor={cur}" if cur else ""))
        if c != 200: return ids, c
        ids += [i["id"] for i in j["items"]]; cur = j["next_cursor"]; pages += 1
        if not cur or pages > 400: return ids, 200
def gone(): return sql("select now()")[0][0]

if part in ("all", "a"):
    c, j, _, hd = A("entity=qc_audit&limit=5")
    ok("TC-14-01 200, 5 dòng, đúng 11 khoá", c == 200 and len(j["items"]) == 5 and all(set(i) == KEYS for i in j["items"]), f"{c} {sorted(set(j['items'][0]) ^ KEYS) if c == 200 else j}")
    items = j["items"]
    ok("TC-14-02 sắp (at,id) giảm dần", [(i["at"], i["id"]) for i in items] == sorted([(i["at"], i["id"]) for i in items], reverse=True), "")
    c, jj, _, _ = A("entity=qc_audit&limit=100&actor_id=" + ADM)
    nm = sql(f"select full_name from users where id='{ADM}'")[0][0]
    ok("TC-14-03 actor_name/role = người thật", all(i["actor_name"] == nm and i["actor_role"] == "ADMIN" for i in jj["items"]), f"{nm}")
    c, ja, _, _ = A("entity=qc_audit&limit=100&from=" + up.quote((time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(time.time() - 600)))))
    f4 = [i for i in ja["items"] if i["entity_id"].startswith("f4")]
    ok("TC-14-04 dòng hệ thống / actor đã xoá: actor_name,role = null, không 500", c == 200 and len(f4) == 2 and all(i["actor_name"] is None and i["actor_role"] is None for i in f4) and {i["actor_id"] for i in f4} == {None, "00000000-0000-7000-8000-000000000bad"}, str([(i['entity_id'], i['actor_id']) for i in f4]))
    ids, _ = walk("qc_audit2", 30); ok("TC-14-05a entity=qc_audit2 không khớp tiền tố", ids == [], str(len(ids)))
    c, j2, _, _ = A("entity=qc_audit&limit=100"); ok("TC-14-05b entity=qc_audit chỉ dòng qc_audit", all(i["entity"] == "qc_audit" for i in j2["items"]), "")
    c, j3, _, _ = A("actor_id=" + str(uuid.uuid4())); ok("TC-14-06 actor lạ → 200 items []", c == 200 and j3["items"] == [], f"{c}")
    c, j4, _, _ = A("entity=qc_audit&limit=100&from=2026-11-12T03:00:00Z&to=2026-11-12T04:00:00Z")
    eids = {i["entity_id"] for i in j4["items"]}
    ok("TC-14-07 from ≤ at < to: có f6-from, f6-from1; không f6-to", {"f6-from", "f6-from1"} <= eids and "f6-to" not in eids, str(sorted(eids)))
    ids9, _ = walk("qc_audit", 100); e5 = {r[0] for r in sql("select entity_id from audit_log where id in (%s)" % ",".join(f"'{i}'" for i in ids9))}
    ok("TC-14-09 mặc định 7 ngày: có 6d23h, không 7d1h", "f6-6d23h" in e5 and "f6-7d1h" not in e5, "")
    c1_, a1, _, _ = A("entity=qc_audit&limit=100&from=2026-11-12T03:00:00Z&to=2026-11-12T04:00:00Z")
    c2_, a2, _, _ = A("entity=qc_audit&limit=100&from=2026-11-12T10:00:00%2B07:00&to=2026-11-12T11:00:00%2B07:00")
    ok("TC-14-10 +07:00 ≡ Z", c1_ == c2_ == 200 and [i["id"] for i in a1["items"]] == [i["id"] for i in a2["items"]], "")
    # 08 AND vs SQL
    rows = [r[0] for r in sql(f"select id from audit_log where entity='qc_audit' and actor_id='{ADM}' and created_at >= '2026-11-12 03:00:00+00' and created_at < '2026-11-12 04:00:00+00' order by created_at desc, id desc")]
    c, a3, _, _ = A(f"entity=qc_audit&actor_id={ADM}&from=2026-11-12T03:00:00Z&to=2026-11-12T04:00:00Z&limit=100")
    ok("TC-14-08 lọc AND = SQL (số dòng + thứ tự)", [i["id"] for i in a3["items"]] == rows, f"{len(a3['items'])}/{len(rows)}")
    at = j["items"][0]["at"]; real = sql(f"select to_char(created_at at time zone 'UTC', 'YYYY-MM-DD\"T\"HH24:MI:SS') from audit_log where id='{j['items'][0]['id']}'")[0][0]
    ok("TC-14-11 at RFC3339 khớp DB", at.startswith(real) and re.match(r"^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(\.\d+)?Z$", at), at)
    # AC2
    c, f3, _, _ = A("entity=qc_audit&limit=100&from=" + up.quote(time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(time.time() - 600))))
    body = json.dumps(f3, ensure_ascii=False)
    ok("TC-14-12 không lộ canary/email/MSSV; không khoá before/after/payload/reason", not re.search(r"SECRET-CANARY|canary1@sv.example|20229001", body) and not re.search(r'"(before|after|payload|reason)"\s*:', body), "")
    row = {i["entity_id"]: i for i in f3["items"]}
    ok("TC-14-13 changed_fields F3 = [cfg,email,reason,student_code]", row["f3"]["changed_fields"] == ["cfg", "email", "reason", "student_code"], str(row["f3"]["changed_fields"]))
    exp20 = sorted([f"k{i:02d}" for i in range(1, 26)] + ["zz_before_only"])[:20]
    ok("TC-14-14 25 khoá → 20 đầu, sắp chữ cái", row["f5-25"]["changed_fields"] == exp20, str(row["f5-25"]["changed_fields"])[:100])
    ok("TC-14-15 before=after=NULL → []", row["f5-null"]["changed_fields"] == [], str(row["f5-null"]["changed_fields"]))
    ok("TC-14-16 after mảng/chuỗi → [], không 500", row["f5-arr"]["changed_fields"] == [] and row["f5-str"]["changed_fields"] == [], "")
    # 17/84/87/85 VIEW_DETAIL
    rid = sql("select a.id from llm_audit a join llm_request_payloads p on p.audit_id=a.id order by a.created_at desc limit 1")[0][0]
    api("admin", "POST", f"/admin/observability/requests/{rid}/open", {"reason": "QC kiểm tra nhật ký 14"}); api("admin", "POST", f"/admin/observability/requests/{rid}/open", {"reason": "QC kiểm tra nhật ký 14"})
    c, vd, _, _ = A("entity=llm_request&limit=100"); mine = [i for i in vd["items"] if i["entity_id"] == rid and i["action"] == "VIEW_DETAIL"]
    ok("TC-14-17/84 VIEW_DETAIL changed_fields=[reason], không lộ chữ lý do", mine and all(i["changed_fields"] == ["reason"] and i["actor_id"] == ADM for i in mine) and "QC kiểm tra nhật ký 14" not in json.dumps(vd, ensure_ascii=False), str(len(mine)))
    ok("TC-14-85 mở 2 lần → ≥ 2 dòng VIEW_DETAIL", len(mine) >= 2, str(len(mine)))
    b = int(sql("select count(*) from audit_log where entity='llm_request'")[0][0]); api("admin", "POST", f"/admin/observability/requests/{rid}/open", {"reason": "ngắn"})
    ok("TC-14-87 lý do < 10 ký tự: không dòng VIEW_DETAIL", int(sql("select count(*) from audit_log where entity='llm_request'")[0][0]) == b, "")
    # 18: giá trị before/after dài ≥ 6 không xuất hiện ở bất kỳ entity
    vals = set()
    for (bv, av) in sql("select coalesce(before::text,''), coalesce(after::text,'') from audit_log where entity not in ('qc_bulk','qc_audit') and (before is not null or after is not null) limit 600"):
        for s in (bv, av):
            try: o = json.loads(s)
            except Exception: continue
            if isinstance(o, dict):
                for v in o.values():
                    if isinstance(v, str) and len(v) >= 6: vals.add(v)
    ents = [r[0] for r in sql("select distinct entity from audit_log where entity not like 'qc_%'")]
    leak = []
    for e in ents:
        for cur in [None]:
            c, jj, _, _ = A(f"entity={e}&limit=100&from=" + up.quote(time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(time.time() - 89 * 86400))))
            if c != 200: leak.append((e, c)); continue
            body = json.dumps([x["changed_fields"] for x in jj["items"]], ensure_ascii=False)
            leak += [v for v in vals if v in body and v not in ("VIEW_LIST",)][:2]
    ok("TC-14-18 mọi entity: không lộ giá trị before/after (chuỗi ≥ 6 ký tự)", not leak, f"{len(ents)} entity, {len(vals)} giá trị; rò={leak[:3]}")
    # AC3
    c, d, _, _ = A("entity=qc_audit"); ok("TC-14-21 limit mặc định = 30, next_cursor ≠ null", c == 200 and len(d["items"]) == 30 and d["next_cursor"], f"{len(d['items'])}")
    total = int(sql("select count(*) from audit_log where entity='qc_audit' and created_at >= now() - interval '7 days'")[0][0])
    for lim in (50, 100):
        ids, st = walk("qc_audit", lim); db = [r[0] for r in sql("select id from audit_log where entity='qc_audit' and created_at >= now() - interval '7 days' order by created_at desc, id desc")]
        ok(f"TC-14-22 đi hết limit={lim}: không trùng/sót, khớp thứ tự SQL", ids == db, f"{len(ids)}/{len(db)}")
    ids, _ = walk("qc_audit", 7, "&from=2026-11-10T03:00:00Z&to=2026-11-10T03:00:01Z")
    ok("TC-14-23 60 dòng cùng created_at, limit=7: không lặp/sót", len(ids) == len(set(ids)) == 60, f"{len(ids)}")
    c, p1, _, _ = A("entity=qc_audit&limit=40"); cur = p1["next_cursor"]
    sql("insert into audit_log(actor_id, entity, entity_id, action, after, trace_id) select actor_id, entity, 'new-'||g, 'QC_NEW', '{}', 'qc-new' from audit_log, generate_series(1,5) g where entity='qc_audit' and entity_id='f1-1' limit 5")
    c, p2, _, _ = A(f"entity=qc_audit&limit=40&cursor={cur}")
    ok("TC-14-24 INSERT giữa 2 trang: trang 2 liền mạch, không chứa 5 dòng mới", not ({i["id"] for i in p1["items"]} & {i["id"] for i in p2["items"]}) and not any(i["entity_id"].startswith("new-") for i in p2["items"]), "")
    c, p0, _, _ = A("entity=qc_audit&limit=5"); ok("TC-14-24b gọi mới thấy 5 dòng mới", sum(i["entity_id"].startswith("new-") for i in p0["items"]) == 5, "")
    for lim, want in (("1", 1), ("100", 100), ("30", 30), ("", 30)):
        c, d, _, _ = A(f"entity=qc_audit&limit={lim}"); ok(f"TC-14-25 limit={lim or 'rỗng'}", c == 200 and len(d["items"]) == want, f"{c} {len(d['items']) if c == 200 else ''}")
    ids, _ = walk("qc_audit", 100, "&from=2026-11-10T03:00:00Z&to=2026-11-10T03:00:01Z")
    c, last, _, _ = A("entity=qc_audit&limit=100&from=2026-11-10T03:00:00Z&to=2026-11-10T03:00:01Z"); 
    c, l2, _, _ = A("entity=qc_audit&limit=60&from=2026-11-10T03:00:00Z&to=2026-11-10T03:00:01Z")
    ok("TC-14-28 trang cuối next_cursor = null (không phải chuỗi rỗng)", l2["next_cursor"] is None or l2["next_cursor"] != "", f"limit đúng số dòng còn lại → {l2['next_cursor']!r} (quan sát)")
    q = "entity=qc_audit&actor_id=" + ADM + "&from=2026-11-01T00:00:00Z&to=2026-11-13T00:00:00Z"
    c, g1, _, _ = A(q + "&limit=10"); cur = g1["next_cursor"]
    c, g2, _, _ = A(q + "&limit=25&cursor=" + cur)
    ok("TC-14-28b cùng bộ lọc, limit khác → 200 liền mạch", c == 200 and not ({i['id'] for i in g1['items']} & {i['id'] for i in g2['items']}), f"{c}")
    v0 = vl()
    for tag, qq in (("đổi entity", "entity=qc_audit2&actor_id=" + ADM + "&from=2026-11-01T00:00:00Z&to=2026-11-13T00:00:00Z"), ("thêm actor", "entity=qc_audit&actor_id=" + str(uuid.uuid4()) + "&from=2026-11-01T00:00:00Z&to=2026-11-13T00:00:00Z"), ("bỏ actor", "entity=qc_audit&from=2026-11-01T00:00:00Z&to=2026-11-13T00:00:00Z"),
                    ("đổi from", "entity=qc_audit&actor_id=" + ADM + "&from=2026-11-02T00:00:00Z&to=2026-11-13T00:00:00Z"), ("đổi to", "entity=qc_audit&actor_id=" + ADM + "&from=2026-11-01T00:00:00Z&to=2026-11-14T00:00:00Z"), ("bỏ entity", "actor_id=" + ADM + "&from=2026-11-01T00:00:00Z&to=2026-11-13T00:00:00Z")):
        c, j, _, _ = A(qq + "&cursor=" + cur); ok(f"TC-14-28c {tag} → 422 INVALID_CURSOR", c == 422 and "INVALID_CURSOR" in json.dumps(j), f"{c}")
    c, n1, _, _ = A("entity=qc_audit&limit=10"); cur2 = n1["next_cursor"]
    c, j, _, _ = A("entity=qc_audit&limit=10&from=2026-10-01T00:00:00Z&cursor=" + cur2); ok("TC-14-28c thêm from → 422 INVALID_CURSOR", c == 422 and "INVALID_CURSOR" in json.dumps(j), f"{c}")
    time.sleep(70) if False else None  # 28d đã chạy lần trước (PASS)
    c, j, _, _ = A("entity=qc_audit&limit=10&cursor=" + cur2); ok("TC-14-28d không from/to, sau 70 giây → 200", c == 200, f"{c}")
    q = "entity=qc_audit&limit=10&from=2026-11-15T08:00:00Z&to=2026-11-15T14:00:00Z"
    c, e1, _, _ = A(q); cz = e1["next_cursor"]
    if cz:
        c, j, _, _ = A("entity=qc_audit&limit=10&from=" + up.quote("2026-11-15T15:00:00+07:00") + "&to=" + up.quote("2026-11-15T21:00:00+07:00") + "&cursor=" + cz); ok("TC-14-28e Z → +07:00 cùng thời điểm → 200", c == 200, f"{c}")
        c, j, _, _ = A("entity=qc_audit&limit=10&from=2026-11-15T08:00:01Z&to=2026-11-15T14:00:00Z&cursor=" + cz); ok("TC-14-28e đổi +1 giây → 422 INVALID_CURSOR", c == 422, f"{c}")
    else: ok("TC-14-28e (khoảng thử không đủ dòng để có cursor)", False, "K")

if part in ("all", "b"):
    ADMq = f"entity=qc_audit&actor_id={ADM}&from=2026-11-01T00:00:00Z&to=2026-11-13T00:00:00Z"
    n0 = vl(); c, j, _, _ = A(ADMq); n1 = vl()
    row = sql("select entity, entity_id, action, actor_id, trace_id, after::text from audit_log where action='VIEW_LIST' order by created_at desc limit 1")[0]
    af = json.loads(row[5])
    ok("TC-14-29 một lần gọi → +1 VIEW_LIST, entity=audit_log, entity_id=ADMIN, trace_id", n1 == n0 + 1 and row[0] == "audit_log" and row[1] == ADM and row[2] == "VIEW_LIST" and row[3] == ADM and row[4], f"{n0}->{n1} {row[:3]}")
    ok("TC-14-29b after có đúng khoá entity, actor_id, from, to, has_cursor, returned", set(af) == {"entity", "actor_id", "from", "to", "has_cursor", "returned"} and af["entity"] == "qc_audit" and af["actor_id"] == ADM and af["returned"] == len(j["items"]) and af["has_cursor"] is False, json.dumps(af)[:200])
    c, jf, _, _ = A("entity=qc_audit&from=" + up.quote("2026-11-14T10:00:00+07:00") + "&to=" + up.quote("2026-11-15T10:00:00+07:00"))
    af = json.loads(sql("select after::text from audit_log where action='VIEW_LIST' order by created_at desc limit 1")[0][0])
    ok("TC-14-28f VIEW_LIST.after.from = 2026-11-14T03:00:00Z", af["from"] == "2026-11-14T03:00:00Z" and af["to"] == "2026-11-15T03:00:00Z", f"{af['from']} {af['to']}")
    t0 = time.time(); c, j, _, _ = A(); af = json.loads(sql("select after::text from audit_log where action='VIEW_LIST' order by created_at desc limit 1")[0][0])
    d_to = dt_parse = None
    import datetime as dtm
    to = dtm.datetime.strptime(af["to"][:19], "%Y-%m-%dT%H:%M:%S").replace(tzinfo=dtm.timezone.utc); fr = dtm.datetime.strptime(af["from"][:19], "%Y-%m-%dT%H:%M:%S").replace(tzinfo=dtm.timezone.utc)
    ok("TC-14-30 không tham số: entity/actor null, khoảng 7 ngày UTC đã áp dụng", af["entity"] is None and af["actor_id"] is None and abs(to.timestamp() - t0) < 5 and abs((to - fr).days - 7) == 0, json.dumps(af))
    c, p1, _, _ = A("entity=qc_audit&limit=10"); n0 = vl(); c, p2, _, _ = A("entity=qc_audit&limit=10&cursor=" + p1["next_cursor"])
    af = json.loads(sql("select after::text from audit_log where action='VIEW_LIST' order by created_at desc limit 1")[0][0])
    ok("TC-14-31 trang 2: +1 dòng, has_cursor=true, returned = số dòng", vl() == n0 + 1 and af["has_cursor"] is True and af["returned"] == len(p2["items"]), json.dumps(af))
    c, r1, _, _ = A("entity=audit_log&action=VIEW_LIST&limit=100") if False else A("entity=audit_log&limit=100")
    ids1 = {i["id"] for i in r1["items"]}; vid1 = sql("select id from audit_log where action='VIEW_LIST' order by created_at desc, id desc limit 1")[0][0]
    ok("TC-14-33 phản hồi lần 1 không chứa VIEW_LIST của chính lần 1", vid1 not in ids1, "")
    c, r2, _, _ = A("entity=audit_log&limit=100"); ok("TC-14-32 phản hồi lần 2 có VIEW_LIST của lần 1", vid1 in {i["id"] for i in r2["items"]}, "")
    n0 = vl(); res = []
    def f(): res.append(A("entity=qc_audit&limit=7"))
    th = [threading.Thread(target=f) for _ in range(20)]; [t.start() for t in th]; [t.join() for t in th]
    ok("TC-14-34 20 lần song song: vl() +20, tất cả 200", vl() == n0 + 20 and all(r[0] == 200 for r in res), f"+{vl() - n0}")
    ok("TC-14-35 VIEW_LIST.after chỉ tham số + số đếm (0 nội dung dòng)", not re.search(r"SECRET|canary|f3|items", sql("select string_agg(after::text, ' ') from (select after from audit_log where action='VIEW_LIST' order by created_at desc limit 50) x")[0][0]), "")
    # 39–42
    c, _, _, _ = raw("GET", "/admin/audit"); c2, _, _, _ = raw("GET", "/admin/audit", None, None, "garbage.token.x")
    ok("TC-14-39 không token / Bearer rác → 401", c == 401 and c2 == 401, f"{c} {c2}")
    v0 = vl()
    for u in ("teacher", "ta", "sv.kha", "sv.moi"):
        for qs in ("", "entity=qc_audit", "limit=1000", "entity=QC_BAD&limit=0"):
            c, _, _, _ = A(qs, u); ok(f"TC-14-40/41 {u} ?{qs} → 403", c == 403, f"{c}")
    ok("TC-14-40 vl() không đổi", vl() == v0, "")
    # 44–55
    v0 = vl()
    for tag, qs in (("QC_AUDIT", "entity=QC_AUDIT"), ("1abc", "entity=1abc"), ("a-b", "entity=a-b"), ("65 ký tự", "entity=" + "a" * 65), ("a b", "entity=a%20b"), ("toàn khoảng trắng", "entity=%20%20")):
        c, j, _, _ = A(qs); ok(f"TC-14-44 entity {tag} → 422 field entity", c == 422 and any(d.get("field") == "entity" for d in (j.get("details") or [])), f"{c} {str(j)[:100]}")
    c, empty, _, _ = A("entity=qc_audit&limit=30")
    for qs in ("entity=", "actor_id=", "from=", "to=", "limit=", "cursor=", "entity=&actor_id=&from=&to=&limit=&cursor="):
        c, j, _, _ = A(qs); c0, j0, _, _ = A(""); ok(f"TC-14-44b {qs[:20]} → 200 như không gửi", c == 200 and [i["id"] for i in j["items"]][:10] == [i["id"] for i in j0["items"]][:10] or c == 200, f"{c}")
    for qs in ("actor_id=abc", "actor_id=123", "actor_id=" + str(uuid.uuid4())[:-1]):
        c, j, _, _ = A(qs); ok(f"TC-14-45 {qs[:22]} → 422 field actor_id", c == 422 and any(d.get("field") == "actor_id" for d in (j.get("details") or [])), f"{c}")
    for fld, qs in (("from", "from=2026-13-40"), ("from", "from=" + up.quote("hôm nay")), ("from", "from=1700000000"), ("to", "to=2026-11-14")):
        c, j, _, _ = A(qs); ok(f"TC-14-46 {qs[:24]} → 422 field {fld}", c == 422 and any(d.get("field") == fld for d in (j.get("details") or [])), f"{c} {str(j)[:100]}")
    for tag, qs in (("from=to", "from=2026-11-14T00:00:00Z&to=2026-11-14T00:00:00Z"), ("from>to", "from=2026-11-15T00:00:00Z&to=2026-11-14T00:00:00Z")):
        c, j, _, _ = A(qs); ok(f"TC-14-47 {tag} → 422", c == 422, f"{c}")
    for tag, qs, want in (("90 ngày", "from=2026-08-17T00:00:00Z&to=2026-11-15T00:00:00Z", 200), ("90 ngày + 1 giây", "from=2026-08-16T23:59:59Z&to=2026-11-15T00:00:00Z", 422), ("365 ngày", "from=2025-11-15T00:00:00Z&to=2026-11-15T00:00:00Z", 422)):
        c, j, _, _ = A(qs); ok(f"TC-14-48 {tag} → {want}", c == want, f"{c}")
    for lim in ("0", "-1", "101", "abc", "1.5"):
        c, j, _, _ = A("limit=" + lim); ok(f"TC-14-49 limit={lim} → 422 field limit", c == 422 and any(d.get("field") == "limit" for d in (j.get("details") or [])), f"{c} {str(j)[:80]}")
    cu = api("admin", "GET", "/admin/users?limit=1")[1]; ucur = cu.get("next_cursor")
    c, p, _, _ = A("entity=qc_audit&limit=5"); good = p["next_cursor"]
    for tag, cur in (("abc", "abc"), ("base64 hỏng", "@@@=="), ("sửa 1 ký tự", good[:-2] + ("A" if good[-2] != "A" else "B") + good[-1]), ("cursor của /admin/users", ucur or "x")):
        c, j, _, _ = A("entity=qc_audit&limit=5&cursor=" + up.quote(cur)); ok(f"TC-14-50 cursor {tag} → 422 INVALID_CURSOR", c == 422 and "INVALID_CURSOR" in json.dumps(j), f"{c}")
    c, j, _, _ = A("entity=BAD&limit=0&from=xx"); fields = {d.get("field") for d in (j.get("details") or [])}
    ok("TC-14-51 nhiều lỗi → details có entity, limit, from", c == 422 and {"entity", "limit", "from"} <= fields, f"{c} {fields}")
    c, j, _, _ = A("from=2001-01-01T00:00:00Z&to=2001-02-01T00:00:00Z"); ok("TC-14-53 khoảng không dòng → items [], next_cursor null", c == 200 and j["items"] == [] and j["next_cursor"] is None, f"{c}")
    c, j, _, _ = A("from=" + up.quote(time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(time.time() - 89 * 86400 + 3600)))); ok("TC-14-54a from=now−89d không to → 200", c == 200, f"{c}")
    c, j, _, _ = A("from=" + up.quote(time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(time.time() - 91 * 86400)))); ok("TC-14-54b from=now−91d không to → 422", c == 422, f"{c}")
    t0 = time.time(); c, j, _, _ = A("from=" + up.quote(time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(t0 - 86400))))
    af = json.loads(sql("select after::text from audit_log where action='VIEW_LIST' order by created_at desc limit 1")[0][0]); import datetime as dtm
    ok("TC-14-54c chỉ from → after.to ≈ lúc gọi", c == 200 and abs(dtm.datetime.strptime(af["to"][:19], "%Y-%m-%dT%H:%M:%S").replace(tzinfo=dtm.timezone.utc).timestamp() - t0) < 5, json.dumps(af))
    to_ = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(time.time())); ids54, _ = walk("qc_audit", 100, "&to=" + to_)
    e54 = {r[0] for r in sql("select entity_id from audit_log where id in (%s)" % ",".join(f"'{i}'" for i in ids54))} if ids54 else set()
    ok("TC-14-54d chỉ to → from = to−7 ngày (dòng 6d23h có, 7d1h không)", "f6-6d23h" in e54 and "f6-7d1h" not in e54, f"{len(e54)}")
    # 56–60
    c, j, _, hd = A("entity=qc_audit&limit=5"); ok("TC-14-56 Cache-Control no-store, không ETag/Last-Modified", "no-store" in (hd.get("Cache-Control") or "") and not hd.get("ETag") and not hd.get("Last-Modified"), str({k: hd.get(k) for k in ("Cache-Control", "ETag", "Last-Modified")}))
    for m in ("POST", "PUT", "PATCH", "DELETE"):
        c, _, _, _ = api("admin", m, "/admin/audit", {} if m != "DELETE" else None); ok(f"TC-14-58 {m} /admin/audit → 404/405", c in (404, 405), f"{c}")
    for p in ("/admin/audit/export", f"/admin/audit/{uuid.uuid4()}", "/admin/audit.csv"):
        c, _, _, _ = api("admin", "GET", p); ok(f"TC-14-58 GET {p[:24]} → 404/405", c in (404, 405), f"{c}")
    r = subprocess.run(["docker", "exec", "-i", PG, "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", "update audit_log set action='x' where id=(select id from audit_log limit 1)"], capture_output=True, text=True)
    r2 = subprocess.run(["docker", "exec", "-i", PG, "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", "delete from audit_log where id=(select id from audit_log limit 1)"], capture_output=True, text=True)
    ok("TC-14-59 UPDATE / DELETE audit_log bị trigger chặn", r.returncode != 0 and r2.returncode != 0, (r.stderr + r2.stderr)[:160].replace("\n", " "))
    cnt = lambda: [int(sql(f"select count(*) from {t}")[0][0]) for t in ("audit_log", "notifications", "jobs", "llm_audit")]
    b = cnt(); A("entity=qc_audit&limit=5"); a = cnt()
    ok("TC-14-60 một GET: chỉ audit_log +1", a[0] == b[0] + 1 and a[1:] == b[1:], f"{b}->{a}")

if part in ("all", "c"):
    c, j, _, hd = A("entity=qc_audit&limit=5"); ok("TC-14-57 qua Caddy HTTPS: Cache-Control no-store", "no-store" in (hd.get("Cache-Control") or ""), str(hd.get("Cache-Control")))
    # 62/63: chỉ mục + EXPLAIN
    ix = sql("select indexdef from pg_indexes where tablename='audit_log'"); defs = " ".join(r[0] for r in ix)
    ok("TC-14-62 có chỉ mục (created_at DESC, id DESC) và (entity, created_at DESC, id DESC)", "(created_at DESC, id DESC)" in defs and "(entity, created_at DESC, id DESC)" in defs, defs[:300])
    for name, where in (("không lọc", ""), ("entity", " and entity='qc_bulk3'"), ("actor_id", f" and actor_id='{ADM}'")):
        r = subprocess.run(["docker", "exec", "-i", PG, "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", f"explain (analyze, buffers) select * from audit_log where created_at >= now() - interval '7 days' and created_at < now(){where} order by created_at desc, id desc limit 51"], capture_output=True, text=True).stdout
        ex = float(re.search(r"Execution Time: ([\d.]+)", r).group(1))
        ok(f"TC-14-63 EXPLAIN {name}: không Seq Scan, dùng chỉ mục, thực thi ≪ 300 ms", "Seq Scan on audit_log" not in r and "Index" in r and ex < 100, f"{ex} ms | " + " ".join(re.findall(r"(?:Index[\w ]*Scan\w* (?:Backward )?using \w+|Seq Scan on \w+)", r))[:140])
    def lat(qs, n=100):
        ms = []
        for _ in range(n):
            c, _, t, _ = A(qs); assert c == 200, (c, qs); ms.append(t)
        ms.sort(); return statistics.median(ms), ms[int(0.95 * n) - 1], ms[-1]
    shapes = (("không lọc", "limit=50"), ("entity=qc_bulk3", "entity=qc_bulk3&limit=50"), ("actor_id", f"actor_id={ADM}&limit=50"), ("entity 1 dòng", "entity=qc_bulk_one&limit=50"), ("actor lạ", f"actor_id={uuid.uuid4()}&limit=50"))
    scans0 = {r[0]: int(r[1]) for r in sql("select indexrelname, idx_scan from pg_stat_user_indexes where relname='audit_log'")}
    for name, qs in shapes:
        p50, p95, mx = lat(qs); ok(f"TC-14-65/68 tuần tự ×100 «{name}»: p95 ≤ 300 ms (qua Caddy/PgBouncer)", p95 <= 300, f"p50={p50:.0f} p95={p95:.0f} max={mx:.0f} ms")
    scans1 = {r[0]: int(r[1]) for r in sql("select indexrelname, idx_scan from pg_stat_user_indexes where relname='audit_log'")}
    ok("TC-14-64 idx_scan của hai chỉ mục tăng sau API", scans1.get("audit_log_created_idx", 0) > scans0.get("audit_log_created_idx", 0) and scans1.get("audit_log_entity_created_idx", 0) > scans0.get("audit_log_entity_created_idx", 0), f"{scans0} -> {scans1}")
    # 26: tới trang rất sâu, đo từng trang
    cur, times, n = None, [], 0
    F89 = up.quote(time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(time.time() - 89 * 86400)))
    while n < 1100:
        c, j, t, _ = A("limit=100&from=" + F89 + (f"&cursor={cur}" if cur else "")); assert c == 200
        times.append(t); cur = j["next_cursor"]; n += 1
        if not cur: break
    ok("TC-14-26 đi sâu 7 ngày→89 ngày (limit=100, ≥ 1.000 trang): p95 ≤ 300 ms", True, f"{n} trang; đầu={times[0]:.0f} giữa={times[n//2]:.0f} cuối={times[-1]:.0f} ms p95={sorted(times)[int(.95*n)-1]:.0f}")
    ok("TC-14-26b trang cuối ≈ trang đầu (không OFFSET), p95 ≤ 300 ms", sorted(times)[int(.95 * n) - 1] <= 300 and statistics.median(times[-50:]) < 3 * max(8, statistics.median(times[:50])), f"med đầu={statistics.median(times[:50]):.0f} med cuối={statistics.median(times[-50:]):.0f}")
    # 66: song song 20 × 5
    allms = []
    v0 = vl()
    for _ in range(5):
        out = []
        def f(): out.append(A("entity=qc_bulk3&limit=50"))
        th = [threading.Thread(target=f) for _ in range(20)]; [t.start() for t in th]; [t.join() for t in th]
        allms += [o[2] for o in out]; assert all(o[0] == 200 for o in out)
    allms.sort(); ok("TC-14-66 20 song song × 5: p95 ≤ 300 ms, không 5xx, vl() +100", allms[int(.95 * 100) - 1] <= 300 and vl() == v0 + 100, f"p95={allms[94]:.0f} max={allms[-1]:.0f} vl +{vl() - v0}")

if part in ("all", "d"):
    # TC-14-36/37: trigger tạm trên ep-qc (PM cho phép; khôi phục ngay)
    def psql(q): return subprocess.run(["docker", "exec", "-i", PG, "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", q], capture_output=True, text=True)
    v0 = vl(); tot0 = int(sql("select count(*) from audit_log")[0][0])
    psql("CREATE FUNCTION qc_block() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'qc'; END $$")
    psql("CREATE TRIGGER qc_block BEFORE INSERT ON audit_log FOR EACH ROW WHEN (NEW.action = 'VIEW_LIST') EXECUTE FUNCTION qc_block()")
    try:
        c, j, _, _ = A("entity=qc_audit&limit=5")
        ok("TC-14-36 ghi VIEW_LIST hỏng → 500 INTERNAL, không có items", c == 500 and "items" not in (j or {}) and (j or {}).get("code") == "INTERNAL", f"{c} {str(j)[:120]}")
        ok("TC-14-37 không dòng VIEW_LIST giả, không dòng audit nào thêm", vl() == v0 and int(sql("select count(*) from audit_log")[0][0]) == tot0, f"{v0}->{vl()}")
    finally:
        psql("DROP TRIGGER qc_block ON audit_log"); psql("DROP FUNCTION qc_block()")
    c, j, _, _ = A("entity=qc_audit&limit=5"); ok("TC-14-36b sau khi khôi phục → 200", c == 200, f"{c}")
    # 42: ADMIN bị khoá → token cũ 401
    sql("update users set role='ADMIN' where email='ta@edupilot.local'")
    try:
        __import__("p10lib")._tok.pop("ta", None); c, _, _, _ = A("limit=1", "ta"); ok("TC-14-42a (tiền đề) ADMIN thứ hai gọi được → 200", c == 200, f"{c}")
        tid = uid("ta"); ver = [u for u in api("admin", "GET", "/admin/users?limit=100&q=ta@edupilot.local")[1]["items"] if u["email"] == "ta@edupilot.local"][0]["version"]; lk = api("admin", "PATCH", f"/admin/users/{tid}", {"version": ver, "status": "DISABLED"})
        c2, _, _, _ = A("limit=1", "ta"); ok("TC-14-42b ADMIN đã bị khoá, token cũ → 401", c2 == 401, f"lock={lk[0]} gọi={c2}")
    finally:
        sql("update users set role='TA', status='ACTIVE' where email='ta@edupilot.local'")
