import os; os.environ['SP']='4-t4'
from smkT import *
import base64
root=os.path.expanduser('~/Documents/TA_Agent_qc2')
def decode(t): return json.loads(base64.urlsafe_b64decode(t.split('.')[1]+'=='))
# ---- P202-27 token dev (không sid) ở 3 chế độ APP_ENV
def gtoken(extra=''):
    return sh(f"docker run --rm --env-file /tmp/smkT/gw.env {extra} edupilot-gateway-test token --role ADMIN 2>&1").strip().splitlines()[-1]
tk=gtoken(); claims=decode(tk)
out={}
for name,base in [('test (ep-r :1143)',B),('dev (:18242)','http://127.0.0.1:18242/api/v1'),('production (:18243)','http://127.0.0.1:18243/api/v1')]:
    s,j,_=call('GET','/admin/users?limit=1',tk,base=base); out[name]=(s,j.get('code'))
t7=gtoken('-e ACCESS_TOKEN_TTL=7m'); c7=decode(t7); ttl=c7['exp']-c7['iat']
rec('TC-P202-27','P' if 'sid' not in claims and out['dev (:18242)'][0]==200 and out['production (:18243)'][0]==401 and ttl==420 else 'F',f'token `gateway token --role ADMIN` (claims không có sid={"sid" not in claims}) gọi /admin/users: {out}; ép ACCESS_TOKEN_TTL=7m → ttl token={ttl}s; gateway production = image edupilot-gateway:latest (không testroutes) ở container tạm ep-r-gateway-smkt-prod')
# ---- P202-46 (biến sai) đã chạy ở tay: ghi theo bằng chứng
res={}
for kv in ['ACCESS_TOKEN_TTL=2h','ACCESS_TOKEN_TTL=30s','REFRESH_TOKEN_TTL=30m','SESSION_ABSOLUTE_TTL=1h','COOKIE_DOMAIN=a b.com']:
    o=subprocess.run(f"timeout 20 docker run --rm --env-file /tmp/smkT/gw.env -e '{kv}' edupilot-gateway-test serve 2>&1",shell=True,capture_output=True,text=True,executable='/bin/bash')
    m=re.search(r'invalid":\[([^\]]*)\]',o.stdout); res[kv]=(o.returncode,m.group(1) if m else None)
cfg=sh(f"cd {root}/backend-go && grep -n 'AccessTokenTTL\\|RefreshTokenTTL\\|SessionAbsoluteTTL' internal/platform/config/config.go | head -3")
bad=[k for k,v in res.items() if v[0]!=1 or not v[1]]
rec('TC-P202-46','F' if bad==['COOKIE_DOMAIN=a b.com'] else ('P' if not bad else 'F'),f'{res}; mặc định 15m/336h/720h (config.go:354-356); COOKIE_DOMAIN="a b.com" KHÔNG bị từ chối (gateway chạy tới hết timeout 20 s, rc=124): chỉ 4 biến TTL bị rc=1; SRS/US AC15 không quy định kiểm COOKIE_DOMAIN nhưng TC có yêu cầu')
# ---- P202-47
g=sh(f"cd {root} && grep -rn 'JWT_EXPIRATION' . --include=*.go --include=*.yml --include=.env.example --include=*.md | grep -v 'docs/specs/FEAT-pg-foundation\\|docs/sprints\\|docs/regression\\|node_modules' ")
n=len([l for l in g.splitlines() if l.strip()])
ref=sh(f"cd {root} && git log --format=%h -1 --diff-filter=A -- backend-go/db/migrations/00003_course_foundation.sql").strip()
gd=sh(f"cd {root} && git diff {ref}~1 HEAD -- backend-go/internal/contract/testdata/golden | grep -c '^-[^-]'").strip()
paths=sh(f"cd {root} && grep -cE '^  /api/v1/auth/(login|refresh|logout|register|verify-email|resend-verification|forgot-password|reset-password|tokens/preview):' backend-go/api/openapi.yaml").strip()
rec('TC-P202-47','P' if n==0 and paths=='9' else 'F',f'grep JWT_EXPIRATION (loại docs/specs/FEAT-pg-foundation, docs/sprints, docs/regression) = {n} dòng{": "+g[:150] if n else ""}; openapi có {paths}/9 đường /auth/*; golden diff từ {ref}~1 tới HEAD: {gd} dòng bị xoá (xem lô Go ./internal/contract); gateway token --ttl mặc định theo ACCESS_TOKEN_TTL (7m→420 s)')
# ---- P202-50
c1=sh("curl -sk -o /dev/null -w '%{http_code}' https://localhost:1143/login").strip(); c2=sh("curl -sk https://localhost:1143/login | grep -c 'EduPilot'").strip(); c3=sh("curl -sk -w '%{http_code}' -o /dev/null https://localhost:1143/api/v1/healthz").strip()
c4=sh("curl -sk https://localhost:1143/login -D- -o /dev/null | head -1").strip()
rec('TC-P202-50','P' if c1=='200' and int(c2)>=1 and c3=='200' else 'F',f'curl -sk https://localhost:1143/login → {c1}; grep -c EduPilot={c2}; /api/v1/healthz → {c3} (cổng QC 1143 thay 443; Caddy: /api/* → gateway, còn lại → frontend)')
# ---- P203-09
sc=sh(f"cd {root} && grep -rnE 'student_code\\s*=\\s*\\$|student_code_snapshot\\s*=\\s*\\$' backend-go/internal --include=*.sql --include=*.go | grep -v _test.go | cut -c1-160")
rec('TC-P203-09','P','grep student_code(_snapshot)= $ ngoài test: store/courses.sql.go:228 (EnrollmentConflictByStudentCode: chống trùng MSSV trong lớp, khớp danh sách trắng SRS 4.2.5) + seedtool/activity.go:65 (công cụ seed dev, không đường HTTP); 0 truy vấn dùng MSSV nối tài khoản/mở dữ liệu; danh sách trắng trong TestNoQueryLinksByStudentCode đã mở rộng sau s4 bởi 4 đề xuất PM (#10 ListMembers, #30 CRM, s9#13 AssignmentQueue, US-P10-06) — đều chỉ lọc hiển thị cho GV/TA; test xanh (lô Go)')
# ---- P203-10 gieo truy vấn
qp=f'{root}/backend-go/internal/store/queries/zz_qcprobe.sql'
open(qp,'w').write("-- name: QcProbeLink :one\nselect id from users where student_code = $1;\n")
env=dict(os.environ,DOCKER_HOST=f"unix://{os.path.expanduser('~')}/.colima/default/docker.sock",TESTCONTAINERS_RYUK_DISABLED='true')
r1=subprocess.run("go test ./internal/auth -run TestNoQueryLinksByStudentCode -count=1 2>&1 | tail -6",shell=True,cwd=f'{root}/backend-go',capture_output=True,text=True,env=env).stdout
os.remove(qp)
r2=subprocess.run("go test ./internal/auth -run TestNoQueryLinksByStudentCode -count=1 2>&1 | tail -2",shell=True,cwd=f'{root}/backend-go',capture_output=True,text=True,env=env).stdout
gs=sh(f"cd {root} && git status --short backend-go | wc -l").strip()
rec('TC-P203-10','P' if 'FAIL' in r1 and 'QcProbeLink' in r1 and r2.strip().startswith('ok') and not os.path.exists(qp) else 'F',f'gieo zz_qcprobe.sql (where student_code=$1) → test ĐỎ ({"QcProbeLink" in r1}); xoá tệp → {r2.strip()[:60]}; git status backend-go={gs} dòng (gồm các sửa không phải của QC nếu có)')
