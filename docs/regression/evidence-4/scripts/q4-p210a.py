import sys, json, random
sys.path.insert(0, '.'); from q4lib import *
TAG = uuid.uuid4().hex[:5]; random.seed(TAG)
def multipart(data, fname='roster.csv', fields=None):
    b = '----q2' + uuid.uuid4().hex
    body = (f'--{b}\r\nContent-Disposition: form-data; name="file"; filename="{fname}"\r\nContent-Type: application/octet-stream\r\n\r\n').encode() + data + b'\r\n'
    for k, v in (fields or {}).items(): body += (f'--{b}\r\nContent-Disposition: form-data; name="{k}"\r\n\r\n{v}\r\n').encode()
    return body + f'--{b}--\r\n'.encode(), f'multipart/form-data; boundary={b}'
def imp(cid, tok, csv, q='', key=None, fname='roster.csv'):
    body, ct = multipart(csv.encode() if isinstance(csv, str) else csv, fname)
    return call('POST', f'/courses/{cid}/roster/import{q}', tok, raw=body, hdr={'content-type': ct, 'Idempotency-Key': key or str(uuid.uuid4())}, idem=False)
def mssv(): return str(random.randint(700000000, 799999999))
def csv(rows): return 'Email,Họ và tên,MSSV\r\n' + '\r\n'.join(','.join(r) for r in rows) + '\r\n'
def e(n): return f'qc2s4.{TAG}.{n}@example.test'
T_ = L('teacher')
# ---------- P210-09: các mã dòng ----------
cid, cc, code = new_course('r'); settings(cid, capacity=3)
uB = student_pool(10)[5]; tB = L(uB); bcode = psql("select student_code from users where email='%s@edupilot.local'" % uB); resetjoin(); call('POST', '/courses/join', tB, {'code': code})
disabled = psql("select email from users where status='DISABLED' limit 1"); staff = 'teacher@edupilot.local'
dup_m = mssv()
rows = [(e(1), 'Ok Một', mssv()), (e(1), 'Trùng email', mssv()),            # dòng 3: DUPLICATE_EMAIL_IN_FILE
        (e(2), 'Lặp MSSV a', dup_m), (e(3), 'Lặp MSSV b', dup_m),             # dòng 5: DUPLICATE_STUDENT_CODE_IN_FILE
        (e(4), 'Trùng ghi danh', bcode),                                       # STUDENT_CODE_CONFLICT
        (e(5), 'Sai MSSV', 'ab'),                                              # INVALID_STUDENT_CODE
        (staff, 'GV', mssv()), (disabled, 'Bị khoá', mssv())]
d = imp(cid, T_, csv(rows), '?dry_run=true')
errs = {x.get('row'): (x.get('code') or x.get('reason')) for x in d[1].get('errors', [])}

rows2 = rows + [(e(6 + i), f'Thêm {i}', mssv()) for i in range(4)]
d2 = imp(cid, T_, csv(rows2), '?dry_run=true'); codes2 = [x.get('code') for x in d2[1].get('errors', [])]
want = {'DUPLICATE_EMAIL_IN_FILE', 'DUPLICATE_STUDENT_CODE_IN_FILE', 'STUDENT_CODE_CONFLICT', 'INVALID_STUDENT_CODE', 'EMAIL_BELONGS_TO_STAFF', 'EMAIL_DISABLED', 'COURSE_FULL'}
byrow = {x['row']: x['code'] for x in d2[1].get('errors', [])}
C('TC-P210-09', want <= set(codes2) and byrow.get(3) == 'DUPLICATE_EMAIL_IN_FILE' and byrow.get(5) == 'DUPLICATE_STUDENT_CODE_IN_FILE' and byrow.get(6) == 'STUDENT_CODE_CONFLICT' and byrow.get(7) == 'INVALID_STUDENT_CODE' and byrow.get(8) == 'EMAIL_BELONGS_TO_STAFF' and byrow.get(9) == 'EMAIL_DISABLED',
  f'dry_run tệp 13 dòng → từng dòng đúng mã: {byrow}; hai mã MSSV không lẫn nghĩa (DUPLICATE_STUDENT_CODE_IN_FILE ở dòng 5 vs STUDENT_CODE_CONFLICT ở dòng 6); COURSE_FULL xuất hiện khi vượt sĩ số 3')
