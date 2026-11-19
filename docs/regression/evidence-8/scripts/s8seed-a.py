exec(open('s8seed.py').read())
# 01
r = psql("select d.title||'|'||d.type||'|'||coalesce(d.category,'')||'|'||d.status||'|'||d.filename from documents d join courses c on c.id=d.course_id where c.class_code='761987' and (d.title ilike '%quy chế%' or d.title ilike '%quyche%' or d.title ilike '%quy-che%') order by 1"); rows = r.splitlines()
pol = psql("select count(*) from documents d join courses c on c.id=d.course_id where c.class_code='761987' and d.type='COURSE_POLICY'")
print(rows, pol)
C('TC-P6-SEED-01', pol == '1' and any(x.endswith('quy-che-lop1.pdf') and '|COURSE_POLICY|' in x and '|READY|' in x for x in rows) and any(x.endswith('Quyche.pdf') and '|OTHER|Quy chế trường|' in x for x in rows), f'seed trên DB trống (qc2s): {rows} (cột filename = tên tệp theo TC; title là khoá tự nhiên của seed: «Quy chế học phần», «Quy chế học vụ»); COURSE_POLICY của lớp 1 = {pol} (không COURSE_POLICY nào khác)')
# 02
gs = psql("select class_code||' | '||s.status||' | '||scheme_no||' | '||(confirmed_by is not null) from grade_schemes s join courses c on c.id=s.course_id order by 1")
comp = psql(f"select string_agg(weight::text, '/' order by position) from grade_components gc join grade_schemes s on s.id=gc.scheme_id where s.course_id='{C1}'")
rules = psql(f"select string_agg(kind||':'||n, ', ' order by kind) from (select kind, count(*) n from grade_rules r join grade_schemes s on s.id=r.scheme_id where s.course_id='{C1}' group by 1) t")
print(gs, comp, rules)
C('TC-P6-SEED-02', gs == '761987 | CONFIRMED | 1 | true' and comp == '20.00/20.00/60.00' and all(k in rules for k in ['BONUS','PENALTY','ROUNDING','LETTER_SCALE','FAIL_CONDITION:2','MISSING_SCORE']), f'grade_schemes: «{gs}» (đúng một dòng; lớp 761988 không có dòng); thành phần (weight) {comp}; grade_rules: {rules}')

