from q8a import *
eid, qs, qc, o, c = mkexam8('QC8 hold', reveal=False, appeal_days=0); A = E + '/' + eid; json.dump(dict(eid=eid, c=c.isoformat()), open('/tmp/q3/q8e.json', 'w'))
print('hold', call('PUT', A + '/publish-hold', T, dict(hold=True, version=ge(eid)['version'])))
time.sleep(max(0, (o - dt.datetime.now(dt.timezone.utc)).total_seconds()) + 3)
rows = db(f"select i.id, q.title from exam_items i join question_bank q on q.id=i.question_id where i.exam_id='{eid}'").split('\n'); idx = {r.split('|')[0]: int(r.split('|')[1].split('-')[1]) for r in rows}
pl = {}
for u in ('sv17', 'sv18', 'sv19'):
    tok = login(u); r = call('POST', A + '/attempts', tok, {}, idem()); aid = r[1]['attempt']['id']; H = {'X-Exam-Tab': str(uuid.uuid4())}; call('POST', A + f'/attempts/{aid}/takeover', tok, {}, H); ans = []
    for it in r[1]['items']:
        if it['type'] == 'MCQ_SINGLE': ans.append(dict(item_id=it['item_id'], answer=dict(option_ids=[it['options'][0]['id']])))
    call('PUT', A + f'/attempts/{aid}/answers', tok, dict(items=ans), H); pl[u] = (aid, H['X-Exam-Tab'])
    print(u, call('POST', A + f'/attempts/{aid}/submit', tok, {}, {'X-Exam-Tab': H['X-Exam-Tab'], **idem()})[0])
json.dump(pl, open('/tmp/q3/q8epl.json', 'w')); print('xong')
