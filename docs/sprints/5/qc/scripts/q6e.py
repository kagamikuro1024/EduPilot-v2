from q6lib import *
st = json.load(open('/tmp/q3/q6.json')); eid = st['eid']; A = E + '/' + eid
def ctx(user):
    tok = login(user); j = call('GET', A + '/attempts/mine', tok)[1]; aid = j['attempt']['id']; iid = [x for x in j['items'] if x['type'] == 'CODE'][0]['item_id']; H = {'X-Exam-Tab': str(uuid.uuid4())}; call('POST', A + f'/attempts/{aid}/takeover', tok, {}, H); return tok, aid, iid, H
# 19 judge sống nhưng hàng nộp đầy: bỏ qua; judge chết:
import subprocess
subprocess.run(['docker', 'stop', 'edupilot100-judge-1'], capture_output=True); time.sleep(20)
tok, aid, iid, H = ctx('sv.nguyco'); t0 = time.time(); s, j, t = call('POST', A + f'/attempts/{aid}/code/{iid}/run', tok, dict(language='cpp17', source=SUMSRC), {**H, **idem()}); print('19 judge chết:', s, '%.1fs' % (time.time() - t0), t[:240])
subprocess.run(['docker', 'start', 'edupilot100-judge-1'], capture_output=True); time.sleep(25)
s, j, t = call('POST', A + f'/attempts/{aid}/code/{iid}/run', tok, dict(language='cpp17', source=SUMSRC), {**H, **idem()}); print('19 judge sống lại:', s)
# 37 nộp bài thi khi KHÔNG có nháp / nháp rỗng: sv.nguyco (không nháp)
print('37 submit exam (không nháp)', call('POST', A + f'/attempts/{aid}/submit', tok, {}, {**H, **idem()})[0:1])
print('37 DB', db(f"select kind,status,auto from code_submissions where attempt_id='{aid}'") or '(không dòng SUBMIT/RUN... )', '|', db(f"select status,auto_score from exam_attempts where id='{aid}'"))
# 35 auto từ nháp: sv.kha lưu nháp rồi nộp bài thi
tk, ak, ik, Hk = ctx('sv.kha')
r = call('PUT', A + f'/attempts/{ak}/code/{ik}/draft', tk, dict(language='cpp17', source=SUMSRC, base_rev=0), Hk); print('35 nháp', r[0], r[2][:40]); 
print('35 submit exam', call('POST', A + f'/attempts/{ak}/submit', tk, {}, {**Hk, **idem()})[0:1]); time.sleep(3)
print('35 DB', db(f"select kind,status,auto,left(source_sha256,8) from code_submissions where attempt_id='{ak}' and kind='SUBMIT'"), '|', db(f"select status,auto_score from exam_attempts where id='{ak}'"))
# 36/34 sv.gioi đã có SUBMIT: nộp bài thi
tg, ag, ig, Hg = ctx('sv.gioi'); print('36 submit exam', call('POST', A + f'/attempts/{ag}/submit', tg, {}, {**Hg, **idem()})[0:1]); time.sleep(3)
print('36 DB', db(f"select kind,auto,is_final from code_submissions where attempt_id='{ag}' and kind='SUBMIT' order by created_at") if 0 else db(f"select kind,auto,status from code_submissions where attempt_id='{ag}' and kind='SUBMIT' order by created_at"), '|', db(f"select status,auto_score from exam_attempts where id='{ag}'"))
