# s8r 08c: QD8R8 LOCKED v1 → mở khoá → sửa 2 ô → chốt v2 → thông báo → v3 (08-16/18/21..24/26/28/47, 07-24, 09-04/38/41, 08-41)
from s8r_g import *
S1 = snap(1); H1 = h(1); sv4, sv8 = "sv04", "sv08"
REASON = "Phúc khảo sv04: sửa điểm giữa kỳ"
a0 = int(sq1(f"select count(*) from audit_log where course_id='{C0}' and action='unlock'"))
c, r = unl(REASON)
g = sql(f"select status, unlock_reason, unlocked_by, unlocked_at is not null from gradebooks where course_id='{C0}'")[0]
a1 = int(sq1(f"select count(*) from audit_log where course_id='{C0}' and action='unlock'"))
C("TC-P6-08-16", c == 200 and g[0] == "OPEN" and g[1] == REASON and g[2] == TCH and g[3] == "t" and a1 == a0 + 1 and h(1) == H1, f"unlock → 200; gradebooks OPEN, unlock_reason đúng, unlocked_by=teacher, unlocked_at có; audit unlock +1; SNAP(1) không đổi", f"{c} {r} {g} {a0}->{a1}")
c, r = unl("Lý do mở khoá thứ hai"); C("TC-P6-08-18", c == 409 and r.get("code") == "NOT_LOCKED", f"unlock khi OPEN → 409 {r.get('code')}", f"{c} {r}")
# 08-21
n_before = notif("GRADE_CHANGED")
s8, v8 = cell(sv8, "final"); s4, v4 = cell(sv4, "midterm")
ca, ra = put(sv8, "final", "9.25", v8); cb, rb = put(sv4, "midterm", "7.25", v4)
c, p = call("GET", f"{Q}/gradebook/preview?limit=100", "teacher"); P = {x["student_id"]: x for x in p["items"]}
h8, h4 = P[SV(sv8)], P[SV(sv4)]
C("TC-P6-08-21", ca == 200 and cb == 200 and (h8["hp"], h8["letter"]) == ("9", "A+") and (h4["hp"], h4["letter"]) == ("6.5", "C+") and (s8, s4) == ("8.25", "6.25"), f"sv08 cuối kỳ {s8}→9,25 (200) xem trước HP {h8['hp']} {h8['letter']} (tay 9,0 A+); sv04 giữa kỳ {s4}→7,25 (200) HP {h4['hp']} {h4['letter']} (tay 6,5 C+ không đổi)", f"{ca} {cb} {h8} {h4} {s8} {s4}")
# 08-22
c, r = fin()
S2 = snap(2); ur = {x[4] for x in S2}
C("TC-P6-08-22", c == 200 and r.get("version") == 2 and len(S2) == 30 and ur == {REASON} and gb()[:2] == ["LOCKED", "2"], f"finalize lần hai → {c} version {r.get('version')}; snapshot v2 {len(S2)} dòng, unlock_reason = '{list(ur)[0] if ur else ''}'; gradebooks LOCKED v2", f"{c} {r} {len(S2)} {ur} {gb()}")
T("TC-P6-08-23", "P" if h(1) == H1 else "F", f"băm md5 SNAP(1) trước = sau chốt lại v2: {H1[:8]} = {h(1)[:8]}")
# 08-24
c, g2 = call("GET", f"{Q}/gradebook?limit=100", "teacher"); st, b, hd = getx(f"{Q}/gradebook/export.xlsx", "teacher"); rx, nf, _ = xlsx_rows(b)
ihp = rx[1].index("Điểm học phần"); il = rx[1].index("Điểm chữ"); r8 = [r for r in rx[2:] if r[0] == CSV[sv8]["student_code"]][0]
cs, ls = call("GET", f"{Q}/gradebook/snapshots", "teacher"); c1, s1 = call("GET", f"{Q}/gradebook/snapshots/1?limit=100", "teacher"); c2, s2 = call("GET", f"{Q}/gradebook/snapshots/2?limit=100", "teacher")
def srow(js, sid): return [x for x in js["items"] if x.get("student_id") == sid or x.get("student", {}).get("id") == sid]
print("snapshots list", json.dumps(ls)[:300]); print("snap1 item", json.dumps(s1)[:300])
C("TC-P6-08-24", g2["state"]["snapshot_version"] == 2 and "Bản chốt v2" in " ".join(rx[0]) and (r8[ihp], r8[il]) == ("9", "A+") and cs == 200 and c1 == 200, f"gradebook state LOCKED v2; export hàng đầu '{' '.join(rx[0])[-20:]}', sv08 {r8[ihp]} {r8[il]}; snapshots list {cs}, snapshots/1 {c1}; UI 'Lịch sử chốt' + chọn v1 (8,5 A): s8r-ui-08", f"{g2['state']} {rx[0]} {r8}")
# 08-26 GRADE_CHANGED
chg = sql(f"select u.student_code, n.title from notifications n join users u on u.id=n.user_id where n.type='GRADE_CHANGED' and n.course_id='{C0}'")
C("TC-P6-08-26", len(chg) == 1 and chg[0][0] == CSV[sv8]["student_code"] and not re.search(r"\d[,.]\d|A\+", chg[0][1]), f"GRADE_CHANGED trong lớp: {len(chg)} thông báo (sv08 mã {chg[0][0] if chg else None}); nội dung '{chg[0][1] if chg else ''}' không có số/chữ điểm; sv04 (HP không đổi) và 28 SV khác không nhận", f"{chg}")
# 08-28 kênh
mm = raw("GET", "http://localhost:8725/api/v1/messages?limit=200", {})[1].decode()
mail_grade = len(re.findall(r"điểm học phần", mm, re.I)); 
C("TC-P6-08-28", mail_grade == 0, f"Mailpit: 0 thư có 'điểm học phần'; thông báo chỉ nằm bảng notifications (chuông); mục 'Hôm nay' sinh viên: kiểm ở 09/UI", f"mail {mail_grade}")
# 07-24 import khi LOCKED (upload thật) -> làm ở 07
# 08-47: v2→mở→chốt v3
H2 = h(2); c, r = unl("Lần mở thứ hai để chốt v3 QC"); c3, r3 = fin(); S3 = snap(3)
C("TC-P6-08-47", c == 200 and c3 == 200 and r3.get("version") == 3 and {x[4] for x in S3} == {"Lần mở thứ hai để chốt v3 QC"} and h(1) == H1 and h(2) == H2 and sql(f"select array_agg(distinct version order by version) from final_grade_snapshots where course_id='{C0}'")[0][0] == "{1,2,3}", f"v1→v2→v3 liên tiếp: version 1,2,3 không nhảy/trùng; v3 mang unlock_reason của lần mở trước nó; v1, v2 giữ nguyên băm; (v2 mang '{REASON}')", f"{c} {c3} {r3}")
# 09-38 / 08-41 IDOR bản chốt
A, Bsv = CSV["sv.gioi"]["student_code"], "sv.kha"
res = []
for p in ("snapshots", "snapshots/1", "snapshots/2", "snapshots/3"): res.append(call("GET", f"{Q}/gradebook/{p}", Bsv)[0])
sid_a = sq1(f"select id from final_grade_snapshots where course_id='{C0}' and student_id='{SV('sv.gioi')}' and version=1")
res2 = [call("GET", f"{Q}/gradebook/snapshots/{sid_a}", Bsv)[0], call("GET", f"{Q}/gradebook/snapshots/{sid_a}", "sv.gioi")[0]]
ids = sql(f"select id from final_grade_snapshots where course_id='{C0}' and version=1 order by id limit 20"); res3 = [call("GET", f"{Q}/gradebook/snapshots/{i[0]}", Bsv)[0] for i in ids]
C("TC-P6-08-41", set(res) == {403} and set(res3) <= {403, 404, 422} and set(res2) <= {403, 404, 422}, f"sinh viên B GET snapshots / snapshots/{{1,2,3}} → {res}; đoán snapshot_id (A + 20 id) → {sorted(set(res3+res2))}: không dòng nào lộ", f"{res} {res2} {res3}")
T("TC-P6-09-38", "P" if set(res) == {403} and set(res3) <= {403, 404, 422} else "F", f"sau chốt, B gọi snapshots/1, snapshots/<id của A>, 20 id lân cận → chỉ {sorted(set(res)|set(res3)|set(res2))}; không rò dòng của A")
print("done 08c", gb())
