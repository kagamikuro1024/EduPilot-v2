"""QC2 smoke sprint 5 (SmkL) — gợi ý AI (provider fake) + test chưa duyệt: TC-PE03-35/37/38/40/41, TC-PE02-21. Không đổi cấu hình toàn cục."""
import os, sys; os.environ['SUITE'] = 'PE-03L'
from r5lib import *
import rl
def R(tc, ok, note): rl.R(tc, 'P' if ok is True else 'F' if ok is False else ok, note)
def S(m, p, tk=None, b=None, h=None, **k): return call(m, p, tk or T, b, h=h, **k)
SVT = login('r5s23'); ADT = AD
def wait(job, n=120):
    for _ in range(n):
        j = S('GET', '/jobs/' + job)[1]
        if j.get('status') in ('SUCCEEDED', 'FAILED', 'DEAD'): return j
        time.sleep(1)
    return j
def prompts(since, course=None):
    q = f"select coalesce(string_agg(p.masked_prompt, ' ### '),'') from llm_request_payloads p join llm_audit a on a.id=p.audit_id where a.task='QUESTION_GEN' and a.created_at >= '{since}'"
    return db(q)
TOPIC = 'Mật mã đối xứng QCL' + uuid.uuid4().hex[:5]
# ---- TC-PE03-35
t0 = db("select now()")
s, j, _ = S('POST', Q + '/suggest', b=dict(kind='MCQ', topic=TOPIC, difficulty='MEDIUM', count=5), h=idem()); job = j.get('job_id'); r = wait(job); res = r.get('result') or {}
uid_t = db("select id from users where email='r5t@edupilot.local'")
rows = db(f"select origin,review_status,created_by from question_bank where id = any(string_to_array(replace(replace('{','',''),'}',''),',')::uuid[])") if False else ''
created = res.get('created') or []
inl = "','".join(created)
rows = db(f"select origin||'/'||review_status||'/'||(created_by='{uid_t}')::text from question_bank where id in ('{inl}')").splitlines() if created else []
aud = db(f"select count(*), min(task), min(lane) from llm_audit where task='QUESTION_GEN' and created_at >= '{t0}' and course_id='{C1}'")
R('TC-PE03-35', s == 202 and r.get('status') == 'SUCCEEDED' and created and set(rows) == {'AI_DRAFT/PENDING/true'} and aud.startswith('1|QUESTION_GEN|BATCH') and 'dropped' in res, f'suggest MCQ ×5: {s}; job {r.get("status")}; tạo {len(created)} câu: {sorted(set(rows))} (created_by = GV bấm); llm_audit QUESTION_GEN: {aud} (1 dòng, lane BATCH); result.dropped={res.get("dropped")}')
# ---- TC-PE03-37 riêng tư prompt
time.sleep(2)
blob = prompts(t0); mine = [1] if TOPIC in blob else []
leak = [x for x in ('R5 SV', 'R5S00', '@edupilot.local', 'r5s', 'r5t@', 'Edupilot#') if x in blob]
src6000 = 'a' * 6000; s1, j1, _ = S('POST', Q + '/suggest', b=dict(kind='MCQ', topic=TOPIC + 'x', count=1, source_text=src6000), h=idem()); s2, j2, _ = S('POST', Q + '/suggest', b=dict(kind='MCQ', topic=TOPIC + 'y', count=1, source_text='a' * 6001), h=idem())
if j1.get('job_id'): wait(j1['job_id'])
R('TC-PE03-37', len(mine) >= 1 and not leak and s1 == 202 and s2 == 422, f'prompt đã che lưu ở llm_request_payloads (provider fake chạy trong worker nên /_test/llm/payloads của gateway không thấy; {len(mine)} prompt chứa chủ đề của QC): chủ đề/độ khó/nguồn; chứa tên/MSSV/email/mật khẩu SV·GV: {leak or "không"}; source_text 6000 ký tự → {s1}; 6001 → {s2} {codes(j2)}')
# ---- TC-PE03-38
one = created[0] if created else None
sv_get = S('GET', f'{Q}/{one}', SVT)[0] if one else None
e_x, _ = None, None
eid = S('POST', E, b=dict(title='R5L 03-38', opens_at=iso(hours=3), closes_at=iso(hours=5), duration_minutes=30), h=idem())[1]['id']; g = S('GET', f'{E}/{eid}')[1]
it = S('PUT', f'{E}/{eid}/items', b=dict(items=[dict(question_id=one, points='1')], version=g['version']))
okq = mkq(77); g = S('GET', f'{E}/{eid}')[1]; S('PUT', f'{E}/{eid}/items', b=dict(items=[dict(question_id=okq, points='1')], version=g['version'])); g = S('GET', f'{E}/{eid}')[1]
sched = S('POST', f'{E}/{eid}/schedule', b={})
pv = S('GET', f'{E}/{eid}/preview')[2]
R('TC-PE03-38', it[0] == 422 and 'ITEM_NOT_APPROVED' in codes(it[1]) and sv_get == 403 and '"origin"' not in pv.replace('"origin_', ''), f'đưa câu AI PENDING vào bài: {it[0]} {codes(it[1])}; SV đọc câu AI: {sv_get}; bản xem trước (đúng định dạng SV) có trường origin: {"origin" in pv}')
# ---- TC-PE03-40/41 + PE02-21: câu CODE đã verify
qa = mkcode(83, tests=[('ok1', '1 2\n', '3\n', True, 1), ('ok2', '5 6\n', '11\n', False, 2)], ref=SUM, verify=True, approve_q=True); P = f'{Q}/{qa}'
t1 = db('select now()')
s, j, _ = S('POST', Q + '/suggest', b=dict(kind='CODE_TESTS', question_id=qa, count=5), h=idem()); r = wait(j.get('job_id')); res = r.get('result') or {}
tl = S('GET', P + '/testcases?limit=100')[1]['items']; ai = [x for x in tl if x.get('source') == 'AI_DRAFT']
def refout(inp): a, b = map(int, inp.split()); return f'{a+b}\n'
good_exp = [x for x in ai if x.get('expected') and all(c.isdigit() or c in '-\n' for c in x['expected'])]
mism = [x['name'] for x in ai if x.get('input') and x.get('expected') is not None and (lambda i, e: (len(i.split()) == 2 and e != refout(i)))(x['input'], x['expected'])]
pj = prompts(t1)
R('TC-PE03-40', s == 202 and r.get('status') == 'SUCCEEDED' and ai and all(x.get('approved') is False for x in ai) and not mism, f'suggest CODE_TESTS ×5: {s}, job {r.get("status")}; {len(ai)} test mới source=AI_DRAFT approved={sorted({x.get("approved") for x in ai})}; expected = đầu ra lời giải mẫu (QC tự tính a+b ngoài hệ thống): lệch {mism}; dropped={res.get("dropped")}; prompt gửi AI có expected/reference: {"expected" in pj and "reference" in pj}')
# PE03-41: test AI chưa duyệt không được chấm
gq = S('GET', P)[1]; st_q = gq['review_status']; print('câu sau AI tests:', st_q, gq['code']['tests'])
if st_q != 'APPROVED': print('re-approve', approve(qa))
e1, _ = mkexam(open_s=14, win=900, dur=10, items=[qa], title='R5L 03-41'); time.sleep(16)
aid, jj, H, s_, _ = start(e1, 'r5s24'); itm = [i for i in jj['items'] if i['type'] == 'CODE'][0]['item_id']; tg = login('r5s24')
s, rj, _ = S('POST', f'{E}/{e1}/attempts/{aid}/code/{itm}/submit', tg, dict(language='cpp17', source=SUM), {**H, **idem()}); sid = rj.get('submission_id')
for _ in range(90):
    row = db(f"select status,verdict,passed_weight,total_weight,jsonb_array_length(results) from code_submissions where id='{sid}'")
    if row.startswith('DONE'): break
    time.sleep(1)
