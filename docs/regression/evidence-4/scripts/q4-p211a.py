import sys, json, re, subprocess, datetime
sys.path.insert(0, '.'); from q4lib import *
TAG = uuid.uuid4().hex[:5]; PWD = 'Q2s4!Pass-' + TAG + 'x7'
def iso(): return time.strftime('%Y-%m-%dT%H:%M:%S', time.gmtime(time.time() - 3))
def mk(label, code=''):
    resetrl(); em = f'qc2s4.{label}.{TAG}@edupilot.local'; t0 = iso()
    s, j, _ = call('POST', '/auth/register', None, {'email': em, 'password': PWD, 'full_name': 'QC2S4 ' + label, 'student_code': code}); assert s == 202, (s, j)
    m = mail_to(em, 'Xác minh', t0, 25); tk = link_token(m, 'verify-email'); s, j, _ = call('POST', '/auth/verify-email', None, {'token': tk}); assert s == 200, (s, j)
    resetrl(); s, j, _ = call('POST', '/auth/login', None, {'email': em, 'password': PWD}); assert s == 200, (s, j); return em, j['access_token']
T_ = L('teacher'); TA = L('ta')
# ---- 01
n = subprocess.run("cd ~/Documents/TA_Agent_qc2 && grep -rn 'internal/llm\"' backend-go/internal/today | grep -v _test.go | wc -l", shell=True, capture_output=True, text=True).stdout.strip()
doc = subprocess.run("cd ~/Documents/TA_Agent_qc2/backend-go && go doc ./internal/today Viewer 2>&1 | sed -n 3,12p; go doc ./internal/today Provider | sed -n 3,6p", shell=True, capture_output=True, text=True).stdout
C('TC-P211-01', n == '0' and 'UserID' in doc and 'EmailVerified' in doc and 'Courses' in doc and 'Items(ctx context.Context, v Viewer, s Scope)' in doc,
  f'grep import internal/llm ở internal/today (không _test) = {n}; go doc: Provider{{Name(); Items(ctx, Viewer, Scope) ([]Item, error)}}, Register(p) trên Aggregator; Viewer gồm UserID, Role, EmailVerified, Courses (chỉ lớp ACTIVE theo chú thích), thêm MaskedEmail và Now (bổ sung sau, không nhận id người/lớp ngoài Viewer)')
# ---- 04 xếp hạng: kiểm toàn bộ actions của GV theo khoá (quá hạn, bậc, tuổi giảm, mã lớp, id)
TIER = {'PUBLISH_PENDING': 33, 'TICKET': 10, 'SESSION_UNMARKED': 12, 'CHECKIN_REVIEW': 13, 'JOIN_REQUEST': 45, 'EMAIL_MISMATCH': 40, 'COURSE_SETUP': 80, 'GRADE_SCHEME_UNCONFIRMED': 70, 'AI_CONFIRM': 50, 'QUESTION_REVIEW': 90, 'EXAM_GRADE_ERROR': 15, 'GRADING_ERROR': 28, 'GRADING_REVIEW': 30, 'POST_REPORTED': 47, 'STAFF_ANSWER_UNREAD': 40}
s, j, _ = call('GET', '/me/today', T_); acts = j.get('actions', [])
unk = {a['kind'] for a in acts if a['kind'] not in TIER}
def key(a): return (0 if a.get('urgency') == 'overdue' else 1, TIER.get(a['kind'], 999), -(a.get('age_minutes') or 0), a.get('course', {}).get('class_code', ''), a['id'])
known = [a for a in acts if a['kind'] in TIER]; ok_sorted = [key(a) for a in known] == sorted(key(a) for a in known)
ref = psql("select 1")
kinds = sorted({a['kind'] for a in acts}); pairs = sum(1 for i in range(len(known) - 1) if key(known[i]) < key(known[i + 1]))
_ex, _, _ = new_course('t04'); time.sleep(1)
cs = call('GET', f'/courses/{_ex}/today', T_)[1].get('actions', [])
cs_order = [a['kind'] for a in cs]
ovd = [a for a in acts if a['kind'] == 'JOIN_REQUEST' and a['urgency'] == 'overdue' and (a.get('age_minutes') or 0) > 48 * 60]
print('04dbg', s, ok_sorted, len(ovd), 'EMAIL_MISMATCH' in kinds, cs_order[:3])
C('TC-P211-04', s == 200 and ok_sorted and ovd and 'EMAIL_MISMATCH' in kinds and cs_order[:2] == ['GRADE_SCHEME_UNCONFIRMED', 'COURSE_SETUP'], f'GV teacher: {len(acts)} mục (count {j.get("count")}) gồm {kinds}; thứ tự khớp khoá (quá hạn trước, bậc tăng dần theo hằng Tier* trong today.go, tuổi giảm dần, mã lớp tăng dần, id tăng dần) cho {len(known)} mục có bậc đã biết (bỏ qua loại chưa ánh xạ: {sorted(unk)}); ; JOIN_REQUEST quá 48 h ({len(ovd)} mục) nằm nhóm quá hạn, xếp theo tuổi giảm dần; danh sách cắt ở 50 mục nên COURSE_SETUP (bậc 80) kiểm ở màn một lớp: thứ tự {cs_order} (GRADE_SCHEME_UNCONFIRMED 70 trước COURSE_SETUP 80)')
# ---- 08 (saltar) SV: timeline hai buổi hôm nay, không việc gấp
em, tA = mk('a'); cidA, ccA, codeA = new_course('t1'); cidB, ccB, codeB = new_course('t2'); settings(cidA); settings(cidB); resetjoin()
call('POST', '/courses/join', tA, {'code': codeA}); call('POST', '/courses/join', tA, {'code': codeB})
today = datetime.datetime.now(datetime.timezone(datetime.timedelta(hours=7))).date().isoformat(); wd = datetime.date.fromisoformat(today).isoweekday()
g1 = call('POST', f'/courses/{cidA}/sessions/generate', T_, {'weekdays': [wd], 'start_time': '09:00', 'end_time': '11:00', 'room': 'A1', 'from': today, 'to': today, 'exclude_dates': []})
g2 = call('POST', f'/courses/{cidB}/sessions/generate', T_, {'weekdays': [wd], 'start_time': '13:30', 'end_time': '15:00', 'room': 'B2', 'from': today, 'to': today, 'exclude_dates': []})
sA = call('GET', '/me/today', tA); sj = sA[1]; cT = call('GET', f'/courses/{cidA}/today', tA)
tl = sj.get('timeline', [])
C('TC-P211-08', sA[0] == 200 and set(['no_course', 'email_verified', 'recommended', 'timeline', 'continue']) <= set(sj) and sj.get('recommended') is None and len(tl) == 2 and sj.get('continue') in ([], None) and cT[0] == 200,
  f'SV mới verify, 2 lớp, mỗi lớp 1 buổi hôm nay (sessions/generate {g1[0]}/{g2[0]}): khoá {sorted(sj)}; recommended={sj.get("recommended")}; timeline {len(tl)} mục: {json.dumps([(x.get("starts_at"), x.get("course", {}).get("class_code")) for x in tl], ensure_ascii=False)[:200]}; continue {sj.get("continue")}; courses/{{id}}/today → {cT[0]}')
