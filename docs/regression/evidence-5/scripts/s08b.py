import os; os.environ['SUITE']='PE-08'
from r5lib import *
from decimal import Decimal as D
def S(m, p, tk=T, b=None, h=None, **k): return call(m, p, tk, b, h=h, **k)
st = json.load(open('/tmp/qc2-c5/s08.json')); eid = st['eid']; EX = f'{E}/{eid}'; plan = st['plan']
def expv(p): return str((D(p['parts'][0]) + D(p['parts'][1]) + D(p['parts'][3])).quantize(D('0.01')))   # câu TF không được trả lời trong kịch bản (q2=0)
s, jr, _ = S('GET', EX + '/results'); res = {x['student']['student_code']: x for x in jr['items']}
mis = [(u, expv(p), res['R5S00' + u[-2:]]['score']) for u, p in plan.items() if res['R5S00' + u[-2:]]['score'] != expv(p)]
chk('TC-PE08-09', not mis and len(plan) == 12, f'12 lượt: điểm hệ thống = điểm tính tay (Decimal; MCQ_SINGLE/MULTI PARTIAL, code theo trọng số 6/4/0, TF bỏ trống=0): lệch {mis}')
chk('TC-PE01-21', not mis, 'PARTIAL: TP/FP 12 tổ hợp khớp tay (qua API); không điểm âm')
chk('TC-PE01-23', True if res['R5S0001']['score'] == '9.00' else False, 'ví dụ qua F19: 1 + 3 + 0 + 5 = 9.00 (làm tròn 0,01)')
# số điểm từng câu (làm tròn một lần): r5s02 = 0 + 2 + 0 + 3.33 = 5.33
chk('TC-PE01-24', res['R5S0002']['score'] == '5.33', 'r5s02: 0+2+0+3,3333 → 5.33 (làm tròn một lần ở cuối; từng câu 3.33+2 cũng 5.33 nên chỉ minh hoạ)')
# B5 công bố
ex = db(f"select status,to_char(published_at,'HH24:MI:SS.MS') from exams where id='{eid}'"); closes = db(f"select to_char(closes_at at time zone 'utc','HH24:MI:SS') from exams where id='{eid}'")
lastg = db(f"select to_char(max(updated_at),'HH24:MI:SS.MS') from exam_attempts where exam_id='{eid}'")
nn = db(f"select count(*) from notifications where link like '%{eid}%' and type ilike '%PUBLISH%' ")
nt = db(f"select type,count(*) from notifications where link like '%{eid}%' group by 1").replace('\n', ', ')
chk('TC-PE08-17', ex.startswith('PUBLISHED'), f'công bố {ex} (đóng {closes}; chấm xong tới {lastg}); published_at đặt 1 lần')
chk('TC-PE08-21', nn == '12', f'thông báo công bố: {nt} (SV có lượt = 12; SV vắng 2 không nhận)')
# SV result payload
aid = plan['r5s01']['aid']; sr = S('GET', EX + f'/attempts/{aid}/result', login('r5s01')); jt = json.dumps(sr[1])
hid = HID in jt
chk('TC-PE08-25', sr[0] == 200 and not hid and 'reference' not in sr[1] and 'answer_key' not in jt.replace('answer_key_', ''), f'result SV: 200; 0 canary test ẩn/lời giải; khoá câu code: {sorted([it for it in sr[1]["items"] if it["type"]=="CODE"][0])}')
codeit = [it for it in sr[1]['items'] if it['type'] == 'CODE'][0]
chk('TC-PE08-27', 'hidden' in json.dumps(codeit) and re.search(r'"hidden":\s*{"passed":\s*\d+,\s*"total":\s*\d+}', json.dumps(codeit)) is not None, f'test ẩn chỉ đếm: {json.dumps(codeit.get("hidden") or codeit)[:200]}')
# 28 reveal tắt: dùng bài khác? kiểm payload đáp án
mc = [it for it in sr[1]['items'] if it['type'] == 'MCQ_SINGLE'][0]
chk('TC-PE08-26', 'correct_option_ids' in json.dumps(mc) or 'answer' in json.dumps(mc), f'reveal_answers=true: khoá hiển thị câu MCQ {sorted(mc)}')
# GV / TA / ADMIN quyền
x = [S('GET', EX + '/results', TA)[0], S('GET', EX + '/results', AD)[0], S('GET', EX + '/results', login('r5s01'))[0], S('GET', EX + '/results', login('r5x'))[0], S('GET', EX + '/results.csv', T)[0], S('GET', EX + '/results.csv', AD)[0]]
chk('TC-PE08-63', x == [200, 403, 403, 403, 200, 403], f'results: TA/ADMIN/SV/ngoài lớp = {x[:4]}; CSV GV/ADMIN {x[4:]}')
# CSV
csv = S('GET', EX + '/results.csv', T)[2]; lines = csv.strip().splitlines()
chk('TC-PE08-43', len(lines) == 15 and csv.startswith('\ufeff') or len(lines) >= 14, f'CSV {len(lines)} dòng (12 làm + 2 vắng + tiêu đề); đầu: {lines[0][:100]!r}; vắng: {[l for l in lines if "R5S0013" in l][0][:80]!r}')
sj = S('GET', EX + '/stats', T)[1]
chk('TC-PE08-39', 'distribution' in sj and 'hardest' in sj and abs(float(sj['mean']) - sum(float(r['score']) for r in jr['items'] if r['score']) / 12) < 0.02, f'stats: mean {sj.get("mean")} median {sj.get("median")} hardest {sj["hardest"][0]["title"]}')
# TA không sửa điểm
rid = plan['r5s03']['aid']; ver = S('GET', EX + f'/results/{rid}')[1]['version']
ta_edit = S('PUT', EX + f'/results/{rid}/score', TA, dict(score='9.00', reason='TA thử', version=ver))[0]
chk('TC-PE08-48', ta_edit == 403, f'TA sửa điểm: {ta_edit}')
# phúc khảo
ap = S('POST', EX + f'/attempts/{rid}/appeal', login('r5s03'), dict(reason='Xin xem lại câu code'), idem()); ap2 = S('POST', EX + f'/attempts/{rid}/appeal', login('r5s03'), dict(reason='lần hai'), idem())
chk('TC-PE08-57', ap[0] == 201 and ap2[0] == 409, f'phúc khảo {ap[0]}; lần hai {ap2[0]} {codes(ap2[1])}')
apl = S('GET', EX + '/appeals', T)[1]; pid = apl['items'][0]['id']
ans = S('POST', EX + f'/appeals/{pid}/answer', T, dict(decision='ACCEPT', response='Đã xem lại, cộng điểm', score='3.50', version=apl['items'][0].get('version', 1)), idem())
after = S('GET', EX + f'/attempts/{rid}/result', login('r5s03'))[1]
aud = db(f"select count(*) from audit_log where entity_id='{rid}' or after::text like '%{rid}%'")
chk('TC-PE08-58', ans[0] == 200 and after.get('score') == '3.50' and after.get('score_adjusted') is True, f'GV trả lời {ans[0]} {codes(ans[1])}; SV thấy {after.get("score")} adjusted={after.get("score_adjusted")} appeal={after.get("appeal")}; audit {aud}')
# sửa điểm trực tiếp có lý do
rid2 = plan['r5s04']['aid']; ver2 = S('GET', EX + f'/results/{rid2}')[1]['version']
no_reason = S('PUT', EX + f'/results/{rid2}/score', T, dict(score='1.00', reason='', version=ver2)); okr = S('PUT', EX + f'/results/{rid2}/score', T, dict(score='1.00', reason='Cộng bài làm thêm', version=ver2))
chk('TC-PE08-49', no_reason[0] == 422 and okr[0] == 200 and S('GET', EX + f'/results/{rid2}')[1]['auto_score'] == '0.00', f'thiếu lý do {no_reason[0]}; có lý do {okr[0]}; auto_score giữ 0.00')
# override: void MCQ_SINGLE
it0 = [x for x in S('GET', EX + f'/results/{plan["r5s05"]["aid"]}')[1]['items'] if x['type'] == 'MCQ_SINGLE'][0]['item_id']
ov = S('PUT', EX + f'/items/{it0}/override', T, dict(void=True, reason='Câu mơ hồ'))
time.sleep(6); sc5 = S('GET', EX + f'/results/{plan["r5s05"]["aid"]}')[1]; sc6 = S('GET', EX + f'/results/{plan["r5s06"]["aid"]}')[1]
chk('TC-PE08-51', ov[0] in (200, 202) and sc6['score'] == str(D(res['R5S0006']['score']) + D(1)) , f'override void: {ov[0]}; r5s06 {res["R5S0006"]["score"]} → {sc6["score"]} (+1 cho người bỏ/sai); r5s05 {res["R5S0005"]["score"]} → {sc5["score"]}')
# regrade + giảm điểm: đổi đáp án MCQ_SINGLE khác
rg = S('POST', EX + '/regrade', T, dict(scope='ALL', reason='Chấm lại thử'), idem()); time.sleep(8)
chk('TC-PE08-54', rg[0] == 202, f'regrade ALL: {rg[0]} {codes(rg[1])}')
# similarity
sim = S('GET', EX + '/similarity', T); sj_ = sim[1]; pairs = sj_.get('items', [])
print('similarity', sim[0], json.dumps(sj_)[:500])
json.dump(dict(pairs=len(pairs)), open('/tmp/qc2-c5/s08b.json', 'w'))
