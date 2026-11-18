import importlib.util, os, json, time, sys
spec = importlib.util.spec_from_file_location('lib', 's8flow-lib.py'); L = importlib.util.module_from_spec(spec); spec.loader.exec_module(L)
PART = L.REPO + '/docs/regression/evidence-8/part-s8r.tsv'
def T(tc, st, n):
    n = ' '.join(str(n).split())[:420]; open(PART, 'a').write(f"{tc}\t{st}\t{n}\n"); print(st, tc, n[:200], flush=True)
cid = lambda c: L.psql(f"select id from courses where class_code='{c}'")
def rdel(): os.system("docker exec ep-r-redis-1 sh -c \"redis-cli --scan --pattern 'ep:today:*' | xargs -r redis-cli del\" >/dev/null")
# 05-06: CONFIRMED + nháp chờ
R6 = cid('QD8R6W'); s, d = L.api('teacher', 'GET', f'/courses/{R6}/grade-scheme/status')
T('TC-P6-05-06', 'P' if d.get('state') == 'CONFIRMED' and d.get('pending_draft') is True else 'F', f'QD8R6W CONFIRMED + nháp mới: status state={d.get("state")} pending_draft={d.get("pending_draft")}')
# 05-28: lớp lưu trữ
H2 = cid('QD8H2'); L.psql(f"update courses set status='ARCHIVED', archived_at=now(), join_enabled=false where id='{H2}'"); rdel()
s, dd = L.api('gv.s8f', 'GET', '/me/today'); arch = [a['title'] for a in dd['actions'] if 'QD8H2' in a['title'] and a['kind'] == 'GRADE_SCHEME_UNCONFIRMED']
L.psql(f"update courses set status='ACTIVE', archived_at=null, join_enabled=true where id='{H2}'"); rdel()
s, dd = L.api('gv.s8f', 'GET', '/me/today'); act = [a['title'] for a in dd['actions'] if 'QD8H2' in a['title'] and a['kind'] == 'GRADE_SCHEME_UNCONFIRMED']
T('TC-P6-05-28', 'P' if not arch and act else 'F', f'QD8H2 chuyển ARCHIVED (SQL, archived_at đặt) + xoá cache: mục công thức {arch} (kỳ vọng rỗng); trả ACTIVE: mục {len(act)} (kỳ vọng 1)')
# 05-42: xoá tài liệu nguồn của bản CONFIRMED (QD8H3): nhập điểm, HP trước/sau
H3 = cid('QD8H3'); Q = f'/courses/{H3}'
items = {k: i for k, i in [r.split('|') for r in L.psql(f"select i.key, i.id from grade_items i join grade_schemes s on s.id=i.scheme_id where i.course_id='{H3}' and s.status='CONFIRMED'").split('\n') if r]}
stu = [r.split('|')[0] for r in L.psql(f"select e.user_id from enrollments e where e.course_id='{H3}' and e.role_in_course='STUDENT' order by 1").split('\n')]
print(items, stu)
for k, sc in (('midterm', '7'), ('weekly', '8'), ('final', '6.5')):
    if k in items:
        for u in stu: print(k, L.api('gv.s8f', 'PUT', Q + '/grade-entries', {'item_id': items[k], 'student_id': u, 'score': sc})[0], end=' ')
def hp():
    s, d = L.api('gv.s8f', 'GET', Q + '/gradebook?limit=100'); return [(r.get('hp') or r.get('numeric_grade') or r.get('final')) for r in (d.get('rows') or d.get('items') or [])] if isinstance(d, dict) else d
before = hp(); doc = L.psql(f"select id from documents where course_id='{H3}' and type='COURSE_POLICY'")
cd, rd = L.api('gv.s8f', 'DELETE', Q + f'/documents/{doc}')
after = hp(); sc = L.psql(f"select status, source_document_id is null, source_title from grade_schemes where course_id='{H3}' and status='CONFIRMED'")
st = L.api('gv.s8f', 'GET', Q + '/grade-scheme/status')[1].get('state')
T('TC-P6-05-42', 'P' if cd in (200, 202, 204) and before == after and sc.startswith('CONFIRMED|t|') and st == 'CONFIRMED' else 'F', f'xoá tài liệu nguồn (DELETE documents → {cd}): scheme {sc}, status {st}; HP trước {before} = sau {after}; phần UI «Tài liệu nguồn đã gỡ» xem Playwright')
