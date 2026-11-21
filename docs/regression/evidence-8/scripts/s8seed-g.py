exec(open('s8seed.py').read())
import csv
tt = tok('teacher')
s, g = req('GET', f'/courses/{C1}/gradebook?limit=100', tt)
rows = g['rows']; by = {r['student']['student_code']: r for r in rows}
acc = {l.split('|')[0]: l.split('|')[1] for l in psql("select split_part(email,'@',1)||'|'||student_code from users where student_code is not null").split('\n')}
exp = list(csv.DictReader(open(f'{REPO}/seed/expected_final_grades.csv')))
ok = 0; bad = []; diffcode = 0
for e in exp:
    code = acc.get(e['account']); r = by.get(code)
    if code != e['student_code']: diffcode += 1
    if r and float(r['result']['hp']) == float(e['hp']) and r['result']['letter'] == e['letter']: ok += 1
    else: bad.append(e['account'])
sc = psql("select c.class_code||'='||count(*)||'/'||string_agg(s.status::text,',') from grade_schemes s join courses c on c.id=s.course_id group by c.class_code")
dr = psql("select count(*) from grade_schemes where status='DRAFT'"); ent = psql("select count(*)||'/'||count(distinct (item_id,student_id)) from grade_entries")
imp = psql("select count(*) from grade_import_batches") if False else ''
print(ok, len(exp), 'mã khác mặc định:', diffcode, bad[:5], sc, dr, ent, len(rows))
C('TC-P6-SEED-36', ok == 30 and diffcode >= 20, f'SEED_RNG=1 trên DB mới (qc2s): sổ điểm lớp 1 so seed/expected_final_grades.csv THEO account: {ok}/30 khớp HP + điểm chữ; {diffcode}/30 MSSV khác bản mặc định 20261029 (A,B,C cố định 20229001–3; ví dụ sv04…); sổ {len(rows)} dòng (log evidence-8/logs/winB-*.log). check-grade-seed.mjs theo MSSV báo lệch đúng thiết kế (so bằng MSSV) — đã so bằng account')
C('TC-P6-SEED-19', ok == 30 and dr == '0' and ent == '60/60' and sc.strip() == '761987=1/CONFIRMED', f'seed SEED_RNG=1 bị kill (PID node seed.mjs) ngay khi vào bước [15/20] Quy chế môn (extract → confirm), nhập điểm (log winB-kill-b.txt, winB-seed1b.log); chạy lại: rc=0, 4 phút 41 s (winB-seed2b.log); sau resume: grade_schemes {sc.strip()} (không bản nháp đôi, DRAFT={dr}), grade_entries {ent.strip()} (60 dòng, không trùng), sổ điểm {ok}/30 khớp expected theo account')
