"""Dựng bài thi cho ui_q2.cjs (SmkL): X6 = 6 phút (MCQ + nhiều đáp án), X1 = 1 phút, XC = MIXED (có câu code) mở 50 phút."""
import os, sys; os.environ['SUITE'] = 'UIX'
from r5lib import *
q = [mkq(500 + i, 'MCQ_SINGLE', 4, (1,)) for i in range(2)] + [mkq(502, 'MCQ_MULTI', 5, (0, 2)), mkq(503, 'TRUE_FALSE')]
x6, _ = mkexam(open_s=14, win=3000, dur=6, items=q, title='R5L UI 6 phút')
q1 = [mkq(510 + i, 'MCQ_SINGLE', 4, (0,)) for i in range(3)]
x1, _ = mkexam(open_s=14, win=3000, dur=1, items=q1, title='R5L UI 1 phút')
qm = [mkq(520, 'MCQ_SINGLE', 4, (1,)), mkq(521, 'MCQ_MULTI', 5, (0, 2)), mkcode(522)]
xc, _ = mkexam(open_s=14, win=3000, dur=50, items=qm, title='R5L UI code 50 phút')
qm1 = [mkq(530, 'MCQ_SINGLE', 4, (1,)), mkcode(531)]
xc1, _ = mkexam(open_s=14, win=3000, dur=1, items=qm1, title='R5L UI code 1 phút')
json.dump(dict(x6=x6, x1=x1, xc=xc, xc1=xc1, c1=C1), open('/tmp/qc2L/uix.json', 'w')); print(x6, x1, xc)
