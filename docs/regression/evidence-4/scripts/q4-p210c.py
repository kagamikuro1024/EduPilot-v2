import sys, json, random
sys.path.insert(0, '.'); from q4lib import *
exec(open('q4-p210a.py').read().split("T_ = L('teacher')")[0].split("from q4lib import *")[1])
T_ = L('teacher'); PWD = 'Q2s4!Pass-' + TAG + 'v9'
cid, cc, code = new_course('v'); settings(cid)
em = f'qc2s4.victim.{TAG}@edupilot.local'; ms = mssv(); t0 = time.strftime('%Y-%m-%dT%H:%M:%S', time.gmtime(time.time() - 3))
r = imp(cid, T_, csv([(em, 'Nạn nhân hộp thư', ms)])); enr0 = psql(f"select e.id from enrollments e join users u on u.id=e.user_id where e.course_id='{cid}' and u.email='{em}'")
m = mail_to(em, 'Bạn được thêm vào lớp', t0, 25); tok = link_token(m, '/invite/') if m else None
resetrl(); ac = call('POST', '/auth/accept-invite', None, {'token': tok, 'password': PWD}) if tok else (0, {}, {})
resetrl(); s, j, _ = call('POST', '/auth/login', None, {'email': em, 'password': PWD}); tk = j.get('access_token')
mc = call('GET', '/me/courses', tk) if tk else (0, {}, {}); today = call('GET', f'/courses/{cid}/today', tk) if tk else (0, {}, {})
enr1 = psql(f"select e.id from enrollments e join users u on u.id=e.user_id where e.course_id='{cid}' and u.email='{em}'"); n = psql(f"select count(*) from enrollments e join users u on u.id=e.user_id where u.email='{em}'")
print('dbg', json.dumps(mc[1], ensure_ascii=False)[:300], enr0, enr1, n)
C('TC-P210-18', r[0] == 200 and tok and ac[0] == 200 and s == 200 and enr0 and enr0 == enr1 and n == '1' and any(i.get('course', {}).get('id') == cid for i in mc[1].get('items', [])) and today[0] == 200,
  f'import → {r[0]}; chủ hộp thư nhận thư mời, accept-invite (token trong liên kết, mật khẩu mới) → {ac[0]}; login → {s}; me/courses có lớp; courses/{{id}}/today {today[0]}; enrollment.id trước/sau accept {"giống" if enr0 == enr1 else "KHÁC"}, tổng bản ghi enrollments của user = {n} (không bản ghi mới)')
