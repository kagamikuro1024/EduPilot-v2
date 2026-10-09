from q6lib import *
st = json.load(open('/tmp/q3/q6.json')); eid = st['eid']; A = E + '/' + eid
tokK = login('sv.kha'); mj = call('GET', A + '/attempts/mine', tokK)[1]; aidK = mj['attempt']['id']; iidK = [x for x in mj['items'] if x['type'] == 'CODE'][0]['item_id']
s, j, t = call('POST', A + f'/attempts/{aidK}/code/{iidK}/run', tokK, dict(language='cpp17', source=SUMSRC), {'X-Exam-Tab': str(uuid.uuid4()), **idem()}); print('17 429 thân:', t[:250])
j = call('GET', A + '/attempts/mine', SV)[1]; aid = j['attempt']['id']; iid = [x for x in j['items'] if x['type'] == 'CODE'][0]['item_id']; H = {'X-Exam-Tab': str(uuid.uuid4())}; call('POST', A + f'/attempts/{aid}/takeover', SV, {}, H); CB = A + f'/attempts/{aid}/code/{iid}'
def sub(src, lang='cpp17'): return call('POST', CB + '/submit', SV, dict(language=lang, source=src), {**H, **idem()})
def poll(sid):
    for _ in range(40):
        s, j, t = call('GET', A + f'/attempts/{aid}/submissions/{sid}', SV)
        if j.get('status') in ('DONE', 'ERROR'): return j, t
        time.sleep(1)
    return j, t
s1 = sub(SUMSRC.replace('a+b', 'a+b+1')); print('23', s1[0], s1[2][:120]); print('24 hai lần trong 15 s', sub(SUMSRC)[0:1], sub(SUMSRC)[2][:200])
print('24 DB', db(f"select kind,status,left(source_sha256,8),auto from code_submissions where attempt_id='{aid}' order by created_at"))
time.sleep(16); s2 = sub(SUMSRC); print('24 sau 15 s', s2[0]); j1, t1 = poll(s1[1]['submission_id']); j2, t2 = poll(s2[1]['submission_id'])
print('26 lần 1', j1.get('status'), j1.get('is_final'), '| lần 2', j2.get('status'), j2.get('is_final'))
print('29 khoá của bản:', sorted(j2.keys()), 'mẫu:', sorted(j2['samples'][0].keys()) if j2.get('samples') else None, '| HID', HID in t2 or HID in t1, 'REF', REF in t2)
time.sleep(1); s3 = sub('int main(){ nope(); }'); print('24 ngay sau (<15s)', s3[0], s3[1].get('details'))
time.sleep(16); s3 = sub('int main(){ nope(); }'); j3, t3 = poll(s3[1]['submission_id']); print('30 CE', {k: j3.get(k) for k in ['status', 'compile_ok', 'is_final']}, (j3.get('compile_log') or '')[:80].replace('\n', ' '))
print('32 danh sách', json.dumps(call('GET', CB + '/submissions?limit=2', SV)[1], ensure_ascii=False)[:600])
print('45 SVB đọc bản của SVA', call('GET', A + f'/attempts/{aid}/submissions/{s2[1]["submission_id"]}', tokK)[0], '| SVB list lượt SVA', call('GET', CB + '/submissions', tokK)[0], '| TA', call('GET', CB + '/submissions', TA)[0], 'GV', call('GET', CB + '/submissions', T)[0], 'AD', call('GET', CB + '/submissions', AD)[0], '| run SVA bởi SVB', call('GET', A + f'/attempts/{aid}/runs/' + str(uuid.uuid4()), tokK)[0])
print('33 ok DB', db(f"select kind,status,auto,is_final from code_submissions where attempt_id='{aid}' order by created_at") if 0 else db(f"select kind,status,auto from code_submissions where attempt_id='{aid}' order by created_at"))
