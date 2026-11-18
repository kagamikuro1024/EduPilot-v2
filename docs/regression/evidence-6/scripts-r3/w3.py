from fx import *
import unicodedata
SC='/Users/kuro/Documents/TA_Agent_qc3p/docs/sprints/6/qc/scripts/p603-payload-scan.sh'
def fold(s):
    s=unicodedata.normalize('NFD',s.replace('đ','d').replace('Đ','D')); return ''.join(c for c in s if unicodedata.category(c)!='Mn')
def roster_lines(course):
    L=[]
    for r in psql(f"select u.full_name,e.student_code_snapshot,u.email from enrollments e join users u on u.id=e.user_id where e.course_id='{course}' and e.status='ACTIVE' and e.role_in_course='STUDENT'").splitlines():
        n,c,m=r.split('|'); w=n.split(); L+=[n,fold(n),' '.join([w[-1]]+w[:-1]),c,m]
    return L
ROSTER='/tmp/qc3e/roster-qa.txt'; open(ROSTER,'w').write('\n'.join(roster_lines(QA)+['0912345678','001099012345','001203004567'])+'\n')
def casedir(name,items):
    d=f'/tmp/qc3e/cd-{name}'; shutil.rmtree(d,ignore_errors=True); os.makedirs(d)
    for k,it in enumerate(items): json.dump(it,open(f'{d}/{k}.json','w'),ensure_ascii=False)
    return d
def scan(dirp,minp=0,minph=0):
    r=subprocess.run(['bash',SC,'--dump',dirp,'--roster',ROSTER,'--min-payloads',str(minp),'--min-placeholders',str(minph)],capture_output=True,text=True); return r.returncode,r.stdout.strip()
GIANG='Vũ Hoàng Giang'; KHAI='Bùi Thanh Khải'
BE='/Users/kuro/Documents/TA_Agent_qc3p/backend-go'
GO=f"cd {BE} && DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true "
def nogen(p): return [i for i in p if i['task']!='EMBEDDING']
# ---- 02-34: tên trong tiêu đề tài liệu ngữ cảnh → hiện tên thật sau khôi phục
r=ask('sv.gioi',f'Em là {GIANG}, MSSV 20229001, cho em xem điểm danh'); p=r['payload']; sysm=nogen(p)[0]['messages'][0] if nogen(p) else ''
r2=ask('sv.gioi',f'Em là {GIANG}, MSSV 20229001, giải thích giúp em về chữ ký số',title=f'Bài ghi chú của {GIANG}'); p2=r2['payload']
ok=bool(nogen(p)) and GIANG not in flat(p) and '20229001' not in flat(p) and 'bạn' in sysm and GIANG in r2['text'] and GIANG not in flat(p2) and '20229001' not in flat(p2)
rec('TC-P3-02-34',ok,f'câu "xem điểm danh": payload 0 tên/MSSV, system có "bạn"; câu hỏi học thuật có tên trong ngữ cảnh: payload 0 tên thật, câu trả lời hiện: {r2["text"][:70]!r}')
# ---- 03-05 lại với ngữ cảnh (đã PASS ở lượt trước; giữ). 03-06 lịch sử
txt=f'{KHAI} 20229002 bui.khai@edupilot.local 0912345678 001203004567 học nhóm với em, giải thích chữ ký số'
r1=ask('sv.gioi',txt); sid=r1['sid']
r2=ask('sv.gioi','Ý trên giải thích thêm giúp em',sid=sid); p2=nogen(r2['payload'])
hist=' '.join(m for it in p2 for m in it['messages']); rc,o=scan(casedir('0306',p2))
rec('TC-P3-03-06',rc==0 and '[[SV_1]]' in hist and len(p2)==1,f'lượt hai: {len(p2)} payload sinh chữ, {sum(len(i["messages"]) for i in p2)} message; lịch sử cũ đã che: [[SV_1]] có={"[[SV_1]]" in hist}; scan roster: {o.splitlines()[-1]}')
# ---- 03-18: đúng 6 thực thể bị thay
t18=f'Em là {GIANG}, bạn {KHAI} cùng nhóm, MSSV 20229001, email sv.gioi@edupilot.local, SĐT 0912345678, CCCD 001099012345, giải thích về hàm băm'
r=ask('sv.gioi',t18); nt=r['notice'][0].get('masked') if r['notice'] else None
pm=psql(f"select masked_count from chat_messages where session_id='{r['sid']}' and role='ASSISTANT' order by created_at desc limit 1")
au=psql(f"select pii_masked_count||'|'||status from llm_audit where user_id=(select id from users where email='sv.gioi@edupilot.local') and task='CHAT' order by created_at desc limit 1")
rec('TC-P3-03-18',nt==6 and pm==str(6) and au.startswith('6|'),f'tin có 2 tên + MSSV + email + SĐT + CCCD (6 thực thể): notice.masked={nt}, chat_messages.masked_count={pm}, llm_audit.pii_masked_count|status={au}')
# ---- 03-34
r=ask('sv.kha','Quy chế thi cuối kỳ nói gì về tài liệu được mang vào phòng thi?'); p=r['payload']
g=subprocess.run(GO+"go test ./internal/agent ./internal/chat -run TestOneGenerationPerMessage -v -count=1 2>&1 | tail -4",shell=True,capture_output=True,text=True).stdout
rec('TC-P3-03-34',len(nogen(p))==1 and len([i for i in p if i['task']=='EMBEDDING'])<=1 and 'ok' in g,f'một tin COURSE_QA (có ngữ cảnh): {len(nogen(p))} lời gọi sinh chữ, {len([i for i in p if i["task"]=="EMBEDDING"])} lời gọi nhúng; go test TestOneGenerationPerMessage: {g.strip().splitlines()[-1][:60]}')
# ---- 03-07 tool search_library trả tiêu đề có tên
dt=f'Bài làm chữ ký số của {KHAI}'; q='Tìm tài liệu về chữ ký số'
did=doc(dt); 
if not psql(f"select 1 from content_chunks where document_id='{did}'"): add_ctx(['chữ ký số'],dt)
r=chat('sv.gioi',q,course=QA); p=nogen(r['payload']); rc,o=scan(casedir('0307',r['payload']))
rec('TC-P3-03-07',bool(p) and rc==0 and KHAI not in flat(r['payload']) and r['row'].startswith('LIBRARY_SEARCH'),f'intent={r["row"]}; {len(p)} lời gọi sinh chữ; kết quả tool (tiêu đề chứa "{KHAI}") trong <ngữ_cảnh>: tên bị che={"[[SV_" in flat(p)}; scan roster: {o.splitlines()[-1]}; câu trả lời: {r["text"][:90]!r}')
json.dump(R,open('/tmp/qc3e/w3.json','w'),ensure_ascii=False)
