import sys, re; sys.path.insert(0, '.')
from s8r_chat import *
R = cid('QD8R6U'); s, d = L.api('sv.kha', 'POST', f'/courses/{R}/me/grades/what-if', {'final_exam_score': '8'})
t0 = now_db(); sid, txt, ev = ask('sv.kha', R, 'nếu cuối kỳ em được 8 thì sao?'); la = L.psql(f"select count(*) from llm_audit where created_at >= '{t0}' and task='CHAT'")
ok = s == 200 and 'B+' in txt and re.search(r'\b8(,0)?\b', txt) and re.search(r'nếu', txt, re.I)
T('TC-P6-09-26', 'P' if ok else 'F', f'what-if API (final_exam_score=8): hp={d.get("hp")} letter={d.get("letter")}; chat «{txt[:260]}»: có B+ {"B+" in txt}, có «nếu» {bool(re.search("nếu", txt, re.I))}; llm_audit CHAT +{la}')
