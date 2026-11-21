"""F19 lần 2 (qc3): bài đóng nhanh dùng lại 3 câu + bài code của f19.json. Ghi /tmp/qc3e/f19b.json (id bài, thời điểm mở/đóng)."""
import sys, os; sys.path.insert(0, os.path.dirname(__file__))
from r5lib import *
F = json.load(open('/tmp/qc3e/f19.json'))
items = [q['id'] for q in F['mcq']] + [F['code']['id']]
eid, _ = mkexam(open_s=75, win=300, dur=4, items=items, title='F19b qc3 ' + F['tag'])
print(eid, db(f"select opens_at||'|'||closes_at||'|'||status from exams where id='{eid}'"))
json.dump(dict(eid=eid, tag=F['tag']), open('/tmp/qc3e/f19b.json', 'w'))
