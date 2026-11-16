"""QC2 smoke sprint 5 (SmkM) — TC-PE07-36: so độ giống 1.000 bản nộp (QC sinh biến thể bằng Python, chèn DB bằng s20h_bulk.py setup). Chạy: python3 s20k_sim.py"""
import os, threading; os.environ['SUITE'] = 'BULK'
from r5lib import *
import rl
bk = json.load(open('/tmp/qc2L/bulk.json')); e = bk['exam']; C4 = bk['c4']; EX = f'/courses/{C4}/exams/{e}'
def mem(c):
    o = subprocess.run(['docker', 'stats', '--no-stream', '--format', '{{.MemUsage}}', c], capture_output=True, text=True).stdout.strip().split('/')[0].strip(); m = re.match(r'([\d.]+)(MiB|GiB|KiB)', o); return float(m.group(1)) * (1024 if m.group(2) == 'GiB' else 1 if m.group(2) == 'MiB' else 1 / 1024)
base = mem('ep-r-worker-1'); peak = [base]; stop = [False]
def poll():
    while not stop[0]: peak.append(mem('ep-r-worker-1'))
th = threading.Thread(target=poll); th.start()
t0 = time.time(); s, j, _ = call('POST', EX + '/similarity/run', T, {}, idem()); print(s, j)
jid = j.get('job_id') or j.get('id'); done = None
for _ in range(240):
    jj = call('GET', '/jobs/' + jid, T)[1]
    if jj.get('status') in ('SUCCEEDED', 'FAILED', 'DEAD'): done = jj; break
    time.sleep(0.5)
took = time.time() - t0; stop[0] = True; th.join()
sim = call('GET', EX + '/similarity?limit=100', T)[1]; npairs = db(f"select count(*) from exam_similarity_pairs where exam_id='{e}'") if False else None
print(took, done and done.get('status'), json.dumps(done)[:400]); print(json.dumps(sim)[:300])
res = (done or {}).get('result') or {}; nsub = db(f"select count(*) from code_submissions where exam_id='{e}'")
rl.R('TC-PE07-36', done is not None and done.get('status') == 'SUCCEEDED' and took <= 30, f'1.000 bản nộp C++ biến thể do QC sinh (Python, seed cố định) chèn thẳng DB (lớp R5C4, {nsub} bản): POST similarity/run → job {done and done.get("status")} sau {took:.1f} s (≤30 s, gồm xếp hàng + thăm dò 0,5 s); kết quả job {json.dumps(res, ensure_ascii=False)[:200]}; RAM worker nền {base:.0f} MiB → đỉnh {max(peak):.0f} MiB (không OOM)')
