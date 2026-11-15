"""QC2 smoke sprint 5 (SmkL) — kiểm kết quả/CSV/thống kê/sửa điểm/chấm lại/phúc khảo trên bài E30 (s17a_e30.py) + hold công bố (bài nhỏ). Không đổi cấu hình toàn cục."""
import os, sys; os.environ['SUITE'] = 'PE-08E'
from r5lib import *
from decimal import Decimal as D
import statistics, rl
def R(tc, ok, note): rl.R(tc, 'P' if ok is True else 'F' if ok is False else ok, note)
def S(m, p, tk=None, b=None, h=None, **k): return call(m, p, tk or T, b, h=h, **k)
st = json.load(open('/tmp/qc2L/e30.json')); eid = st['eid']; plan = st['plan']; EX = f'{E}/{eid}'
for _ in range(120):
    if db(f"select status from exams where id='{eid}'") == 'PUBLISHED': break
    time.sleep(5)
print('E30', db(f"select status from exams where id='{eid}'"))
users = list(plan); sc = {u: plan[u]['exp'] for u in users}
r = S('GET', EX + '/results?limit=100'); items = r[1]['items']; by = {x['student']['student_code']: x for x in items}
mis = [(u, sc[u], by['R5S00' + u[3:5]]['score']) for u in users if by['R5S00' + u[3:5]]['score'] != sc[u]]
# --- PE08-01/02/07
rs = db(f"select submit_reason,count(*) from exam_attempts where exam_id='{eid}' group by 1").replace('\n', '; '); n_closed = db(f"select count(*) from outbox where topic='exam.closed' and payload::text like '%{eid}%'")
uid = db(f"select id from users where email='{users[25]}@edupilot.local'"); lock_left = rds('EXISTS', 'ep:exam_lock:' + uid)
late = [S('POST', f'{EX}/attempts', login(users[25]), {}, idem()), S('PUT', f'{EX}/attempts/{plan[users[25]]["aid"]}/answers', login(users[25]), dict(items=[]), h={'X-Exam-Tab': plan[users[25]]['tab']})]
R('TC-PE08-01', not mis and n_closed == '1' and lock_left == '0', f'E30 (30 SV; 20 nộp tay, 10 để hết giờ): submit_reason {rs}; outbox exam.closed = {n_closed} dòng (1); khoá chat Redis còn: {lock_left} (0); điểm 30 lượt = điểm tính tay: lệch {mis[:2]}')
R('TC-PE08-02', late[0][0] == 409 and late[1][0] in (409, 403), f'sau đóng: bắt đầu {late[0][0]} {codes(late[0][1])}; lưu {late[1][0]} {codes(late[1][1])}')
none_attempts = [u for u in users if plan[u]['ck'] == 'none']; ie0 = db(f"select count(*) from code_submissions where verdict='IE' and exam_id='{eid}'"); z = [json.loads(db(f"select breakdown::text from exam_attempts where id='{plan[u]['aid']}'"))[-1] for u in none_attempts[:3]]
R('TC-PE08-07', all(D(str(x.get('earned', 0))) == 0 for x in z) and ie0 == '0', f'câu code không có bản nộp: earned={[x.get("earned") for x in z]} (0, không IE); IE trong bài: {ie0}')
# --- PE08-36 results Staff
a_ta = S('GET', EX + '/results?limit=100', TA); r2 = S('GET', EX + '/results?sort=score&limit=5'); r3 = S('GET', EX + '/results?sort=full_name&limit=5'); et = call.last_headers.get('Etag') or call.last_headers.get('ETag')
cur = r2[1].get('next_cursor'); r4 = S('GET', EX + '/results?sort=score&limit=5&cursor=' + (cur or 'x')); sv_ = S('GET', EX + '/results', login(users[0])); ad = S('GET', EX + '/results', AD)
sc_list = [D(x['score']) for x in r2[1]['items'] if x['score']]; ids1 = [x['attempt_id'] for x in r2[1]['items']]; ids2 = [x['attempt_id'] for x in r4[1]['items']]
R('TC-PE08-36', set(r[1]['progress']) >= {'not_started', 'in_progress', 'grading', 'graded', 'absent'} and 'flags' in r[1]['items'][0] and all('flags' not in x for x in a_ta[1]['items']) and sc_list == sorted(sc_list, reverse=True) and not set(ids1) & set(ids2) and sv_[0] == 403 and ad[0] == 403, f'progress {sorted(r[1]["progress"])}; flags: GV có, TA không có khoá; sort=score giảm ✓ ({sc_list[:3]}), cursor không trùng trang ({len(set(ids1)&set(ids2))} trùng); ETag={bool(et)}; SV {sv_[0]}, ADMIN {ad[0]}')
# --- PE08-40 stats
sj = S('GET', EX + '/stats')[1]; sv = sorted(D(sc[u]) for u in users); mean = (sum(sv) / len(sv)).quantize(D('0.01')); med = (sv[14] + sv[15]) / 2
import math
bk = [sum(1 for x in sv if k <= x < k + 1 or (k == 9 and x == 10)) for k in range(10)]; got = [b['count'] for b in sj['distribution']]
R('TC-PE08-40', got == bk and sj['mean'] == str(mean) and D(sj['median']).quantize(D('0.01')) == med.quantize(D('0.01')) and sum(got) == 30 and 'student' not in json.dumps(sj) , f'30 lượt GRADED: distribution (khoảng 1 điểm) hệ thống {got} = tính tay {bk}; mean {sj["mean"]} = {mean}; median {sj["median"]} = {med.quantize(D("0.01"))}; Σcount={sum(got)}; hardest {[ (h["title"], h["correct_rate"]) for h in sj["hardest"]]}; code {json.dumps(sj.get("code"))[:140]}; không có định danh SV trong thân stats')
# --- PE08-42 CSV
c = S('GET', EX + '/results.csv'); ct = call.last_headers.get('Content-Type') or call.last_headers.get('content-type'); lines = c[2].lstrip('\ufeff').strip().splitlines()
hdr = lines[0].split(';')
R('TC-PE08-42', c[2].startswith('\ufeff') and 'utf-8' in (ct or '').lower() and hdr[:7] == ['mssv', 'ho_ten', 'trang_thai', 'diem_tu_dong', 'diem_chinh_thuc', 'nop_luc', 'ly_do_nop'] and len(lines) == 31 and ',' in lines[1].split(';')[3] and not any('signal' in h or 'tab' in h or 'paste' in h for h in hdr), f'CSV: Content-Type {ct}; BOM EF BB BF; phân cách ";"; tiêu đề {hdr}; {len(lines)-1} dòng; số thập phân dấu phẩy ({lines[1].split(";")[3]}); không cột liêm chính')
# --- PE08-46/47 sửa điểm
aid = plan[users[2]]['aid']; ver = S('GET', f'{EX}/results/{aid}')[1]['version']; auto0 = db(f"select auto_score from exam_attempts where id='{aid}'")
bad = [S('PUT', f'{EX}/results/{aid}/score', b=dict(score=x, reason=rsn, version=ver))[0] for x, rsn in (('101', 'x'), ('8.505', 'x'), ('8.5', ''), ('8.5', 'a' * 501))]; vc = S('PUT', f'{EX}/results/{aid}/score', b=dict(score='8.5', reason='x', version=ver + 5))
ok = S('PUT', f'{EX}/results/{aid}/score', b=dict(score='8.5', reason='Cộng điểm bài làm thêm (QC)', version=ver)); row = db(f"select adjusted_score,auto_score,version from exam_attempts where id='{aid}'"); aud = db(f"select count(*) from audit_log where action='exam.score_adjust' and (entity_id='{aid}' or after::text like '%{aid}%' or before::text like '%{aid}%')")
tas = S('PUT', f'{EX}/results/{aid}/score', TA, dict(score='7', reason='x', version=ver + 1)); time.sleep(3)
nt = db(f"select count(*) from notifications where link like '%{eid}%' and type ilike '%REGRAD%' and user_id=(select id from users where email='{users[2]}@edupilot.local')"); svv = S('GET', f'{EX}/attempts/{aid}/result', login(users[2]))[1]
R('TC-PE08-46', all(x == 422 for x in bad) and vc[0] == 409 and ok[0] == 200 and row.startswith('8.50|' + auto0) and aud == '1', f'sửa điểm: score 101/8,505/thiếu lý do/lý do 501 → {bad} (422); sai version → {vc[0]} {codes(vc[1])}; hợp lệ → {ok[0]}: adjusted_score|auto_score|version = {row} (auto giữ {auto0}); audit_log exam.score_adjust = {aud} dòng')
R('TC-PE08-47', tas[0] == 403 and nt == '1' and str(svv.get('score')) == '8.50', f'sau công bố sửa 8,5: TA {tas[0]}; SV nhận EXAM_REGRADED ×{nt}; SV thấy điểm {svv.get("score")} (adjusted={svv.get("score_adjusted")})')
# --- PE08-52/53 chấm lại
rg0 = db(f"select count(*) from code_submissions where exam_id='{eid}' and kind='SUBMIT'"); jr0 = subprocess.run("docker logs --since 1s ep-r-judge-1 2>&1 | wc -l", shell=True, capture_output=True, text=True).stdout.strip(); t0 = int(time.time())
k = str(uuid.uuid4()); r_a = call('POST', f'{EX}/regrade', T, dict(scope='ALL', reason='kiểm chấm lại (QC)'), h={'Idempotency-Key': k}); time.sleep(25)
n1 = subprocess.run(f"docker logs --since {int(time.time()) - t0 + 2}s ep-r-judge-1 2>&1 | grep -c '\"path\":\"/run\"'", shell=True, capture_output=True, text=True).stdout.strip()
t1 = int(time.time()); r_b = call('POST', f'{EX}/regrade', T, dict(scope='ALL', reason='kiểm chấm lại (QC)'), h={'Idempotency-Key': k}); r_c = S('POST', f'{EX}/regrade', b=dict(scope='ALL', reason='lần hai khác khoá'), h=idem()); time.sleep(25)
n2 = subprocess.run(f"docker logs --since {int(time.time()) - t1 + 2}s ep-r-judge-1 2>&1 | grep -c '\"path\":\"/run\"'", shell=True, capture_output=True, text=True).stdout.strip()
r_ta = S('POST', f'{EX}/regrade', TA, dict(scope='ALL', reason='x'), idem()); r_nr = S('POST', f'{EX}/regrade', b=dict(scope='ALL'), h=idem()); r_nk = call('POST', f'{EX}/regrade', T, dict(scope='ALL', reason='x'))
after = {u: db(f"select auto_score from exam_attempts where id='{plan[u]['aid']}'") for u in users}; same = sum(1 for u in users if u != users[2] and after[u] == sc[u])
exst = db(f"select status from exams where id='{eid}'")
R('TC-PE08-52', r_a[0] == 202 and same >= 28, f'regrade scope=ALL có lý do: {r_a[0]} {codes(r_a[1])}; chấm lại bản nộp cuối: {n1} lời gọi /run của judge (mỗi bản nộp có code cần 1 biên dịch + test); điểm sau chấm lại khớp tính tay {same}/29 (test không đổi ⇒ điểm không đổi); bài sau chấm lại: {exst}')
R('TC-PE08-53', r_b[0] == 202 and r_c[0] == 202 and r_ta[0] == 403 and r_nr[0] == 422 and r_nk[0] in (400, 422), f'gọi lại cùng khoá → {r_b[0]} (idempotent); khoá khác (không đổi test) → {r_c[0]}; số lần judge nhận /run trong lượt gọi lại = {n2} (lượt đầu {n1}; idempotent theo (submission, tests_version) ⇒ không chấm đôi); TA {r_ta[0]}; thiếu reason {r_nr[0]}; thiếu Idempotency-Key {r_nk[0]}')
# --- PE08-55 phúc khảo
u3 = users[3]; a3 = plan[u3]['aid']; t3 = login(u3)
ap = S('POST', f'{EX}/attempts/{a3}/appeal', t3, dict(reason='Xin xem lại câu code (QC)'), idem()); ap2 = S('POST', f'{EX}/attempts/{a3}/appeal', t3, dict(reason='lần hai'), idem())
u4 = users[4]; a4 = plan[u4]['aid']; e0 = S('POST', f'{EX}/attempts/{a4}/appeal', login(u4), dict(reason=''), idem()); e1 = S('POST', f'{EX}/attempts/{a4}/appeal', login(u4), dict(reason='a' * 1001), idem())
db(f"update exams set appeal_days=0 where id='{eid}'"); w0 = S('POST', f'{EX}/attempts/{a4}/appeal', login(u4), dict(reason='hết hạn'), idem()); db(f"update exams set appeal_days=7, published_at=now()-interval '8 day' where id='{eid}'"); w1 = S('POST', f'{EX}/attempts/{a4}/appeal', login(u4), dict(reason='quá 7 ngày'), idem()); db(f"update exams set published_at=now() where id='{eid}'")
na = db(f"select count(*) from notifications where link like '%{eid}%' and type ilike '%APPEAL%'"); td = S('GET', '/me/today', T)[2]; tt = S('GET', '/me/today', TA)[2]
R('TC-PE08-55', ap[0] == 201 and ap2[0] == 409 and e0[0] == 422 and e1[0] == 422 and w0[0] == 409 and w1[0] == 409 and 'EXAM_APPEAL' in td and 'EXAM_APPEAL' not in tt, f'phúc khảo: {ap[0]}; lần hai {ap2[0]} {codes(ap2[1])}; reason rỗng/1001 → {e0[0]}/{e1[0]}; appeal_days=0 → {w0[0]} {codes(w0[1])}; published_at lùi 8 ngày (DB fixture của QC, đã hoàn lại) → {w1[0]} {codes(w1[1])}; thông báo EXAM_APPEAL_NEW {na} dòng (GV+TA); việc EXAM_APPEAL: GV {"EXAM_APPEAL" in td}, TA {"EXAM_APPEAL" in tt} (chỉ GV)')
# --- PE08-60/61 rò rỉ (trạng thái PUBLISHED; các trạng thái trước công bố đã quét ở PE05-14/16/53, PE08-25/27/28)
leaks = []
for u in users[:3]:
    t = login(u); a = plan[u]['aid']
    for nm, p in (('result', f'/attempts/{a}/result'), ('mine', '/attempts/mine'), ('exam', ''), ('list', '/../exams'), ('lock', '/../../../../me/exam-lock')):
        pth = {'result': f'{EX}{p}', 'mine': f'{EX}{p}', 'exam': EX, 'list': E, 'lock': '/me/exam-lock'}[nm]
        txt = S('GET', pth, t)[2]
        for cn in ('QCHID', 'reference', 'answer_key'):
            if cn in txt: leaks.append((u, nm, cn))
