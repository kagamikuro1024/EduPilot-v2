#!/usr/bin/env python3
"""QC TC-PE01-23/24/25: oracle điểm bài thi bằng fractions.Fraction (không float). In bảng để đối chiếu với hàm Go."""
from fractions import Fraction as F
from math import floor
def rnd(x, step):  # làm tròn nửa lên về bội của step
    n = x / step; return floor(n + F(1,2)) * step
def partial(points, K, TP, FP): return points * max(F(0), F(TP - FP, K))
def code(points, wts, passed): return points * F(sum(w for w, p in zip(wts, passed) if p), sum(wts))
raw = F(1) + partial(F(2), 3, 2, 0) + code(F(3), [1,1,2], [True,False,True])
print('raw =', raw, '=', float(raw))
for step in [F(1,100), F(1,10), F(1,4), F(1,2), F(1)]: print('bước', step, '->', float(rnd(raw*F(10,6), step)) if False else float(rnd(raw, step)))
# max=10, tổng điểm câu 6 => thang 10/6
tot = F(6); print('thang 10:', float(raw*F(10)/tot), [float(rnd(raw*F(10)/tot, s)) for s in [F(1,100),F(1,10),F(1,4),F(1,2),F(1)]])
