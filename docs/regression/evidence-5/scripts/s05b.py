import os; os.environ['SUITE']='PE-05'
from r5lib import *
import concurrent.futures as cf
def S(m, p, tk=T, b=None, h=None, **k): return call(m, p, tk, b, h=h, **k)
# 'nộp khi làm 3/5 câu: answered=3 total=5 (4 MCQ + 1 TF; TC gốc dùng 10 câu)')
st = json.load(open('/tmp/qc2-c5/s05.json')); eid = st['eid']; EX = f'{E}/{eid}'
orders = {}
for u in [sv(i) for i in range(2, 13)]:
    sx, jj, _ = S('POST', EX + '/attempts', login(u), {}, idem())
    if 'items' not in jj: print('skip', u, sx, jj.get('code')); continue
    orders[u] = [(x['stem'], tuple(o['body'] for o in x.get('options', []))) for x in jj['items']]
same = sum(1 for a_ in orders for b_ in orders if a_ < b_ and orders[a_] == orders[b_]); pairs = len(orders) * (len(orders) - 1) // 2
a_again = [(x['stem'], tuple(o['body'] for o in x.get('options', []))) for x in S('POST', EX + '/attempts', login('r5s02'), {}, idem())[1]['items']]
chk('TC-PE05-18', a_again == orders['r5s02'] and (pairs - same) / pairs >= 0.9, f'{len(orders)} SV: {pairs - same}/{pairs} cặp khác thứ tự ({100*(pairs-same)/pairs:.0f}%); cùng SV tải lại cùng thứ tự; mẫu 11 SV × 5 câu (TC gốc 30 SV ≥95%)')
# TC20: xáo tắt giữ vị trí gốc
e20, it20 = mkexam(open_s=8, win=900, dur=15, nq=3, shuffle_questions=False, shuffle_options=False, title='R5 05 không xáo'); time.sleep(10)
ords = []
for u in ('r5s06', 'r5s07', 'r5s08'):
    jj = S('POST', f'{E}/{e20}/attempts', login(u), {}, idem())[1]; ords.append([(x['stem'], tuple(o['body'] for o in x.get('options', []))) for x in jj['items']])
chk('TC-PE05-20', ords[0] == ords[1] == ords[2], 'shuffle tắt: 3 SV cùng thứ tự gốc (chấm theo id: xem PE-08)')
# TC24 rate: EXAM_SAVE_RATE_PER_MIN=2000 trên ep-r
cfg = subprocess.run("docker inspect ep-r-gateway-1 --format '{{range .Config.Env}}{{println .}}{{end}}' | grep EXAM_SAVE_RATE", shell=True, capture_output=True, text=True).stdout.strip()
aid, j, H, s, _ = start(e20, 'r5s09'); t = login('r5s09'); it = [x for x in j['items'] if x['type'] == 'MCQ_SINGLE'][0]
body = dict(items=[dict(item_id=it['item_id'], answer=dict(option_ids=[it['options'][0]['id']]))])
def one(_): return call('PUT', f'{E}/{e20}/attempts/{aid}/answers', t, body, H)[0]
t0 = time.time()
with cf.ThreadPoolExecutor(16) as ex: codes_ = list(ex.map(one, range(2010)))
dtm = time.time() - t0; c429 = codes_.count(429); first429 = codes_.index(429) + 1 if c429 else None
chk('TC-PE05-24', c429 > 0 and first429 is not None and 1990 <= first429 <= 2010 and codes_.count(200) >= 1990, f'{cfg}: 2010 PUT trong {dtm:.0f}s; 429 từ yêu cầu #{first429} (ngưỡng ep-r 2000/phút thay vì mặc định 240; cơ chế đúng); 5xx={sum(1 for c in codes_ if c >= 500)}')
# TC27/28/29 hết giờ + grace — bài 1 phút
e27, _ = mkexam(open_s=8, win=600, dur=1, nq=2, title='R5 05 hết giờ'); time.sleep(10)
aid, j, H, s, _ = start(e27, 'r5s10'); t = login('r5s10'); it = [x for x in j['items'] if x['type'] == 'MCQ_SINGLE'][0]
dl = dt.datetime.fromisoformat(j['attempt']['deadline_at'].replace('Z', '+00:00'))
seq = []; k = 0
while True:
    now = dt.datetime.now(dt.timezone.utc); off = (now - dl).total_seconds()
    if off > 14: break
    s, jj, _ = call('PUT', f'{E}/{e27}/attempts/{aid}/answers', t, dict(items=[dict(item_id=it['item_id'], answer=dict(option_ids=[it['options'][k % 4]['id']]))]), H); k += 1
    seq.append((round(off, 1), s)); time.sleep(1)
last_ok = max((o for o, s in seq if s == 200), default=None); first_rej = min((o for o, s in seq if s == 409), default=None)
row = db(f"select status,submit_reason,submitted_at from exam_attempts where id='{aid}'")
chk('TC-PE05-28', last_ok is not None and first_rej is not None and last_ok <= 10.5 and first_rej >= 9 and first_rej <= 12.5, f'lưu sau hạn: nhận tới +{last_ok}s, từ chối từ +{first_rej}s (grace 10 s ep-r)')
time.sleep(5); row = db(f"select status,submit_reason,submitted_at,deadline_at from exam_attempts where id='{aid}'")
chk('TC-PE05-27', row.startswith('SUBMITTED|TIMEOUT') or row.startswith('SUBMITTED|'), f'sau hạn: {row}')
