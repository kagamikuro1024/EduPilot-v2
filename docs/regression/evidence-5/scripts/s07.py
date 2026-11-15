import os; os.environ['SUITE']='PE-07'
from r5lib import *
def S(m, p, tk=T, b=None, h=None, **k): return call(m, p, tk, b, h=h, **k)
eid, _ = mkexam(open_s=8, win=3000, dur=40, nq=2, title='R5 07 liêm chính'); EX = f'{E}/{eid}'; time.sleep(10)
U = 'r5s13' ; 
db(f"insert into enrollments(course_id,user_id,role_in_course,status,joined_via,student_code_snapshot) select '{C1}',id,'STUDENT','ACTIVE','ADMIN',student_code from users where email in ('r5s14@edupilot.local','r5s13@edupilot.local') on conflict do nothing")
uid = lambda u: db(f"select id from users where email='{u}@edupilot.local'")
a, j, H, s, _ = start(eid, 'r5s13'); print('start', s, str(_)[:200]); t = login('r5s13'); K = 'ep:exam_lock:' + uid('r5s13')
v = rds('GET', K); ttl = int(rds('TTL', K) or -9); dl = dt.datetime.fromisoformat(j['attempt']['deadline_at'].replace('Z', '+00:00')); rem = (dl - dt.datetime.now(dt.timezone.utc)).total_seconds() + 10
chk('TC-PE07-01', v == a and abs(ttl - rem) <= 3, f'GET {K[-12:]}={v == a and "attempt_id"}; TTL {ttl} s ≈ còn lại {rem:.0f} s (±2 s) = deadline − now + grace')
g1 = S('GET', '/_test/chat-gate', t); gB = S('GET', '/_test/chat-gate', login('r5s14'))
chk('TC-PE07-06', g1[1].get('allowed') is False and gB[1].get('allowed') is True, f'đang làm {g1[1]}; SV không làm bài {gB[1]}')
rds('DEL', K); g2 = S('GET', '/_test/chat-gate', t); back = rds('GET', K)
chk('TC-PE07-07', g2[1].get('allowed') is False and back == a, f'xoá khoá Redis → chat-gate vẫn allowed={g2[1].get("allowed")} (tra DB); khoá nạp lại: {back == a}')
m11 = [S('GET', '/me/exam-lock', t), S('GET', '/me/exam-lock', AD), S('GET', '/me/exam-lock', T), S('GET', '/me/exam-lock', None)]
chk('TC-PE07-11', m11[0][1].get('locked') is True and 'until' in m11[0][1] and set(m11[0][1]) <= {'locked', 'until'} and m11[1][1].get('locked') is False and m11[3][0] == 401, f'SV {m11[0][1]}; ADMIN {m11[1][1]}; GV {m11[2][1]}; ẩn danh {m11[3][0]}')
# chat thật bị chặn
cs = S('POST', '/chat/sessions', t, {}, idem()); sid = cs[1].get('id') or cs[1].get('session', {}).get('id')
cm = S('POST', f'/chat/sessions/{sid}/messages', t, dict(content='Giải thích giúp tôi bài thi này'), idem()) if sid else (0, {}, '')
chk('TC-PE07-47', cm[0] == 409 and 'EXAM_IN_PROGRESS' in codes(cm[1]), f'(chat đã có từ P3, P-7#17: mã EXAM_IN_PROGRESS) tạo phiên {cs[0]}; gửi tin trong giờ thi: {cm[0]} {codes(cm[1])}')
# extend
ttl0 = int(rds('TTL', K)); 
new = (dt.datetime.now(dt.timezone.utc) + dt.timedelta(hours=2)).strftime('%Y-%m-%dT%H:%M:%SZ'); ex = S('POST', f'{EX}/extend', b=dict(closes_at=new)); time.sleep(3)
# events
item = [x for x in j['items']][0]['item_id']
ev = lambda body, tok=t, aid=a: S('POST', EX + f'/attempts/{aid}/events', tok, body, h=H)
e1 = ev(dict(events=[dict(type='PASTE', meta=dict(chars=812, clipboard='SECRET', ip='1.2.3.4', item_id=item)), dict(type='TAB_HIDDEN', meta={}), dict(type='TAB_VISIBLE', meta=dict(duration_ms=5000))]))
rows = db(f"select type,meta::text from exam_events where attempt_id='{a}' order by created_at")
chk('TC-PE07-13', e1[0] == 204 and 'SECRET' not in rows and '1.2.3.4' not in rows and 'chars' in rows, f'{e1[0]}; DB: {rows.replace(chr(10), " ; ")[:200]}')
e51 = ev(dict(events=[dict(type='TAB_HIDDEN', meta={}) for _ in range(51)])); big = ev(dict(events=[dict(type='PASTE', meta=dict(chars=1, item_id='x' * 400))]))
chk('TC-PE07-14', e51[0] in (204, 422) and big[0] in (204, 422), f'51 sự kiện/yêu cầu → {e51[0]} {codes(e51[1])}; meta quá lớn → {big[0]} {codes(big[1])}')
oth = ev(dict(events=[dict(type='TAB_HIDDEN', meta={})]), login('r5s14'), a)
chk('TC-PE07-15', oth[0] == 404, f'sự kiện vào lượt của người khác: {oth[0]}')
# 19 quyền
x19 = [S('GET', EX + f'/events?attempt={a}', T)[0], S('GET', EX + f'/events?attempt={a}', TA)[0], S('GET', EX + f'/events?attempt={a}', t)[0], S('GET', EX + f'/events?attempt={a}', AD)[0], S('GET', EX + f'/events?attempt={a}', login('r5x'))[0]]
chk('TC-PE07-19', x19 == [200, 403, 403, 403, 403], str(x19))
r20 = [S('GET', EX + '/results', T), S('GET', EX + '/results', TA)]
tj = json.dumps(r20[0][1]); ta_j = json.dumps(r20[1][1])
chk('TC-PE07-20', r20[0][0] == 200 and r20[1][0] == 200 and ('integrity' in tj or 'tab_hidden' in tj or 'paste' in tj.lower()) and not ('integrity' in ta_j or 'paste' in ta_j.lower()), f'GV thấy tóm tắt liêm chính: {("integrity" in tj)}; TA: {("integrity" in ta_j)}; khoá GV: {sorted(r20[0][1].get("items", [{}])[0]) if r20[0][1].get("items") else r20[0][1].keys()}')
# 23 cột phạt
cols = db("select column_name from information_schema.columns where table_name in ('exam_attempts','exam_answers','exam_events') and (column_name ilike '%penal%' or column_name ilike '%cheat%' or column_name ilike '%deduct%' or column_name ilike '%flag%')")
rec('TC-PE07-23', cols == '' , f'cột "điểm trừ/gian lận/cờ" trong exam_*: {cols or "không có"}')
x42 = db("select column_name from information_schema.columns where table_name='exam_events' order by ordinal_position").replace('\n', ',')
chk('TC-PE07-42', not re.search(r'ip|agent|content|clipboard', x42.replace('type', '').replace('exam', '')), f'exam_events: {x42}')
# nộp → khoá gỡ
t0 = time.time(); sb = submit_exam(eid, a, 'r5s13', H)
for _ in range(40):
    if rds('EXISTS', K) == '0': break
    time.sleep(0.5)
gone = time.time() - t0; g3 = S('GET', '/_test/chat-gate', t)[1]
chk('TC-PE07-02', sb[0] == 200 and gone <= 10 and g3.get('allowed') is True, f'nộp tay {sb[0]}; khoá Redis gỡ sau {gone:.1f}s (≤10); chat-gate allowed={g3.get("allowed")}')
# 04: gia hạn đã làm: kiểm TTL dài ra với r5s14
a2, j2, H2, _, _ = start(eid, 'r5s14'); K2 = 'ep:exam_lock:' + uid('r5s14'); tt0 = int(rds('TTL', K2))
chk('TC-PE07-04', ex[0] == 200 and tt0 > 0, f'extend +2h → {ex[0]}; TTL khoá lượt mới = {tt0} s (deadline cố định started+40p; closes mới 2h): theo deadline_at')
json.dump(dict(eid=eid, a2=a2), open('/tmp/qc2-c5/s07.json', 'w'))
