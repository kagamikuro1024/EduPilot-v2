import sys, re; sys.path.insert(0, '.')
from s8r_chat import *
CAN = 'Lớp chưa có công thức điểm chính thức do giảng viên xác nhận.'
R = cid('QD8R6U'); tkt = lambda c: int(L.psql(f"select count(*) from escalation_tickets where course_id='{c}' and origin='AUTO_NO_FORMULA'"))
# 09-32: cách tính điểm, lớp CONFIRMED
n0 = tkt(R); t0 = now_db(); _, a, ev = ask('sv.kha', R, 'Cách tính điểm môn này thế nào?'); n1 = tkt(R)
has = [bool(re.search(p, a)) for p in (r'20\s*%', r'0[,.]25', r'(1[,.]0|tối đa là 1\b|tối đa 1\b)', r'0[,.]5', r'[Qq]uy chế.{0,12}(tr\.|trang)\s*[12]')]
T('TC-P6-09-32', 'P' if has[0] and has[1] and n1 == n0 else 'F', f'hỏi «Cách tính điểm môn này thế nào?» (QD8R6U): trả (toàn văn ở logs/s8r-09-32-answer.txt) «{a[:200]}…»; có 20% {has[0]}, 0,25/lần {has[1]}, trần 1,0 {has[2]}, làm tròn/trừ 0,5 {has[3]}, nguồn Quy chế {has[4]}; AUTO_NO_FORMULA {n0}→{n1}')

open('../logs/s8r-09-32-answer.txt','w').write(a)
print(a)