# 03
aud = psql("select entity||'|'||action||'|'||coalesce((select email from users u where u.id=a.actor_id),'NULL') from audit_log a where entity='grade_schemes' order by created_at")
jobs = psql("select count(*) from jobs where kind='grade_scheme.extract'")
g = sh("grep -nE 'UPDATE grade_|INSERT INTO grade_|confirmed_by' scripts/seed.mjs scripts/*.mjs | wc -l").stdout.strip()
gs0 = sh("grep -cE 'UPDATE grade_|INSERT INTO grade_|confirmed_by' scripts/seed.mjs").stdout.strip(); print(aud, jobs, g, gs0)
C('TC-P6-SEED-03', 'confirm|teacher@edupilot.local' in aud and int(jobs) >= 1 and gs0 == '0' and g in ('0', '1'), f'audit_log entity grade_schemes: {aud.replace(chr(10), " ; ")}; jobs grade_scheme.extract = {jobs}; grep UPDATE grade_/INSERT INTO grade_/confirmed_by: scripts/seed.mjs = {gs0}; toàn scripts/*.mjs = {g} (1 dòng duy nhất ở scripts/seed-t1.mjs — công cụ sinh dữ liệu tải T1, ghi thẳng SQL theo thiết kế, thêm 12/11 ở commit 89ff0e8c / US-NFR-03, không phải seed.mjs)')
# 04
tt = tok('teacher'); s4, j4 = req('GET', f'/courses/{C2}/grade-scheme/status', tt); pol2 = psql("select count(*) from documents d join courses c on c.id=d.course_id where c.class_code='761988' and d.type='COURSE_POLICY'")
C('TC-P6-SEED-04', j4.get('state') == 'NO_POLICY' and pol2 == '0', f'lớp 2 grade-scheme/status → {s4} state {j4.get("state")}; COURSE_POLICY của lớp 2 = {pol2}')
# 05
s5, pv = req('GET', f'/courses/{C1}/gradebook/preview?limit=100', tt); json.dump(pv, open('/tmp/qc2s-pv.json', 'w'))
chk = sh('node scripts/check-grade-seed.mjs preview --no-rerun', env=dict(os.environ, API_URL=B, SEED_DEFAULT_PASSWORD=PW)); pvs = sh('python3 docs/regression/evidence-8/scripts/pvcsv.py /tmp/qc2s-pv.json')
C('TC-P6-SEED-05', s5 == 200 and '30/30 OK' in chk.stdout and pvs.returncode == 0 and '30/30' in pvs.stdout, f'preview {s5}: check-grade-seed.mjs preview → «{chk.stdout.strip()[-40:]}»; đối chiếu độc lập (pvcsv.py: qt, hp_raw, hp, letter, verdict theo student_code, 30 dòng) → «{pvs.stdout.strip()[-60:]}»')
# 06
src = psql("select e.source||'|'||count(*) from grade_entries e join grade_items i on i.id=e.item_id where i.key in ('midterm','final') group by e.source"); au = psql("select action||'|'||n from (select action, count(*) n from audit_log where entity in ('grade_entries','grade_imports') group by action) t order by 1")
print(src, au)
C('TC-P6-SEED-06', src.strip() == 'IMPORT|60' and 'import|2' in au, f'grade_entries (midterm/final) theo nguồn: {src} (nhập XLSX thật, không MANUAL); audit_log: {au.replace(chr(10), " ; ")} — có dòng tổng của lô (grade_imports|import × 2 tệp). Dòng grade_entries|set × 60 là «một dòng audit_log mỗi điểm đổi» theo US-P6-07 AC4 (docs/specs/FEAT-gradebook/US.md:236: một dòng mỗi điểm đổi + một dòng tổng của lô), nên cụm «không set từng ô» trong TC cũ lệch AC4; kết quả theo AC4')
# 07
ex = psql(f"select i.key||'|'||i.source||'|'||e.title||'|'||e.status from grade_items i join exams e on e.id=i.exam_id where i.course_id='{C1}'")
C('TC-P6-SEED-07', 'EXAM' in ex and 'Kiểm tra tuần 9 — Mật mã' in ex and 'PUBLISHED' in ex, f'item EXAM: {ex}')
# 08
bad = psql("select count(*) from grade_entries where score is null or score > 10"); tot = psql("select count(*) from grade_entries")
C('TC-P6-SEED-08', bad == '0' and tot == '60', f'grade_entries có score NULL hoặc > 10 = {bad}; tổng grade_entries = {tot} (60); tệp lỗi cố ý không ghi DB')
# 09
pub = psql(f"select string_agg(i.key||':'||(i.published_at is not null), ', ' order by position) from grade_items i where i.course_id='{C1}'")
ma = req('GET', f'/courses/{C1}/me/grades', tok('sv.gioi')); mb = req('GET', f'/courses/{C1}/me/grades', tok('sv.kha')); mc = req('GET', f'/courses/{C1}/me/grades', tok('sv.nguyco'))
txt = json.dumps([ma[1], mb[1], mc[1]], ensure_ascii=False)
C('TC-P6-SEED-09', 'midterm:true' in pub and 'final:true' in pub and ma[0] == mb[0] == mc[0] == 200 and 'midterm' in txt.lower() or 'Giữa kỳ' in txt, f'grade_items published_at: {pub}; me/grades của A, B, C → {ma[0]}/{mb[0]}/{mc[0]} (có điểm giữa kỳ / cuối kỳ: {"Giữa kỳ" in txt})')
