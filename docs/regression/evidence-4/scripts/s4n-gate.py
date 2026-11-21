# SmkT4N: kiểm chứng trực tiếp GATEP2-49 (CSRF), GATEP2-42 (đoán mã join), GATEP2-40 (mạo danh MSSV)
import sys, json, re; sys.path.insert(0, '.'); from q4lib import *
S = json.load(open('../qc2-s4-ui.json')); out = {}
def cookie_of(h):
    sc = h.get('Set-Cookie', '') or ''
    m = re.search(r'ep_rt=([^;]+)', sc); return m.group(1) if m else None
def rf(ck, extra=None, method='POST', ct='nochange'):
    h = {'Cookie': 'ep_rt=' + ck}; h.update(extra or {})
    return call(method, '/auth/refresh', None, None, hdr=h, idem=False)
def fresh(u):
    resetrl(); s, j, h = call('POST', '/auth/login', None, {'email': u + '@edupilot.local', 'password': PW}); assert s == 200, (s, j); ck = cookie_of(h); assert ck, h; return ck
# ---- GATEP2-49 CSRF
res = {}
ck = fresh('sv30')
for lab, hd in [('Origin evil.example', {'Origin': 'https://evil.example'}), ('Origin null', {'Origin': 'null'}), ('Origin localhost.evil.example', {'Origin': 'https://localhost.evil.example'}), ('Sec-Fetch-Site cross-site', {'Sec-Fetch-Site': 'cross-site'})]:
    s, j, _ = rf(ck, hd); res[lab] = (s, (j.get('details') or {}).get('reason') if isinstance(j.get('details'), dict) else j.get('code'))
s, j, h = rf(ck, {'Origin': 'https://localhost:1143'}); res['Origin hợp lệ'] = s; ck2 = cookie_of(h) or ck   # xoay
s, j, h = rf(ck2); res['không Origin/SFS (sau xoay)'] = s; ck3 = cookie_of(h) or ck2
res['cookie cũ ck sau xoay (kẻ phá)'] = rf(ck)[0]
# phiên chưa bị thu hồi bởi các lần chặn: tạo phiên mới, bị chặn 3 lần, rồi refresh hợp lệ cùng cookie
ck = fresh('sv31')
for hd in [{'Origin': 'https://evil.example'}, {'Sec-Fetch-Site': 'cross-site'}]: rf(ck, hd)
res['refresh hợp lệ sau các lần bị chặn (cùng cookie)'] = rf(ck)[0]
res['GET /auth/refresh'] = call('GET', '/auth/refresh', None, None, hdr={'Cookie': 'ep_rt=' + ck}, idem=False)[0]
out['49'] = res
# ---- GATEP2-42 đoán mã tham gia: nhiều tài khoản, XFF giả
import random, string
alpha = 'ABCDEFGHJKMNPQRSTUVWXYZ23456789'
resetjoin(); codes = []; rs = []
for i, u in enumerate(['sv32', 'sv33', 'sv34', 'sv35']):
    tok = L(u); seq = []
    for k in range(8):
        c = ''.join(random.choice(alpha) for _ in range(7)); codes.append(c)
        s, j, hd = call('POST', '/courses/join', tok, {'code': c}, hdr={'X-Forwarded-For': f'10.9.{i}.{k+1}'}); seq.append(s)
    rs.append(seq)
out['42'] = {'tokens': rs, 'tổng mã thử': len(codes)}
logs = ''.join(subprocess.run(['docker', 'logs', c], capture_output=True, text=True).stdout + subprocess.run(['docker', 'logs', c], capture_output=True, text=True).stderr for c in ['ep-r-gateway-1', 'ep-r-caddy-1', 'ep-r-worker-1'])
out['42']['mã thử xuất hiện trong log'] = sum(1 for c in codes if c in logs)
resetjoin()
print(json.dumps(out, ensure_ascii=False, indent=1))
json.dump(out, open('../logs/s4n-gate.json', 'w'), ensure_ascii=False, indent=1)
