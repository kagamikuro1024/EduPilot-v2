"""QC2 smoke sprint 5 (SmkM) — bài thi cho UI (class R5C1; tái dùng câu code đã xác minh). Ghi /tmp/qc2L/uix2.json. Chạy: python3 s20d_fixture.py"""
import os; os.environ['SUITE'] = 'smkP'
from r5lib import *
CODE = '01a1e14f-5763-7788-822f-5658704a1bc8'
q = [mkq(800 + i, 'MCQ_SINGLE', 4, (1,)) for i in range(2)] + [mkq(802, 'MCQ_MULTI', 5, (0, 2)), mkq(803, 'TRUE_FALSE')]
xmcq, _ = mkexam(open_s=14, win=14400, dur=30, items=q, title='R5M UI trắc nghiệm 30 phút')
xcode, _ = mkexam(open_s=14, win=14400, dur=50, items=[mkq(810, 'MCQ_SINGLE', 4, (1,)), mkq(811, 'MCQ_MULTI', 5, (0, 2)), CODE], title='R5M UI code 50 phút')
xcode1, _ = mkexam(open_s=14, win=14400, dur=1, items=[mkq(820, 'MCQ_SINGLE', 4, (1,)), CODE], title='R5M UI code 1 phút')
json.dump(dict(xmcq=xmcq, xcode=xcode, xcode1=xcode1, c1=C1), open('/tmp/qc2L/uix2.json', 'w')); print(xmcq, xcode, xcode1)
