"""Chuẩn bị F19 (qc3): 3 câu trắc nghiệm đã duyệt + 1 bài code đã duyệt (R5C1, ep-r). Ghi /tmp/qc3e/f19.json."""
import sys, os; sys.path.insert(0, os.path.dirname(__file__))
from r5lib import *
tag = uuid.uuid4().hex[:5].upper()
qs = []
for i, (typ, n, cor) in enumerate([('MCQ_SINGLE', 4, (1,)), ('MCQ_MULTI', 4, (0, 2)), ('MCQ_SINGLE', 4, (2,))]):
    qid = mkq(i, typ, n, cor, title=f'F19{tag} câu {i+1}'); qs.append(dict(id=qid, title=f'F19{tag} câu {i+1}'))
qc = mkcode(77); call('PUT', f'{Q}/{qc}', T, {}) if False else None
g = call('GET', f'{Q}/{qc}', T)[1]
json.dump(dict(tag=tag, mcq=qs, code=dict(id=qc, title=g.get('title')), canary=CANARY, hid=HID, c1=C1), open('/tmp/qc3e/f19.json', 'w'), ensure_ascii=False)
print(open('/tmp/qc3e/f19.json').read())
