from q6lib import *
import subprocess
st = json.load(open('/tmp/q3/q8.json')); plan = json.load(open('/tmp/q3/q8plan.json')); eid = st['eid']; qc = st['qc']; A = E + '/' + eid
sc = lambda: {r.split('|')[0].split('@')[0]: r.split('|')[1] for r in db(f"select u.email, coalesce(a.adjusted_score,a.auto_score) from exam_attempts a join users u on u.id=a.student_id where a.exam_id='{eid}'").split('\n')}
before = sc(); tv0 = db(f"select tests_version from question_code where question_id='{qc}'") if 0 else ''
r = call('POST', Q + '/' + qc + '/testcases', T, dict(name='hid-big', input='1000 1\n', expected='1001\n', is_sample=False)); print('thêm test ẩn', r[0], codes(r[1]))
print('52 regrade không Idempotency-Key', call('POST', A + '/regrade', T, dict(scope='all', reason='QC'))[0:1], '| TA', call('POST', A + '/regrade', TA, dict(scope='all', reason='QC'), idem())[0], 'SV', call('POST', A + '/regrade', login('sv04'), dict(scope='all', reason='QC'), idem())[0], '| scope lạ', call('POST', A + '/regrade', T, dict(scope='x', reason='QC'), idem())[0])
k = idem(); r = call('POST', A + '/regrade', T, dict(scope='all', reason='QC thêm test ẩn'), k); print('52 regrade all', r[0], r[2][:100]); r2 = call('POST', A + '/regrade', T, dict(scope='all', reason='QC thêm test ẩn'), k); print('   cùng khoá', r2[0], r2[1].get('job_id') == r[1].get('job_id'))
time.sleep(25); after = sc(); diff = {u: (before[u], after[u]) for u in before if before[u] != after[u]}; print('52 điểm đổi:', diff)
print('   kỳ vọng: small (sv05, sv10, sv15) giảm 0.75 (3/5 thay vì 3/4): ', {u: (plan[u]['ck']) for u in diff})
print('   trạng thái', db(f"select status, regrading from exams where id='{eid}'"), '| số thông báo:', db("select type,count(*) from notifications group by 1"))
# 55 phúc khảo
sva = login('sv06'); aid = plan['sv06']['aid']; AP = A + f'/attempts/{aid}/appeal'
print('55 reason rỗng', call('POST', AP, sva, dict(reason=''), idem())[0], '1001 ký tự', call('POST', AP, sva, dict(reason='x' * 1001), idem())[0])
r = call('POST', AP, sva, dict(reason='Em nghĩ câu code đúng'), idem()); print('55 tạo', r[0], r[2][:160]); print('55 lần hai', call('POST', AP, sva, dict(reason='lại'), idem())[0:1], codes(call('POST', AP, sva, dict(reason='lại'), idem())[1]))
print('55 SV vắng (sv20) / người khác', call('POST', A + f'/attempts/{aid}/appeal', login('sv20'), dict(reason='x'), idem())[0], '| ADMIN', call('POST', AP, AD, dict(reason='x'), idem())[0], 'GV', call('POST', AP, T, dict(reason='x'), idem())[0])
s, j, t = call('GET', A + '/appeals', T); print('59 danh sách phúc khảo GV', s, json.dumps(j, ensure_ascii=False)[:300], '| TA', call('GET', A + '/appeals', TA)[0], 'SV', call('GET', A + '/appeals', sva)[0], 'AD', call('GET', A + '/appeals', AD)[0])
pid = j['items'][0]['id']; ver = j['items'][0].get('version', 1); ANS = A + f'/appeals/{pid}/answer'
print('58 TA', call('POST', ANS, TA, dict(decision='UPHELD', response='x', version=ver))[0], '| thiếu response', call('POST', ANS, T, dict(decision='UPHELD', version=ver))[0], '| ADJUSTED thiếu score', call('POST', ANS, T, dict(decision='ADJUSTED', response='ok', version=ver))[0], '| score vượt', call('POST', ANS, T, dict(decision='ADJUSTED', response='ok', score='99', version=ver))[0], '| sai version', call('POST', ANS, T, dict(decision='UPHELD', response='ok', version=99))[0])
r = call('POST', ANS, T, dict(decision='ADJUSTED', response='Đã xem lại: cộng thêm', score='4.00', version=ver)); print('58 ADJUSTED', r[0], r[2][:200]); print('   lần hai', call('POST', ANS, T, dict(decision='UPHELD', response='ok', version=ver))[0:1], '| DB', db(f"select adjusted_score, auto_score from exam_attempts where id='{aid}'"), '| appeal', db("select status, score_before, score_after from exam_appeals order by created_at desc limit 1"))
s, j, _ = call('GET', A + f'/attempts/{aid}/result', sva); print('   SV thấy', j.get('score'), j.get('score_adjusted'), json.dumps(j.get('appeal'))[:160])
print('56 LLM trong exam:', subprocess.run("grep -rln 'internal/llm' ../../../../backend-go/internal/exam | head", shell=True, capture_output=True, text=True).stdout.strip().replace('\n', ' ') if False else '')