base_tw = row.split('|')[3]
n_un = len(ai)
qa2 = mkcode(85, tests=[('ok1', '1 2\n', '3\n', True, 1), ('ok2', '5 6\n', '11\n', False, 2)], ref=SUM, verify=True, approve_q=True); P2 = f'{Q}/{qa2}'
sg = S('POST', Q + '/suggest', b=dict(kind='CODE_TESTS', question_id=qa2, count=5), h=idem()); wait(sg[1]['job_id'])
ids = [x['id'] for x in S('GET', P2 + '/testcases?limit=100')[1]['items'] if x.get('source') == 'AI_DRAFT']
ap_in_use = S('POST', P + '/testcases/approve', b=dict(ids=[x['id'] for x in ai]))
ap = S('POST', P2 + '/testcases/approve', b=dict(ids=ids)); reapprove = None
if S('GET', P2)[1]['review_status'] != 'APPROVED':
    v = S('POST', P2 + '/reference/verify', b={}, h=idem())[1]; wait(v['job_id']); reapprove = approve(qa2)
n_un = len(ai)
# thiếu reference → 422 REFERENCE_REQUIRED
qn = S('POST', Q, b=dict(type='CODE', title='r5l-noref', topic='R5', stem='x'))[1]['id']
sn = S('POST', Q + '/suggest', b=dict(kind='CODE_TESTS', question_id=qn, count=3), h=idem())
e2, _ = mkexam(open_s=14, win=900, dur=10, items=[qa2], title='R5L 03-41b'); time.sleep(16)
aid2, jj2, H2, _, _ = start(e2, 'r5s25'); itm2 = [i for i in jj2['items'] if i['type'] == 'CODE'][0]['item_id']; tg2 = login('r5s25')
s, rj2, _ = S('POST', f'{E}/{e2}/attempts/{aid2}/code/{itm2}/submit', tg2, dict(language='cpp17', source=SUM), {**H2, **idem()}); sid2 = rj2.get('submission_id')
for _ in range(90):
    row2 = db(f"select status,verdict,passed_weight,total_weight from code_submissions where id='{sid2}'")
    if row2.startswith('DONE'): break
    time.sleep(1)
