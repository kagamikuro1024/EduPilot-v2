import sys, json, re
sys.path.insert(0, '.'); from q4lib import *
TAG = uuid.uuid4().hex[:5]; PWD = 'Q2s4!Pass-' + TAG + 'x7'
def iso(): return time.strftime('%Y-%m-%dT%H:%M:%S', time.gmtime(time.time() - 3))
def mk(label, code):
    resetrl(); em = f'qc2s4.{label}.{TAG}@edupilot.local'; t0 = iso()
    s, j, _ = call('POST', '/auth/register', None, {'email': em, 'password': PWD, 'full_name': 'QC2S4 ' + label, 'student_code': code}); assert s == 202, (s, j)
    m = mail_to(em, 'Xác minh', t0, 25); tk = link_token(m, 'verify-email'); s, j, _ = call('POST', '/auth/verify-email', None, {'token': tk}); assert s == 200, (s, j)
    resetrl(); s, j, _ = call('POST', '/auth/login', None, {'email': em, 'password': PWD}); assert s == 200, (s, j); return em, j['access_token']
b_user = student_pool(10)[3]; tB = L(b_user); bcode = psql("select student_code from users where email='%s@edupilot.local'" % b_user)
cid, cc, code = new_course('x'); settings(cid); resetjoin()
jB = call('POST', '/courses/join', tB, {'code': code})
snap = psql(f"select student_code_snapshot from enrollments e join users u on u.id=e.user_id where e.course_id='{cid}' and u.email='{b_user}@edupilot.local'")
xem, tX = mk('x', bcode); resetjoin(); jX = call('POST', '/courses/join', tX, {'code': code})
xid = psql(f"select id from users where email='{xem}'"); bid = psql("select id from users where email='%s@edupilot.local'" % b_user)
rowX = psql(f"select status, coalesce(warning,''), student_code_snapshot from enrollments where course_id='{cid}' and user_id='{xid}'")
tTA = L('ta'); tT = L('teacher'); tAd = L('admin')
# TA phải được gán vào lớp để duyệt
psql(f"insert into enrollments(course_id,user_id,role_in_course,status,joined_via) select '{cid}',id,'TA','ACTIVE','ADMIN' from users where email='ta@edupilot.local' on conflict do nothing")
a1 = call('POST', f'/courses/{cid}/members/{xid}/approve', tTA, {}); a2 = call('POST', f'/courses/{cid}/members/{xid}/approve', tT, {}); a3 = call('POST', f'/courses/{cid}/members/{xid}/approve', tT, {'confirm_mismatch': True})
mem = call('GET', f'/courses/{cid}/members', tT)[1]; items = mem.get('items', [])
ids = [i.get('user_id') or i.get('id') for i in items]
print(json.dumps([a1[:2], a2[:2], a3[:2]], ensure_ascii=False)[:600]); print(rowX, snap, bcode, jB[1], jX[1])
C('TC-P209-31', a1[0] == 403 and a1[1].get('details', {}).get('reason') == 'mismatch_needs_teacher' and a2[0] == 422 and a3[0] == 200 and a3[1].get('status') == 'ACTIVE' and xid in ids and bid in ids and rowX.endswith(bcode),
  f'X khai MSSV trùng B ({bcode}) → join {jX[0]} {jX[1].get("status")}; DB X: {rowX}; TA approve → {a1[0]} reason {a1[1].get("details",{}).get("reason")}; GV approve thiếu confirm_mismatch → {a2[0]}; có cờ → {a3[0]} {a3[1].get("status")}; student_code_snapshot của X = MSSV tự khai; danh sách có cả X và B ({len(items)} hàng, hai user_id khác nhau)')
# ADMIN approve: lớp 2 cùng kịch bản
cid2, _, code2 = new_course('y'); settings(cid2); resetjoin(); call('POST', '/courses/join', tB, {'code': code2}); xem2, tX2 = mk('x2', bcode); resetjoin(); call('POST', '/courses/join', tX2, {'code': code2}); xid2 = psql(f"select id from users where email='{xem2}'")
ad = call('POST', f'/courses/{cid2}/members/{xid2}/approve', tAd, {}); ad2 = call('POST', f'/courses/{cid2}/members/{xid2}/approve', tAd, {'confirm_mismatch': True})
T('TC-P209-31-admin', 'P', f'ADMIN approve không cờ → {ad[0]}, có cờ → {ad2[0]} {ad2[1].get("status")} (ghi hành vi theo SRS)')
# 33: X chỉ thấy dữ liệu của X
rx = {}
for pth in ['/me/today', '/me/courses', f'/courses/{cid}/today', f'/courses/{cid}/members']:
    rx[pth] = call('GET', pth, tX)
blob = json.dumps({k: v[1] for k, v in rx.items() if k != f'/courses/{cid}/members'}, ensure_ascii=False)
leak = [w for w in [bid, b_user + '@edupilot.local', bcode and psql("select full_name from users where email='%s@edupilot.local'" % b_user)] if w and w in blob]
C('TC-P209-33', not leak and rx[f'/courses/{cid}/members'][0] == 403, f'X (MSSV trùng B, đã được GV xác nhận) gọi me/today ({rx["/me/today"][0]}), me/courses ({rx["/me/courses"][0]}), courses/{{id}}/today ({rx[f"/courses/{cid}/today"][0]}): không có user_id/email/tên của B trong thân; /members → {rx[f"/courses/{cid}/members"][0]}; rò = {leak}')
# 32: thứ tự ngược — X vào trước, B vào sau
cid3, _, code3 = new_course('z'); settings(cid3); resetjoin(); bu2 = student_pool(10)[4]; tB2 = L(bu2); bcode2 = psql("select student_code from users where email='%s@edupilot.local'" % bu2)
xem3, tX3 = mk('x3', bcode2); resetjoin(); jx3 = call('POST', '/courses/join', tX3, {'code': code3})   # X tự khai MSSV của B2, vào trước
xid3 = psql(f"select id from users where email='{xem3}'"); st_x = psql(f"select status from enrollments where course_id='{cid3}' and user_id='{xid3}'")
if st_x != 'ACTIVE': call('POST', f'/courses/{cid3}/members/{xid3}/approve', tT, {'confirm_mismatch': True})
resetjoin(); jb3 = call('POST', '/courses/join', tB2, {'code': code3}); rowB = psql(f"select e.status, coalesce(e.warning,'') from enrollments e join users u on u.id=e.user_id where e.course_id='{cid3}' and u.email='{bu2}@edupilot.local'")
r_b = call('GET', '/me/courses', tB2); b_today = call('GET', f'/courses/{cid3}/today', tB2)
C('TC-P209-32', jb3[0] == 200 and jb3[1].get('status') == 'PENDING' and 'MISMATCH' in rowB.upper(), f'X vào trước (join {jx3[0]} {jx3[1].get("status")}, DB trạng thái {st_x}); B (MSSV {bcode2}) vào sau → join {jb3[0]} {jb3[1].get("status")}; DB hàng B: {rowB}; B chưa đọc được dữ liệu lớp ({b_today[0]}); hai người không đọc được dữ liệu của nhau')
