import sys, json
sys.path.insert(0, '.'); from q4lib import *
src = open('q4-p211a.py').read().split("T_ = L('teacher')")[0]; exec(src)
T_ = L('teacher'); pool = student_pool(40)
S = {'pwd': PWD}
cO, ccO, codeO = new_course('uo'); settings(cO); S['O'] = {'id': cO, 'class_code': ccO, 'code': codeO}
cR, ccR, codeR = new_course('ur'); settings(cR, require_approval=True); S['R'] = {'id': cR, 'class_code': ccR, 'code': codeR}
cF, ccF, codeF = new_course('uf'); settings(cF, require_approval=True); S['F'] = {'id': cF, 'class_code': ccF, 'code': codeF}
S['D1'] = mk('ud1')[0]; S['D2'] = mk('ud2')[0]; S['D3'] = mk('ud3')[0]
uB = pool[20]; tB = L(uB); bcode = psql("select student_code from users where email='%s@edupilot.local'" % uB); resetjoin()
# lớp R: B vào (cần duyệt → duyệt), 4 SV chờ duyệt, X trùng MSSV của B
bid = psql("select id from users where email='%s@edupilot.local'" % uB)
call('POST', '/courses/join', tB, {'code': codeR}); call('POST', f'/courses/{cR}/members/{bid}/approve', T_, {})
pend = []
for u in pool[21:25]: resetjoin(); call('POST', '/courses/join', L(u), {'code': codeR}); pend.append(u)
emX, tX = mk('ux', bcode); resetjoin(); call('POST', '/courses/join', tX, {'code': codeR}); S['X'] = emX; S['pend'] = pend; S['B'] = uB
# TA vào lớp R để thử quyền
psql(f"insert into enrollments(course_id,user_id,role_in_course,status,joined_via) select '{cR}',id,'TA','ACTIVE','ADMIN' from users where email='ta@edupilot.local' on conflict do nothing")
json.dump(S, open('/tmp/qc2-s4-ui.json', 'w'), ensure_ascii=False, indent=1); print(json.dumps(S, ensure_ascii=False)[:600])
