exec(open('s8seed.py').read())
import hashlib, urllib.request
tt = tok('teacher'); Q = f'/courses/{C2}'
def put(url, headers, b):
    r = urllib.request.Request(url, data=b, method='PUT', headers=headers)
    ctx = __import__('ssl').create_default_context(); ctx.check_hostname = False; ctx.verify_mode = __import__('ssl').CERT_NONE
    urllib.request.urlopen(r, context=ctx, timeout=120).read()
def upload(path, title, typ='COURSE_POLICY'):
    b = open(f'{REPO}/{path}', 'rb').read(); sha = hashlib.sha256(b).hexdigest()
    s, pre = req('POST', f'{Q}/uploads/presign', tt, {'purpose': 'document', 'filename': os.path.basename(path), 'mime_type': 'application/pdf', 'size_bytes': len(b), 'sha256': sha}); assert s == 200, (s, pre)
    put(pre['url'], pre['headers'], b)
    s, d = req('POST', f'{Q}/uploads/complete', tt, {'upload_id': pre['upload_id'], 'title': title, 'type': typ}); assert s in (200, 201, 202), (s, d)
    for _ in range(80):
        st = psql(f"select status from documents where course_id='{C2}' and title='{title}'")
        if st in ('READY', 'FAILED'): break
        time.sleep(3)
    return psql(f"select id from documents where course_id='{C2}' and title='{title}'"), st
def status(): return req('GET', f'{Q}/grade-scheme/status', tt)[1]
if sys.argv[1:] == ['34']:
    doc, st = upload('seed/demo/quy-che-lop1.pdf', 'QC2 PDF lạ (nội dung lớp 1, khác bản ghi lớp 2)', 'OTHER') if False else (None, None)
    import zipfile
    # PDF tuỳ ý: tạo PDF 1 trang bằng văn bản lạ (không có bản ghi trong fake_llm.json)
    pdf = b'%PDF-1.4\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj 2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj 3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 300 200]/Contents 4 0 R/Resources<</Font<</F1 5 0 R>>>>>>endobj 4 0 obj<</Length 60>>stream\nBT /F1 12 Tf 20 100 Td (Quy che la QC2-khong-co-ban-ghi) Tj ET\nendstream endobj 5 0 obj<</Type/Font/Subtype/Type1/BaseFont/Helvetica>>endobj\ntrailer<</Root 1 0 R/Size 6>>\n%%EOF\n'
    open(f'{REPO}/docs/regression/evidence-8/qc2-unknown-policy.pdf', 'wb').write(pdf)
    pdf = open(f'{REPO}/seed/documents/Quyche.pdf', 'rb').read() + b'\n%qc2-seed34\n'; sha = hashlib.sha256(pdf).hexdigest(); s, pre = req('POST', f'{Q}/uploads/presign', tt, {'purpose': 'document', 'filename': 'Quyche-qc2.pdf', 'mime_type': 'application/pdf', 'size_bytes': len(pdf), 'sha256': sha}); put(pre['url'], pre['headers'], pdf)
    s, d = req('POST', f'{Q}/uploads/complete', tt, {'upload_id': pre['upload_id'], 'title': 'QC2 quy chế lạ', 'type': 'COURSE_POLICY'}); print('complete', s, str(d)[:100])
    for _ in range(120):
        st = psql(f"select status from documents where course_id='{C2}' and title='QC2 quy chế lạ'")
        if st in ('READY', 'FAILED'): break
        time.sleep(3)
    doc = psql(f"select id from documents where course_id='{C2}' and title='QC2 quy chế lạ'"); print('doc', st)
    drafts0 = psql(f"select count(*) from grade_schemes where course_id='{C2}'")
    t0 = time.time(); s, d = req('POST', f'{Q}/grade-scheme/extract', tt, {'document_id': doc}); print('extract', s, str(d)[:200])
    for _ in range(40):
        stt = status()
        if stt.get('state') not in ('EXTRACTING',): break
        time.sleep(3)
    dt = time.time() - t0; drafts1 = psql(f"select count(*) from grade_schemes where course_id='{C2}'")
    job = psql(f"select status||'|'||coalesce(error,'') from jobs where kind='grade_scheme.extract' order by created_at desc limit 1")
    print(stt, job, drafts0, drafts1)
    C('TC-P6-SEED-34', dt < 120 and drafts0 == drafts1 and ('EXTRACTION' in job or 'bản ghi' in job or 'FAILED' in job or stt.get('state') in ('EXTRACTION_FAILED', 'NOT_EXTRACTED')), f'PDF lạ (seed/documents/Quyche.pdf — quy chế đào tạo của trường, KHÔNG có bản ghi trong fake_llm.json; gửi làm COURSE_POLICY) ở lớp 2 với provider giả/DEMO: extract {s} {str(d)[:80]}; sau {dt:.0f} s trạng thái {json.dumps(stt, ensure_ascii=False)[:160]}; job: «{job[:140]}»; số grade_schemes của lớp 2 trước/sau {drafts0}/{drafts1} (không bản nháp rác); không treo')
    sys.exit()
doc, st = upload('seed/demo/quy-che-lop2.pdf', 'Quy chế lớp 2')
print('doc', doc, st, status())
s, d = req('POST', f'{Q}/grade-scheme/extract', tt, {'document_id': doc}); print('extract', s, str(d)[:150])
for _ in range(60):
    stt = status()
    if stt.get('state') not in ('EXTRACTING', 'NOT_EXTRACTED', None): break
    time.sleep(3)
s, g = req('GET', f'{Q}/grade-scheme', tt); dr = g.get('draft') or {}
comps = {c['key']: (c['name'], c['weight']) for c in dr.get('components', [])}
rules = {r['kind']: r.get('params') for r in dr.get('rules', [])}
print(stt, comps, json.dumps(rules, ensure_ascii=False)[:600], dr.get('unclear_items'))
exp = json.load(open(f'{REPO}/docs/specs/FEAT-gradebook/fixtures/extraction-lop2.json'))
ew = sorted(str(c.get('weight')) for c in exp.get('components', []))
gw = sorted(str(c['weight']) for c in dr.get('components', []))
unc = [u.get('code') or u for u in dr.get('unclear_items', [])]
cn = psql(f"select count(*) from grade_schemes where course_id='{C2}' and status='DRAFT'")
bonus = rules.get('BONUS', {}); pen = rules.get('PENALTY', {})
C('TC-P6-SEED-14', stt.get('state') == 'DRAFT_UNCLEAR' and len(comps) == 3 and sorted(float(x) for x in gw) == [25.0, 25.0, 50.0] and len(unc) == 1 and 'ROUNDING' in json.dumps(unc), f'lớp 2 (provider giả DEMO): upload quy-che-lop2.pdf READY → extract {s} → state {stt.get("state")}, unclear_count {stt.get("unclear_count")}; 3 thành phần trọng số {gw} (đúng 25/25/50); BONUS {json.dumps(bonus)}, PENALTY {json.dumps(pen)}; unclear_items {json.dumps(unc, ensure_ascii=False)}; fixture extraction-lop2.json weights {ew}; số bản nháp {cn}. Ghi chú: luồng đi bằng API thật (giao diện «Tải quy chế» + tooltip khoá đã P ở S8-UI/05-02..05-04 trên ep-r với LLM thật)')
