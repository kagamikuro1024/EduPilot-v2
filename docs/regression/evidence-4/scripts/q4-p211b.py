import sys, json
sys.path.insert(0, '.'); from q4lib import *
src = open('q4-p211a.py').read().split("T_ = L('teacher')")[0]; exec(src)
T_ = L('teacher')
c1, cc1, code1 = new_course('i1'); c2, cc2, code2 = new_course('i2'); settings(c1); settings(c2)
uB = student_pool(10)[8]; tB = L(uB); bcode = psql("select student_code from users where email='%s@edupilot.local'" % uB); resetjoin()
uA = student_pool(10)[9]; tA = L(uA)
for tk, cd in [(tA, code1), (tA, code2), (tB, code1)]: call('POST', '/courses/join', tk, {'code': cd})
emX, tX = mk('ix', bcode); resetjoin(); call('POST', '/courses/join', tX, {'code': code1})     # X: PENDING (MSSV trùng B)
emD, tD = mk('id')
people = {'A': (uA + '@edupilot.local', tA, [c1, c2]), 'B': (uB + '@edupilot.local', tB, [c1]), 'X': (emX, tX, [c1]), 'D': (emD, tD, [])}
ids = {k: psql("select id||'|'||full_name||'|'||coalesce(student_code,'') from users where email='%s'" % v[0]).split('|') for k, v in people.items()}
bad = []; summary = []
for k, (em, tk, cs) in people.items():
    bodies = {'me/today': call('GET', '/me/today', tk)}
    for c in cs: bodies[f'courses/{c[:8]}/today'] = call('GET', f'/courses/{c}/today', tk)
    for other in people:
        if other == k: continue
        uid, name, code = ids[other]; em_o = people[other][0]
        for nm, r in bodies.items():
            blob = json.dumps(r[1], ensure_ascii=False)
            for lab, v in [('user_id', uid), ('tên', name), ('email', em_o), ('MSSV', code)]:
                if v and v in blob: bad.append((k, nm, other, lab))
    for nm, r in bodies.items():
        blob = json.dumps(r[1]); 
        for cd in (code1, code2):
            if cd in blob: bad.append((k, nm, 'join_code', cd))
        for c in (c1, c2):
            if c not in cs and c in blob: bad.append((k, nm, 'lớp ngoài', c[:8]))
    summary.append(f'{k}: ' + ', '.join(f'{nm}={r[0]}' for nm, r in bodies.items()))
    # X (PENDING) không đọc today của lớp
tx_course = call('GET', f'/courses/{c1}/today', tX)
D_c = call('GET', f'/courses/{c1}/today', tD)
C('TC-P211-18', not bad and D_c[0] == 403, f'4 người (A: 2 lớp; B: lớp 1; X: PENDING vì MSSV trùng B; D: chưa có lớp) gọi me/today + courses/<lớp mình>/today: {"; ".join(summary)}; quét user_id, họ tên, email, MSSV của người khác, join_code, id lớp ngoài trong mọi thân: vi phạm = {bad}; X vào today lớp đang chờ → {tx_course[0]}; D vào today lớp lạ → {D_c[0]}')
