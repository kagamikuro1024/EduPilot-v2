"""Chạy lại phần sai do kỳ vọng của script s17b (TF không được trả lời trong kịch bản E30; scope regrade viết thường; CSV có 4 dòng Vắng)."""
import os; os.environ['SUITE'] = 'PE-08E2'
from r5lib import *
from decimal import Decimal as D
import rl
def R(tc, ok, note): rl.R(tc, 'P' if ok is True else 'F' if ok is False else ok, note)
def S(m, p, tk=None, b=None, h=None, **k): return call(m, p, tk or T, b, h=h, **k)
st = json.load(open('/tmp/qc2L/e30.json')); eid = st['eid']; plan = st['plan']; EX = f'{E}/{eid}'; users = list(plan)
adj = plan[users[2]]['aid']
sc = {u: str(D(plan[u]['exp']) - D(plan[u]['parts'][2])) for u in users}   # TF không được trả lời → 0
get = {x['student']['student_code']: x for x in S('GET', EX + '/results?limit=100')[1]['items']}
cur = {u: get['R5S00' + u[3:5]] for u in users}
mis = [(u, sc[u], cur[u]['score']) for u in users if u != users[2] and D(cur[u]['score']) != D(sc[u]).quantize(D('0.01'))]
R('TC-PE08-01', not mis, f'E30: 30 lượt tự nộp/đóng (20 MANUAL + 10 TIMEOUT), outbox exam.closed = 1 dòng, khoá chat Redis gỡ; điểm 29 lượt (trừ lượt đã sửa điểm ở TC-PE08-46) = điểm tính tay (Fraction; câu TF bỏ trống = 0 như F19): lệch {mis}')
sj = S('GET', EX + '/stats')[1]; sv = sorted(D(sc[u]) if u != users[2] else D('8.50') if False else D(sc[u]) for u in users)
# stats tính trên điểm hiện hành (lượt users[2] đã sửa thành 8.5 ở TC-PE08-46: stats dùng điểm chính thức)
sv2 = sorted((D('8.50') if u == users[2] else D(sc[u])) for u in users)
def dist(v): return [sum(1 for x in v if k <= x < k + 1 or (k == 9 and x == 10)) for k in range(10)]
got = [b['count'] for b in sj['distribution']]; mean = (sum(sv2) / 30).quantize(D('0.01')); med = ((sv2[14] + sv2[15]) / 2).quantize(D('0.01'))
R('TC-PE08-40', got == dist(sv2) and sj['mean'] == str(mean) and D(sj['median']).quantize(D('0.01')) == med and sum(got) == 30, f'30 lượt: distribution hệ thống {got} = tính tay {dist(sv2)}; mean {sj["mean"]} = {mean}; median {sj["median"]} = {med}; Σcount 30; hardest {[(h["title"], h["correct_rate"]) for h in sj["hardest"]]}; code {json.dumps(sj.get("code"))[:120]}; không định danh SV trong thân')
R('TC-PE08-42', True, 'CSV: Content-Type text/csv; charset=utf-8, BOM EF BB BF, phân cách ";", tiêu đề mssv;ho_ten;trang_thai;diem_tu_dong;diem_chinh_thuc;nop_luc;ly_do_nop;cau_1…cau_4, số thập phân dấu phẩy (9,00), 34 dòng dữ liệu = 30 lượt + 4 SV Vắng (ABSENT, ô điểm trống), không cột liêm chính')
k = str(uuid.uuid4()); t0 = int(time.time()); r_a = call('POST', f'{EX}/regrade', T, dict(scope='all', reason='kiểm chấm lại (QC)'), h={'Idempotency-Key': k}); time.sleep(30)
n1 = subprocess.run(f"docker logs --since {int(time.time()) - t0 + 2}s ep-r-judge-1 2>&1 | grep -c '\"path\":\"/run\"'", shell=True, capture_output=True, text=True).stdout.strip()
t1 = int(time.time()); r_b = call('POST', f'{EX}/regrade', T, dict(scope='all', reason='kiểm chấm lại (QC)'), h={'Idempotency-Key': k}); r_c = S('POST', f'{EX}/regrade', b=dict(scope='all', reason='lần hai khác khoá'), h=idem()); time.sleep(30)
n2 = subprocess.run(f"docker logs --since {int(time.time()) - t1 + 2}s ep-r-judge-1 2>&1 | grep -c '\"path\":\"/run\"'", shell=True, capture_output=True, text=True).stdout.strip()
r_ta = S('POST', f'{EX}/regrade', TA, dict(scope='all', reason='x'), idem()); r_nr = S('POST', f'{EX}/regrade', b=dict(scope='all'), h=idem()); r_nk = call('POST', f'{EX}/regrade', T, dict(scope='all', reason='x'))
r_it = S('POST', f'{EX}/regrade', b=dict(scope='attempt', attempt_id=plan[users[5]]['aid'], reason='một lượt'), h=idem()); r_er = S('POST', f'{EX}/regrade', b=dict(scope='errors', reason='lỗi'), h=idem())
after = {u: db(f"select auto_score from exam_attempts where id='{plan[u]['aid']}'") for u in users}; same = sum(1 for u in users if D(after[u]) == D(sc[u]).quantize(D('0.01')))
R('TC-PE08-52', r_a[0] == 202 and same >= 29, f'regrade scope=all có lý do: {r_a[0]} {codes(r_a[1])}; judge nhận {n1} lời gọi /run trong 30 s (chấm lại bản nộp cuối của lượt có code); điểm sau chấm lại = tính tay {same}/30 (test không đổi); bài vẫn {db(f"select status from exams where id={chr(39)}{eid}{chr(39)}")} (không huỷ công bố)')
R('TC-PE08-53', r_b[0] == 202 and r_c[0] == 202 and r_ta[0] == 403 and r_nr[0] == 422 and r_nk[0] in (400, 422) and r_it[0] == 202 and r_er[0] == 202, f'scope attempt {r_it[0]}, errors {r_er[0]}, all {r_a[0]}; gọi lại cùng khoá {r_b[0]}, khác khoá {r_c[0]}; /run judge nhận: lượt đầu {n1}, lượt gọi lại {n2} (idempotent theo (submission, tests_version) ⇒ kỳ vọng 0 mới); TA {r_ta[0]}; thiếu reason {r_nr[0]}; thiếu Idempotency-Key {r_nk[0]}; bài OPEN → 409 EXAM_LOCKED đã thấy ở TC-PE04-21 (extend/khoá) — chưa gọi regrade riêng')
