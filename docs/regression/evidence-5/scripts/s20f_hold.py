"""QC2 smoke sprint 5 (SmkM): bài đã đóng chưa công bố (hold) cho PE08-32 + bài đang mở có SV đã nộp. Ghi /tmp/qc2L/hold.json"""
import os, sys; os.environ['SUITE'] = 'UIM'
from r5lib import *
fx = json.load(open('/tmp/qc2L/uix2.json'))
e, _ = mkexam(open_s=10, win=75, dur=1, nq=2, title='R5M hold đã đóng'); time.sleep(12)
g = call('GET', f'{E}/{e}', T)[1]; s, j, _ = call('PUT', f'{E}/{e}/publish-hold', T, dict(hold=True, version=g['version'])); print('hold', s, codes(j))
aid, jj, H, s0, _ = start(e, 'r5s35'); print('start', s0); time.sleep(1); print('submit', submit_exam(e, aid, 'r5s35', H, E)[0])
# bài đang mở: r5s34 nộp ở xmcq
aid2, _, H2, s1, _ = start(fx['xmcq'], 'r5s34'); print('start2', s1); time.sleep(1); print('submit2', submit_exam(fx['xmcq'], aid2, 'r5s34', H2, E)[0])
json.dump(dict(hold=e, open=fx['xmcq'], c1=C1), open('/tmp/qc2L/hold.json', 'w'))