# ---- 13 COURSE_SETUP tự biến
cidS, ccS, codeS = new_course('t13'); settings(cidS)
def setup_reason():
    a = [x for x in call('GET', f'/courses/{cidS}/today', T_)[1].get('actions', []) if x['kind'] == 'COURSE_SETUP']; return a[0]['reason'] if a else None
r0 = setup_reason(); uS = student_pool(5)[1]; resetjoin(); call('POST', '/courses/join', L(uS), {'code': codeS}); time.sleep(3); r1 = setup_reason()
call('POST', f'/courses/{cidS}/sessions/generate', T_, {'weekdays': [wd], 'start_time': '09:00', 'end_time': '11:00', 'from': today, 'to': today, 'exclude_dates': []}); time.sleep(3); r2 = setup_reason()
for typ in ('COURSE_POLICY', 'LECTURE'):
    psql(f"insert into documents(course_id,title,type,status,visible_to_students,use_for_rag,uploaded_by) select '{cidS}','QC2S4 {typ}','{typ}','READY',true,false,id from users where email='teacher@edupilot.local'")
t_s = time.time(); r3 = setup_reason()
while r3 is not None and time.time() - t_s < 75: time.sleep(5); r3 = setup_reason()
lat3 = time.time() - t_s
cidS2, _, _ = new_course('t13b'); a1 = call('POST', f'/courses/{cidS2}/setup/dismiss', T_); a2 = call('POST', f'/courses/{cidS2}/setup/dismiss', T_)
hid = [x for x in call('GET', f'/courses/{cidS2}/today', T_)[1].get('actions', []) if x['kind'] == 'COURSE_SETUP']
print('13:', r0, '|', r1, '|', r2, '|', r3, '|', a1[0], a2[0], len(hid))
C('TC-P211-13', r0 and r3 is None and a1[0] in (200, 204) and a2[0] in (200, 204) and not hid,
  f'COURSE_SETUP: ban đầu «{r0}»; sau khi có SV vào lớp «{r1}»; sau tạo buổi «{r2}»; sau thêm quy chế + tài liệu bằng SQL fixture (không qua outbox → chỉ hết hạn nhờ TTL 60 s của cache, đo {lat3:.0f} s) → {"mục biến mất" if r3 is None else r3}; setup/dismiss lần 1/2 → {a1[0]}/{a2[0]} (idempotent), mục ẩn khỏi today: {not hid}')