# ---------- P210-20: không bản ghi đôi ----------
cid2, cc2, code2 = new_course('q'); settings(cid2)
uB2 = student_pool(10)[6]; bcode2 = psql("select student_code from users where email='%s@edupilot.local'" % uB2)
rows20 = [(f'{uB2}@edupilot.local', 'B trong roster', bcode2)] + [(e(100 + i), f'SV {i}', mssv()) for i in range(29)]
r1 = imp(cid2, T_, csv(rows20)); n_enr1 = psql(f"select count(*) from enrollments where course_id='{cid2}' and role_in_course='STUDENT'")
tB2 = L(uB2); resetjoin(); jn = call('POST', '/courses/join', tB2, {'code': code2}); n_enr2 = psql(f"select count(*) from enrollments where course_id='{cid2}' and role_in_course='STUDENT'")
r2 = imp(cid2, T_, csv(rows20)); mails_before = mp('/messages?limit=1').get('messages_count', mp('/messages?limit=1').get('total'))
r3 = imp(cid2, T_, csv([(e(300), 'A', 'D' + mssv()[:8]), (e(301), 'B', 'D' + mssv()[:8])]))
dupf = mssv(); r4 = imp(cid2, T_, csv([(e(310), 'X', dupf), (e(311), 'Y', dupf)])); r5 = imp(cid2, T_, csv([(e(312), 'Z', bcode2)]))
err4 = [x['code'] for x in r4[1].get('errors', [])]; err5 = [x['code'] for x in r5[1].get('errors', [])]
C('TC-P210-20', jn[0] == 200 and jn[1].get('already_member') is True and n_enr1 == n_enr2 and r2[1].get('already_member') == r2[1].get('total') and err4 == ['DUPLICATE_STUDENT_CODE_IN_FILE'] and err5 == ['STUDENT_CODE_CONFLICT'],
  f'import 30 dòng → {r1[0]} (enrollments STUDENT {n_enr1}); B tự join → {jn[0]} already_member={jn[1].get("already_member")}, enrollments vẫn {n_enr2}; import lại cùng tệp: already_member {r2[1].get("already_member")} = total {r2[1].get("total")}; MSSV lặp trong tệp → {err4}; MSSV của B ở email khác → {err5}')
# ---------- P210-24: thư mời ----------
cid3, cc3, code3 = new_course('p'); settings(cid3)
ex_users = student_pool(10)[7:10]
em24 = [e(400 + i) for i in range(27)]; ms24 = [mssv() for _ in range(27)]
rows24 = [(a, f'Mời {i}', m) for i, (a, m) in enumerate(zip(em24, ms24))] + [(f'{u}@edupilot.local', f'Có sẵn {u}', psql("select student_code from users where email='%s@edupilot.local'" % u)) for u in ex_users]
t0 = iso = time.strftime('%Y-%m-%dT%H:%M:%S', time.gmtime(time.time() - 3))
r = imp(cid3, T_, csv(rows24)); time.sleep(12)
def mails():
    got = {}
    for m in mp('/messages?limit=500').get('messages', []):
        for a in m['To']:
            if a['Address'] in em24 and 'Bạn được thêm vào lớp' in m['Subject']: got.setdefault(a['Address'], []).append(m)
    return got
g1 = mails(); r_again = imp(cid3, T_, csv(rows24)); time.sleep(10); g2 = mails()
body = mp('/message/' + g1[em24[0]][0]['ID']) if g1 else {}; text = (body.get('Text') or '') + (body.get('HTML') or '')
leak = sum(text.count(m) for m in ms24)
off = [e(500 + i) for i in range(4)]; r_off = imp(cid3, T_, csv([(a, f'Tắt {i}', mssv()) for i, a in enumerate(off)]), '?send_invites=false'); time.sleep(8)
off_mail = [m for m in mp('/messages?limit=500').get('messages', []) if any(a['Address'] in off for a in m['To'])]
off_state = psql("select string_agg(distinct status::text, ',') from users where email in (%s)" % ','.join("'%s'" % a for a in off))
C('TC-P210-24', len(g1) == 27 and all(len(v) == 1 for v in g2.values()) and len(g2) == 27 and not off_mail and off_state == 'INVITED' and 'Lớp' in text and leak == 0,
  f'import 30 dòng (27 chưa có tài khoản): {len(g1)} thư «Bạn được thêm vào lớp» (mỗi người 1); import lại: tổng thư mỗi người vẫn {sorted({len(v) for v in g2.values()})} (không gửi lại); send_invites=false (4 dòng mới): {len(off_mail)} thư, tài khoản {off_state}; thư mẫu: «{text[:160]}»; MSSV trong thư: {leak}; tên GV trong thư: {"Teacher" in text or "Giảng" in text}')
