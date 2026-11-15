# QC2-I slice r4ui: dựng tài khoản + lớp RIÊNG (tiền tố qc2iu) trên ep-r. Chạy lại được (bỏ qua cái đã có). Ghi trạng thái /tmp/qc2-i/qc2iu-state.json
import os, sys, json, uuid, time
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', '..', 'evidence-3', 'scripts'))
os.environ.setdefault('SP', 'qc2iu')
from lib import *
ST = '/tmp/qc2-i/qc2iu-state.json'
PWD = 'Qc2iu!Pass-2026x'
state = json.load(open(ST)) if os.path.exists(ST) else {}
def save(): json.dump(state, open(ST, 'w'), ensure_ascii=False, indent=1)
def iso(): return time.strftime('%Y-%m-%dT%H:%M:%S', time.gmtime(time.time() - 3))
resetrl(); A = login('admin')
def lg(em):
    resetrl(); s, j, _ = call('POST', '/auth/login', None, {'email': em, 'password': PWD}); assert s == 200, (em, s, j); return j['access_token']
def staff(key, name, role):
    em = f'qc2iu.{key}@edupilot.local'
    if key in state: return
    t0 = iso(); s, j, _ = call('POST', '/admin/users', A, {'email': em, 'full_name': name, 'role': role}); assert s == 201, (s, j)
    m = mail_to(em, 'mời|Lời mời', t0, 40); resetrl(); s2, j2, _ = call('POST', '/auth/accept-invite', None, {'token': link_token(m, 'invite'), 'password': PWD}); assert s2 == 200, (s2, j2)
    state[key] = {'email': em, 'id': j['id'], 'name': name}; save()
def student(key, name, code=''):
    em = f'qc2iu.{key}@edupilot.local'
    if key in state: return
    resetrl(); t0 = iso(); s, j, _ = call('POST', '/auth/register', None, {'email': em, 'password': PWD, 'full_name': name, 'student_code': code}); assert s == 202, (s, j)
    m = mail_to(em, 'Xác minh', t0, 40); call('POST', '/auth/verify-email', None, {'token': link_token(m, 'verify-email')})
    state[key] = {'email': em, 'name': name}; save()
staff('gv1', 'Giảng viên QC2IU Một', 'TEACHER'); staff('gv2', 'Giảng viên QC2IU Hai', 'TEACHER'); staff('ta', 'Trợ giảng QC2IU', 'TA'); staff('adm', 'Quản trị QC2IU', 'TEACHER')
psql(f"update users set role='ADMIN' where email='{state['adm']['email']}'")  # tài khoản Admin riêng (Admin chỉ mời được GV/TA)
for i, n in enumerate(['An', 'Bình', 'Cường', 'Dũng'], 1): student(f'sv{i}', f'SV QC2IU {n}')  # vào lớp A (ACTIVE)
student('svd', 'SV QC2IU Dư')  # "SV D": chưa vào lớp nào
for i, n in enumerate(['Giang', 'Hà', 'Khoa'], 5): student(f'sv{i}', f'SV QC2IU {n}')  # PENDING (lớp bật duyệt)
student('svm', 'SV QC2IU Lệch', 'QC2IU0001')  # MSSV trùng roster của người khác → EMAIL_MISMATCH
if 'courseA' not in state:
    s, c, _ = call('POST', '/admin/courses', A, {'subject_code': 'INT1006', 'class_code': 'QC2IUA', 'name': 'QC2IU lớp A', 'semester': '2026-2027-HK1', 'teacher_id': state['gv1']['id']}); assert s == 201, (s, c)
    call('POST', f"/admin/courses/{c['id']}/assign", A, {'ta_ids': [state['ta']['id']]})
    state['courseA'] = {'id': c['id'], 'code': 'QC2IUA'}; save()
cid = state['courseA']['id']; GV = lg(state['gv1']['email'])
jc = call('GET', f'/courses/{cid}/join-code', GV)[1]; state['courseA']['join_code'] = jc['join_code']; code = jc['join_code']
for i in range(1, 5):
    em = state[f'sv{i}']['email']
    if psql(f"select count(*) from enrollments e join users u on u.id=e.user_id where u.email='{em}' and e.course_id='{cid}'") == '0':
        call('POST', '/courses/join', lg(em), {'code': code})
# roster 1 dòng (MSSV QC2IU0001) để svm bị EMAIL_MISMATCH khi vào lớp
def multipart(data, fname):
    b = '----qc2iu' + uuid.uuid4().hex
    return (f'--{b}\r\nContent-Disposition: form-data; name="file"; filename="{fname}"\r\nContent-Type: text/csv\r\n\r\n').encode() + data + f'\r\n--{b}--\r\n'.encode(), f'multipart/form-data; boundary={b}'
if 'roster' not in state:
    body, ct = multipart('Email,Họ và tên,MSSV\nqc2iu.roster1@example.test,Người Roster Một,QC2IU0001\n'.encode(), 'r.csv')
    s, j, _ = call('POST', f'/courses/{cid}/roster/import', GV, raw=body, hdr={'content-type': ct, 'Idempotency-Key': str(uuid.uuid4())}, idem=False); print('roster', s, j); state['roster'] = s; save()
cur = call('GET', f'/courses/{cid}/join-code', GV)[1]
r = call('PUT', f'/courses/{cid}/join-settings', GV, {'require_approval': True, 'version': cur.get('version', 1)}); print('require_approval', r[0], r[1])
for k in ['sv5', 'sv6', 'sv7', 'svm']:
    em = state[k]['email']
    if psql(f"select count(*) from enrollments e join users u on u.id=e.user_id where u.email='{em}' and e.course_id='{cid}'") == '0':
        print(k, call('POST', '/courses/join', lg(em), {'code': code})[:2])
print(psql(f"select u.email,e.status,e.warning from enrollments e join users u on u.id=e.user_id where e.course_id='{cid}' order by 1"))
save(); print(json.dumps(state, ensure_ascii=False)[:600])
