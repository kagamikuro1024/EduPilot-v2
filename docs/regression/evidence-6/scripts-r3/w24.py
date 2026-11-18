from lib import *
import subprocess
needles=['Bùi Thanh Khải','Bui Thanh Khai','20229002','sv.kha@edupilot.local','0912345678','001099012345']
tabs=['forum_threads','forum_posts','pii_events','outbox','jobs','audit_log','llm_audit']
res={}
for t in tabs:
    for n in needles:
        c=psql(f"select count(*) from {t} where {t}::text ilike '%{n.replace(chr(39),chr(39)*2)}%'")
        if c!='0': res[(t,n)]=c
logs={}
for c in ('ep-r-gateway-1','ep-r-worker-1'):
    o=subprocess.run(['docker','logs',c],capture_output=True,text=True,errors='replace'); L=o.stdout+o.stderr
    for n in needles: 
        k=L.lower().count(n.lower())
        if k: logs[(c,n)]=k
print(res,logs)
g=subprocess.run("cd /Users/kuro/Documents/TA_Agent_qc3p/backend-go && DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true go test -tags integration ./internal/thread -run TestRawPIINeverStored -v -count=1 2>&1 | grep -E '^(--- |ok|FAIL)'",shell=True,capture_output=True,text=True).stdout
rec('TC-P3-06-14',not res and not logs and 'FAIL' not in g,f'quét Python toàn dòng (row::text ILIKE) 7 bảng × 6 chuỗi canary (tên có/không dấu, MSSV, email, SĐT, CCCD) + toàn log gateway/worker: bảng khớp={res or 0}; log khớp={logs or 0}; (script scan-pii-leak.sh cũ lỗi tách "::" ở 5 bảng nên QC quét lại bằng SQL); go test TestRawPIINeverStored: {g.replace(chr(10)," | ")[:120]}')
