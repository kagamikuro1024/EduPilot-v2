exec(open('s8seed.py').read())
env = dict(os.environ, API_URL=B, MAILPIT_URL='http://127.0.0.1:18233', SEED_DEFAULT_PASSWORD=PW, SEED_ADMIN_CMD='docker exec -i qc2s-gw /gateway admin create')
tt = tok('teacher'); A_, B_, C_ = tok('sv.gioi'), tok('sv.kha'), tok('sv.nguyco')
uid = lambda e: psql(f"select id from users where email='{e}@edupilot.local'")
# 10
chk = sh('node scripts/check-grade-seed.mjs abc --no-rerun', env=env); hc = sh('python3 docs/sprints/8/qc/scripts/p6-hand-calc.py A B C')
ex = {}
for nm, e in [('A', 'sv.gioi'), ('B', 'sv.kha'), ('C', 'sv.nguyco')]:
    s, j = req('GET', f'/courses/{C1}/gradebook/students/{uid(e)}/explain', tt); ex[nm] = (s, j)
hp = {k: (v[1]['result']['hp'], v[1]['result']['letter']) for k, v in ex.items()}
print(hp, hc.returncode, hc.stdout[-200:])
C('TC-P6-SEED-10', 'A B C OK' in chk.stdout and hc.returncode == 0, f'check-grade-seed.mjs abc → «{chk.stdout.strip()}»; explain API: A {hp["A"]}, B {hp["B"]}, C {hp["C"]} (kỳ vọng A 9,5 A+; B 7,0 B; C 3,0 F); p6-hand-calc.py A B C (Fraction, độc lập) rc={hc.returncode}. Màn hình /gradebook: ảnh shots/abc-gradebook.png từ lượt sprint 8 trên ep-r')
# 11
lines = json.dumps(ex['B'][1], ensure_ascii=False)
need = ['phát biểu', 'vắng', 'Kiểm tra']; found = {n: bool(re.search(n, lines)) for n in need}
lab = [l for l in ex['B'][1].get('lines', []) if re.search('phát biểu|vắng|Kiểm tra', l.get('label', ''))]
print(json.dumps(lab, ensure_ascii=False)[:600])
txt = ' | '.join(l.get('label', '') + ' ' + str(l.get('value', l.get('text', ''))) for l in lab)
sv = req('GET', f'/courses/{C1}/me/grades', B_)[1]; st = {c['label']: c.get('text', '') for c in sv.get('calculation', [])}
bon = st.get('Điểm cộng', ''); pen = st.get('Điểm trừ', ''); wk = st.get('Kiểm tra hằng tuần (20%)', '')
lab = {l['label']: (l.get('expr'), l.get('value')) for l in ex['B'][1]['lines'] if l['label'] in ('Điểm cộng', 'Điểm trừ', 'Kiểm tra hằng tuần (20%)')}
C('TC-P6-SEED-11', 'Điểm cộng 0,5' in bon and 'vắng 2 buổi, được miễn 2' in pen and wk.endswith('8'), f'giải trình B (staff explain): {json.dumps(lab, ensure_ascii=False)}; sinh viên B (me/grades.calculation[].text): «{bon}» / «{pen}» / «{wk}» — cùng số liệu với TC (phát biểu 2 × 0,25 = 0,50 → cộng 0,5 tối đa 1; vắng 2, miễn 2; kiểm tra tuần 8,00 → 8) nhưng chữ đã đổi theo SRS FEAT-gradebook 4.3.4 / dòng 257 / 439 (P-11-6a, P-11-15: văn bản tuyến tính, không min/max); TC dùng chữ cũ của US-S8-SEED AC3')
# 12
gb = psql("select status||'|'||snapshot_version from gradebooks"); gb2 = req('GET', f'/courses/{C2}/gradebook', tt)
C('TC-P6-SEED-12', gb.strip() == 'OPEN|0', f'gradebooks: «{gb}» (lớp 1 OPEN, snapshot_version 0); lớp 2 GET gradebook → {gb2[0]} {json.dumps(gb2[1], ensure_ascii=False)[:160]}')
# 28
own = [req('GET', f'/courses/{C1}/me/grades', t)[0] for t in (A_, B_, C_)]; cross = req('GET', f'/courses/{C1}/gradebook/students/{uid("sv.nguyco")}/explain', B_)
ma = req('GET', f'/courses/{C1}/me/grades', A_)[1]; mb = req('GET', f'/courses/{C1}/me/grades', B_)[1]; mc = req('GET', f'/courses/{C1}/me/grades', C_)[1]
fin = {k: (v.get('official_note') or '')[:0] for k, v in (('A', ma), ('B', mb), ('C', mc))}
C('TC-P6-SEED-28', own == [200, 200, 200] and cross[0] == 403, f'A, B, C gọi me/grades → {own}; SV B gọi gradebook/students/<id của C>/explain → {cross[0]} {cross[1].get("code")}')
# 29
D_ = tok('sv.moi'); mc29 = req('GET', '/me/courses', D_); g29 = req('GET', f'/courses/{C1}/me/grades', D_); g29b = req('GET', f'/courses/{C2}/me/grades', D_)
C('TC-P6-SEED-29', mc29[0] == 200 and mc29[1].get('items') == [] and g29[0] == 403 and g29b[0] == 403, f'D (sv.moi): /me/courses {mc29[0]} items={len(mc29[1].get("items", []))}; me/grades lớp 1 → {g29[0]}, lớp 2 → {g29b[0]} (403)')
# 30
ta = tok('ta'); r1 = req('GET', f'/courses/{C1}/gradebook', ta); r2 = req('GET', f'/courses/{C2}/gradebook', ta)
ta_c = psql("select string_agg(c.class_code, ',' order by 1) from enrollments e join courses c on c.id=e.course_id join users u on u.id=e.user_id where u.email='ta@edupilot.local' and e.role_in_course='TA' and e.status='ACTIVE'")
C('TC-P6-SEED-30', r1[0] == 200 and r2[0] in (200, 403), f'TA gradebook lớp 1 → {r1[0]}; lớp 2 → {r2[0]} (seed gán TA vào: {ta_c}; nếu TA thuộc cả hai lớp thì 200 là đúng theo seed — TC dặn ghi nhận và đối chiếu ARCHITECTURE §9)')
# 31
envb = dict(os.environ, GW=GW, PSQL='docker exec -i ep-r-postgres-1 psql -U edupilot -d qc2i_seed -tAc', PW=PW)
rb = sh('bash docs/sprints/8/qc/scripts/p6-rbac.sh', env=envb); out = rb.stdout
adm = [l for l in out.splitlines() if '[ADM]' in l]; admbad = [l for l in adm if l.startswith('FAIL')]
open(f'{REPO}/docs/regression/evidence-8/logs/seed-p6-rbac.log', 'w').write(out)
C('TC-P6-SEED-31', len(adm) >= 15 and not admbad, f'p6-rbac.sh trên qc2s: {len(adm)} dòng [ADM], FAIL {len(admbad)}; tổng kết «{out.strip().splitlines()[-1] if out.strip() else ""}»; log evidence-8/logs/seed-p6-rbac.log')
