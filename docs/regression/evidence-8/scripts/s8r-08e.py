# s8r 08e: QD8R82 — đua chốt (08-30), PUT×finalize 20 lần (08-31), và 06-44 PUT audit fail-closed, 06-53/54 cảnh báo PENALTY/BONUS ngoài công thức
from s8r_g import *
# ---- 06-44 audit fail-closed khi PUT (trigger riêng lớp)
s, v = cell("sv05", "midterm")
sql("create or replace function qc_s8_failaudit() returns trigger language plpgsql as $$ begin raise exception 'QC audit fail'; end $$")
sql(f"create trigger qc_s8_failaudit before insert on audit_log for each row when (new.course_id='{C0}') execute function qc_s8_failaudit()")
c, r = put("sv05", "midterm", "1", v); after = cell("sv05", "midterm")
sql("drop trigger qc_s8_failaudit on audit_log"); sql("drop function qc_s8_failaudit()")
C("TC-P6-06-44", c >= 500 and after == (s, v), f"audit_log chặn INSERT (trigger riêng lớp, thay vì đổi tên bảng dùng chung) → PUT {c}; điểm không ghi: {after} = {(s, v)}; đã gỡ trigger", f"{c} {r} {after}")
# ---- 08-30 hai finalize song song
def f(i): return fin(key=str(uuid.uuid4()))
with cf.ThreadPoolExecutor(2) as ex: rs = list(ex.map(f, range(2)))
codes = sorted(c for c, _ in rs); nv = sql(f"select count(distinct version), count(*) from final_grade_snapshots where course_id='{C0}'")[0]
C("TC-P6-08-30", codes == [200, 409] and [r[1].get("code") for r in rs if r[0] == 409] == ["GRADEBOOK_LOCKED"] and nv == ["1", "30"], f"2 finalize song song (khoá khác nhau) → {codes} (409 = GRADEBOOK_LOCKED); count(distinct version)={nv[0]}, {nv[1]} dòng", f"{rs} {nv}")
# ---- 08-31 PUT × finalize 20 lần
bad = []; outcomes = {"put200": 0, "put409": 0}
for i in range(20):
    unl(f"Vòng thử đua số {i:02d} QC s8r")
    s, v = cell("sv05", "midterm"); newv = "2" if (s or "").startswith("9") else "9"
    with cf.ThreadPoolExecutor(2) as ex:
        fp = ex.submit(fin, str(uuid.uuid4())); fu = ex.submit(put, "sv05", "midterm", newv, v)
        (cf_, rf), (cp, rp) = fp.result(), fu.result()
    ver = gb()[1]; sn = [x for x in snap(int(ver)) if x[0] == SV("sv05")][0]; c, p = call("GET", f"{Q}/gradebook/preview?limit=100", "teacher")
    live = [x for x in p["items"] if x["student_id"] == SV("sv05")][0]["hp"]; cur = cell("sv05", "midterm")[0]
    consistent = cf_ == 200 and abs(float(sn[1]) - float(live)) < 1e-9 and len(snap(int(ver))) == 30
    if cp == 200: outcomes["put200"] += 1; ok = consistent and cur.startswith(newv)
    elif cp == 409 and rp.get("code") == "GRADEBOOK_LOCKED": outcomes["put409"] += 1; ok = consistent and cur == s
    else: ok = False
    if not ok: bad.append((i, cf_, cp, rp.get("code") if isinstance(rp, dict) else rp, sn[1], live))
C("TC-P6-08-31", not bad, f"20 vòng (mở khoá→finalize ∥ PUT sv05): PUT 200 trước chốt {outcomes['put200']} lần (giá trị có trong snapshot), 409 GRADEBOOK_LOCKED {outcomes['put409']} lần (snapshot không có); 20/20 snapshot khớp grade_entries, không nửa mới nửa cũ", f"lệch {bad[:3]}")
