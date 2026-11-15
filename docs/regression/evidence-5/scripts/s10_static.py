import os; os.environ['SUITE']='static'
from r5lib import *
R='/Users/kuro/Documents/TA_Agent_qc2'
def sh(c,t=120): return subprocess.run(c,shell=True,capture_output=True,text=True,cwd=R,timeout=t).stdout.strip()
# PE01-04
mig = sh("git log --format='%h %ad %s' --date=short -- backend-go/db/migrations/00001_*.sql backend-go/db/migrations/00002_*.sql backend-go/db/migrations/00003_*.sql backend-go/db/migrations/00004_*.sql | head -5")
alt = sh("grep -ciE 'alter table (users|courses|enrollments|auth_|mail_|llm_|jobs|outbox|audit_log)' backend-go/db/migrations/00006_weekly_exam.sql")
rec('TC-PE01-04', 'LT', 'migration PE nay là 00006 (00005 là vn_fold của P2); 00001–00004 đã bị sửa sau sprint 4: ' + mig.replace(chr(10), ' ; ')[:200] + '; ALTER bảng cũ trong 00006: ' + alt)
# PE01-31 : đếm truy vấn
db("alter system set log_statement='all'"); db("select pg_reload_conf()")
def cnt(lim):
    t0 = subprocess.run(['docker','logs','--since','1s','ep-r-postgres-1'],capture_output=True,text=True)
    mark = time.strftime('%Y-%m-%dT%H:%M:%S', time.gmtime(time.time()-1)); call('GET', Q + f'?limit={lim}', T); time.sleep(1.5)
    lg = subprocess.run(['docker','logs','--since',mark+'Z','ep-r-postgres-1'],capture_output=True,text=True); txt = lg.stdout + lg.stderr
    return len([l for l in txt.splitlines() if 'statement:' in l or 'execute ' in l and 'question_bank' in l or 'FROM question_bank' in l.upper() or 'from question_bank' in l])
a, b = cnt(5), cnt(100); db("alter system reset log_statement"); db("select pg_reload_conf()")
off = sh("grep -rn 'OFFSET' backend-go/internal/exam backend-go/internal/store/queries/question*.sql backend-go/internal/store/queries/exam*.sql | grep -v _test | grep -iv '^.*--' | wc -l")
chk('TC-PE01-31', a == b and a > 0 and off == '0', f'số câu SQL liên quan question_bank: limit=5 → {a}, limit=100 → {b}; OFFSET trong mã/SQL PE: {off}')
# PE02-12
g = sh("grep -rnE 'os/exec|syscall\\.Exec|exec\\.Command' backend-go/internal/judge backend-go/internal/exam | grep -v _test.go | wc -l")
chk('TC-PE02-12', g == '0', f'os/exec trong internal/judge + internal/exam (không test): {g}')
# PE02-56
ci = sh("grep -n 'judge-attacks' .github/workflows/ci.yml | head -2; grep -c 'no-seccomp' .github/workflows/ci.yml docker-compose.test.yml")
rec('TC-PE02-56', ('judge-attacks' in ci and 'ubuntu-24.04' in sh("sed -n 178,185p .github/workflows/ci.yml") and sh("grep -c 'no-seccomp' .github/workflows/ci.yml") == '0'), f'job judge-attacks (ubuntu-24.04 amd64) có; -no-seccomp ở ci.yml: {sh("grep -c no-seccomp .github/workflows/ci.yml")}; kết luận CI GitHub (gh) không có trong repo clone — chỉ đọc cấu hình')
# PE09-24, 30, 29, 28
rec('TC-PE09-24', int(sh("grep -c 'GATE PE' docs/sprints/5/handoff/dev-US-PE-09.md") or 0) >= 1, 'GATE PE trong handoff: ' + sh("grep -c 'GATE PE' docs/sprints/5/handoff/dev-US-PE-09.md"))
d30 = sh("grep -c 'weekly_exam' docs/PROGRESS.md"); rep = os.path.exists(R + '/docs/sprints/5/report.md'); debts = [k for k in ('sandbox tách máy', 'precompiled', 'checker tuỳ chỉnh', 'xoá `RUN`', 'sổ điểm', 'chat') if k.lower() in open(R + '/docs/sprints/5/report.md', encoding='utf-8').read().lower()]
rec('TC-PE09-30', int(d30) >= 1 and rep and len(debts) >= 5, f'weekly_exam trong PROGRESS.md: {d30}; report.md có; nợ nêu: {debts}')
ui = sh("bash scripts/ui-antipatterns.sh 2>&1 | tail -6"); st = sh("bash scripts/ui-antipatterns.sh --selftest 2>&1 | tail -3")
rec('TC-PE09-29', 'rc' not in ui and ('✓' in ui or 'OK' in ui), f'ui-antipatterns: {ui[-160:]!r}; selftest: {st[-100:]!r}')
tech = sh("grep -rnE 'sandbox|verdict|testcase|token|reasoning|RAG|PII' frontend/src/features/exam --include=*.tsx | grep -v 'className\\|//' | grep -E '>[^<]*(sandbox|verdict|RAG|PII)[^<]*<|\"[^\"]*(Sandbox|sandbox)[^\"]*\"' | head -5")
rec('TC-PE09-28', tech == '', f'từ kỹ thuật trong chữ hiển thị feature exam: {tech[:160] or "không thấy"}')
print(sh("ls frontend/src/features | head -30"))
