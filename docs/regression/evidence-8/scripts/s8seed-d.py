exec(open('s8seed.py').read())
env = dict(os.environ, API_URL=B, MAILPIT_URL='http://127.0.0.1:18233', SEED_DEFAULT_PASSWORD=PW, SEED_ADMIN_CMD='docker exec -i qc2s-gw /gateway admin create')
Q5 = "select (select count(*) from grade_schemes),(select count(*) from grade_components),(select count(*) from grade_items),(select count(*) from grade_entries),(select count(*) from documents)"
a = psql(Q5); t0 = time.time(); r = sh('node scripts/seed.mjs', env=env); b = psql(Q5); dt = time.time() - t0
ck = sh('node scripts/check-grade-seed.mjs counts --no-rerun', env=env)
pol = psql(f"select count(*) from documents where course_id='{C1}' and filename='quy-che-lop1.pdf'")
err409 = '409' in (r.stdout + r.stderr)
print(a, b, r.returncode, pol, ck.stdout[-120:])
C('TC-P6-SEED-18', a == b and r.returncode == 0 and pol == '1' and not err409 and ck.returncode == 0, f'(grade_schemes, grade_components, grade_items, grade_entries, documents) trước {a} → sau seed lần hai (rc={r.returncode}, {dt:.0f} s) {b}; tài liệu quy-che-lop1.pdf của lớp 1 = {pol} (không thêm bản); không lỗi 409 làm dừng; check-grade-seed.mjs counts (--no-rerun) → «{ck.stdout.strip()[-60:]}» rc={ck.returncode}')
# 22 check scripts
res = {}
for sc in ['check-crm-seed.mjs', 'check-chat-seed.mjs', 'check-exam-seed.mjs', 'check-docs-seed.mjs']:
    x = sh(f'node scripts/{sc} --no-rerun', env=env); res[sc] = (x.returncode, (x.stdout + x.stderr).strip().splitlines()[-1][:120] if (x.stdout + x.stderr).strip() else '')
print(res)
open(f'{REPO}/docs/regression/evidence-8/logs/seed-checks.json', 'w').write(json.dumps(res, ensure_ascii=False, indent=1))
