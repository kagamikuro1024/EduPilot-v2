import sys, re, json; sys.path.insert(0, '.')
from s8r_chat import *
R = cid('QD8R6U'); _, a, ev = ask('sv.kha', R, 'Cách tính điểm môn này thế nào?')
print(type(ev), len(ev)); print([ (e[0] if isinstance(e, tuple) else e.get('event')) for e in ev][:30])
for e in ev:
    s = json.dumps(e, ensure_ascii=False)
    if e[0]=='block': print(json.dumps(e[1],ensure_ascii=False)[:1800])