R('TC-PE03-41', ap[0] == 200 and int(base_tw) == 3 and int(row2.split('|')[3]) > 3 and sn[0] == 422 and 'REFERENCE_REQUIRED' in codes(sn[1]), f'trước duyệt: {n_un} test AI không có trong kết quả, total_weight={base_tw} ({row.split("|")[:3]}); sau POST testcases/approve ({len(ids)} test AI, trước khi câu vào bài; duyệt khi câu đã ở bài SCHEDULED+ → {ap_in_use[0]} QUESTION_IN_USE): total_weight={row2.split("|")[3]} (gồm test AI); câu không có reference gọi suggest tests → {sn[0]} {codes(sn[1])}; câu bị hạ khỏi APPROVED sau khi duyệt test: {reapprove is not None}')
# ---- TC-PE02-21 lượt chuẩn: test AI chưa duyệt (approved=false) + duyệt; Σweight=0 đã chạy ở s13 w0 (IE)
qb = mkcode(84, tests=[('ok1', '1 2\n', '3\n', True, 1), ('ok2', '5 6\n', '11\n', False, 2)], ref=SUM, verify=True, approve_q=True); PB = f'{Q}/{qb}'
S('POST', '/_test/llm/payloads/reset', ADT)
s, j, _ = S('POST', Q + '/suggest', b=dict(kind='CODE_TESTS', question_id=qb, count=4), h=idem()); wait(j['job_id'])
if S('GET', PB)[1]['review_status'] != 'APPROVED': print('re-approve', approve(qb))
e3, _ = mkexam(open_s=14, win=900, dur=10, items=[qb], title='R5L 02-21'); time.sleep(16)
aid3, jj3, H3, _, _ = start(e3, 'r5s26'); itm3 = [i for i in jj3['items'] if i['type'] == 'CODE'][0]['item_id']; tg3 = login('r5s26')
s, rj3, _ = S('POST', f'{E}/{e3}/attempts/{aid3}/code/{itm3}/submit', tg3, dict(language='cpp17', source=SUM), {**H3, **idem()}); sid3 = rj3.get('submission_id')
for _ in range(90):
    row3 = db(f"select status,verdict,passed_weight,total_weight,jsonb_array_length(results) from code_submissions where id='{sid3}'")
    if row3.startswith('DONE'): break
    time.sleep(1)
nun = db(f"select count(*) from code_testcases where problem_id='{qb}' and not approved"); nap = db(f"select count(*) from code_testcases where problem_id='{qb}' and approved")
R('TC-PE02-21', row3.startswith('DONE|AC|3|3|2') and int(nun) > 0 or (row3.split('|')[3] == '3' and int(nun) > 0), f'câu có {nap} test đã duyệt (w=1+2) + {nun} test AI chưa duyệt (approved=false): chấm {row3} (verdict|passed|total|số test trong kết quả) → chỉ test duyệt được chấm; Σweight=0: ca s13_judge.py w0 → IE, không panic/chia 0 (nhánh bằng SQL trên fixture vì API chặn duyệt câu Σweight=0: 422 TOTAL_WEIGHT_ZERO)')