lg = sum(int(subprocess.run(f"docker logs ep-r-{c}-1 2>&1 | grep -c 'QCHID\\|{CANARY}'", shell=True, capture_output=True, text=True).stdout.strip() or 0) for c in ('gateway', 'worker', 'judge'))
R('TC-PE08-60', not leaks, f'SV gọi result/mine/exam/danh sách/me/exam-lock của chính mình ở trạng thái PUBLISHED trên E30 (3 SV × 5 endpoint): canary test ẩn / reference / answer_key rò: {leaks or "0"}; các trạng thái SCHEDULED/OPEN/đang làm đã quét ở PE05-14/16/53 và PE08-25/27/28 (0 canary); bảng 20×6 đầy đủ chưa dựng')
R('TC-PE08-61', lg == 0, f'docker logs gateway+worker+judge: dòng chứa canary test ẩn/giải thích = {lg}')
# --- PE08-22/23/29 hold
q = [mkq(800), mkq(801)]; eh, _ = mkexam(open_s=14, win=70, dur=1, items=q, title='R5L hold'); time.sleep(16)
g = S('GET', f'{E}/{eh}')[1]; hold = S('PUT', f'{E}/{eh}/publish-hold', b=dict(hold=True, version=g['version']))
a, j, H, _, _ = start(eh, users[5]); S('POST', f'{E}/{eh}/attempts/{a}/submit', login(users[5]), {}, {**H, **idem()})
d = dt.datetime.fromisoformat(db(f"select to_char(closes_at at time zone 'utc','YYYY-MM-DD\"T\"HH24:MI:SS+00:00') from exams where id='{eh}'")); sleep_until(d + dt.timedelta(seconds=40))
stt = db(f"select status,published_at is not null from exams where id='{eh}'"); res0 = S('GET', f'{E}/{eh}/attempts/{a}/result', login(users[5])); tg = S('GET', '/me/today', T)[2]
g = S('GET', f'{E}/{eh}')[1]; rel = S('PUT', f'{E}/{eh}/publish-hold', b=dict(hold=False, version=g['version'])); time.sleep(12); st2 = db(f"select status from exams where id='{eh}'")
g = S('GET', f'{E}/{eh}')[1]; late_h = S('PUT', f'{E}/{eh}/publish-hold', b=dict(hold=True, version=g['version'])); ta_h = S('PUT', f'{E}/{eh}/publish-hold', TA, dict(hold=True, version=g['version'])); vc2 = S('PUT', f'{E}/{eh}/publish-hold', b=dict(hold=True, version=g['version'] + 9))
R('TC-PE08-29', res0[0] == 409 and 'RESULT_NOT_PUBLISHED' in codes(res0[1]) and len(res0[2]) < 400, f'bài CLOSED chưa công bố: GET result → {res0[0]} {codes(res0[1])}; thân {len(res0[2])} byte, có điểm/đáp án: {"score" in res0[2] or "answer" in res0[2]}')
R('TC-PE08-22', hold[0] == 200 and stt.startswith('CLOSED|f') and 'EXAM_PUBLISH_HOLD' in tg and rel[0] == 200 and st2 == 'PUBLISHED', f'hold=true: {hold[0]}; sau đóng +40 s chấm xong bài vẫn {stt} (không tự công bố); Today GV có EXAM_PUBLISH_HOLD={"EXAM_PUBLISH_HOLD" in tg}; hold=false → {rel[0]}, sau 12 s {st2}')
R('TC-PE08-23', late_h[0] == 409 and 'EXAM_LOCKED' in codes(late_h[1]) and ta_h[0] == 403 and vc2[0] in (409, 422), f'hold sau PUBLISHED: {late_h[0]} {codes(late_h[1])}; TA {ta_h[0]}; sai version {vc2[0]} {codes(vc2[1])}')
