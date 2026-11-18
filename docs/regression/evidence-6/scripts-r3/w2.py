from h import *
import subprocess,unicodedata,threading
SC='/Users/kuro/Documents/TA_Agent_qc3p/docs/sprints/6/qc/scripts/p603-payload-scan.sh'
def fold(s): 
    s=unicodedata.normalize('NFD',s.replace('đ','d').replace('Đ','D')); return ''.join(c for c in s if unicodedata.category(c)!='Mn')
def roster_lines(course):
    rows=psql(f"select u.full_name,e.student_code_snapshot,u.email from enrollments e join users u on u.id=e.user_id where e.course_id='{course}' and e.status='ACTIVE' and e.role_in_course='STUDENT'").splitlines()
    L=[]
    for r in rows:
        n,c,m=r.split('|'); w=n.split(); L+= [n,fold(n),' '.join([w[-1]]+w[:-1]),c,m]
    return L
ROSTER='/tmp/qc3e/roster.txt'; open(ROSTER,'w').write('\n'.join(roster_lines(C1)+['0912345678','001099012345'])+'\n')
def scan(dirp,minp=0,minph=0):
    r=subprocess.run(['bash',SC,'--dump',dirp,'--roster',ROSTER,'--min-payloads',str(minp),'--min-placeholders',str(minph)],capture_output=True,text=True); return r.returncode,r.stdout.strip()
def casedir(name,items):
    d=f'/tmp/qc3e/cd-{name}'; shutil.rmtree(d,ignore_errors=True); os.makedirs(d)
    for k,it in enumerate(items): json.dump(it,open(f'{d}/{k}.json','w'),ensure_ascii=False)
    return d
GIANG='Vũ Hoàng Giang'; KHAI='Bùi Thanh Khải'
nlines=len(open(ROSTER).read().splitlines())
# ---------- 02-34 (agent path, system prompt có 'bạn', văn bản hiện tên thật)
r=chat('sv.gioi',f'Em là {GIANG}, MSSV 20229001, cho em xem điểm danh'); p=r['payload']
sysmsg=p[0]['messages'][0] if p else ''
rec('TC-P3-02-34',bool(p) and GIANG not in flat(p) and '20229001' not in flat(p) and 'bạn' in sysmsg and GIANG in r['text'],f'payload {len(p)} lời gọi: 0 tên/MSSV; system có "bạn": {"bạn" in sysmsg}; câu trả lời hiện: {r["text"][:110]!r}; notice={r["notice"]}')
# ---------- 03-05 / 03-06 / 03-18
txt=f'{KHAI} 20229002 bui.khai@edupilot.local 0912345678 001203004567 học nhóm với em'
r1=chat('sv.gioi',txt); d1=casedir('0305',r1['payload']); rc,o=scan(d1)
ph=[x for x in ['[[SV_1]]','[[MSSV_1]]','[[EMAIL_1]]','[[SDT_1]]','[[CCCD_1]]'] if x in flat(r1['payload'])]
rec('TC-P3-03-05',rc==0 and len(ph)==5,f'scan roster: {o.splitlines()[-1]}; placeholder: {ph}; payload={ [m for it in r1["payload"] for m in it["messages"][1:]][:1] }')
r2=chat('sv.gioi','Ý trên giải thích thêm giúp em',sid=r1['sid']); p2=r2['payload']; rc,o=scan(casedir('0306',p2))
hist=' '.join(m for it in p2 for m in it['messages'])
rec('TC-P3-03-06',rc==0 and '[[SV_1]]' in hist,f'scan: {o.splitlines()[-1]}; lịch sử cũ trong payload lượt hai chứa [[SV_1]]={"[[SV_1]]" in hist}; số message={[len(it["messages"]) for it in p2]}')
txt6=f'Em là {GIANG}, bạn {KHAI} cùng nhóm, MSSV 20229001 và 20229002, mail sv.gioi@edupilot.local, SĐT 0912345678, CCCD 001099012345, hỏi quy chế thi'
r3=chat('sv.gioi',txt6)
pm=psql(f"select masked_count from chat_messages where session_id='{r3['sid']}' and role='ASSISTANT'"); au=psql(f"select pii_masked_count,status from llm_audit where user_id=(select id from users where email='sv.gioi@edupilot.local') and task='CHAT' order by created_at desc limit 1")
nt=r3['notice'][0].get('masked') if r3['notice'] else None
rec('TC-P3-03-18',nt==pm.strip() or str(nt)==pm.strip(),f'tin có 2 tên + 2 MSSV + email + SĐT + CCCD: notice.masked={nt}; chat_messages.masked_count={pm}; llm_audit.pii_masked_count|status={au}; intent={r3["row"]}')
# ---------- 02-31 phần notice
# ---------- 03-34: một tin COURSE_QA → 1 sinh chữ + ≤1 embed
r=chat('sv.kha','Quy chế thi cuối kỳ nói gì về tài liệu được mang vào phòng thi?'); p=r['payload']
gen=[i for i in p if i['task']!='EMBEDDING']; emb=[i for i in p if i['task']=='EMBEDDING']
GO=f"cd /Users/kuro/Documents/TA_Agent_qc3p/backend-go && DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true "
g=subprocess.run(GO+"go test ./internal/agent ./internal/chat -run TestOneGenerationPerMessage -v -count=1 2>&1 | tail -4",shell=True,capture_output=True,text=True).stdout
rec('TC-P3-03-34',len(gen)==1 and len(emb)<=1 and 'ok' in g,f'một tin COURSE_QA: {len(gen)} lời gọi sinh chữ, {len(emb)} lời gọi nhúng; go test TestOneGenerationPerMessage: {g.strip().splitlines()[-1][:70]}')
# ---------- 03-35 chéo lớp: tên chỉ ở C1 trong phiên C2
only1=psql(f"select u.full_name from enrollments e join users u on u.id=e.user_id where e.course_id='{C1}' and e.status='ACTIVE' and e.role_in_course='STUDENT' and e.user_id not in (select user_id from enrollments where course_id='{C2}' and status='ACTIVE') limit 1")
sg=psql(f"select count(*) from enrollments e join users u on u.id=e.user_id where e.course_id='{C2}' and u.email='sv.gioi@edupilot.local' and e.status='ACTIVE'")
sid2=newsess('sv.gioi',C2) if sg=='1' else None
if sid2:
    r=chat('sv.gioi',f'Thảo luận về bài {only1} làm tuần trước',sid=sid2,course=C2); m=flat(r['payload'])
    nt=r['notice'][0].get('masked') if r['notice'] else 0
    rec('TC-P3-03-35',only1 in m and (nt in(0,None)),f'phiên C2: tên "{only1}" (chỉ roster C1) không bị che trong payload={only1 in m}, notice.masked={nt}; roster lấy theo CourseID của phiên')
else: rec('TC-P3-03-35',False,'sv.gioi không ở C2')
# ---------- 03-23 scanner phân biệt
d=f'/tmp/qc3e/decoy'; shutil.rmtree(d,ignore_errors=True); os.makedirs(d)
json.dump({'task':'CHAT','messages':['xin chào Bùi Thanh Khải']},open(d+'/a.json','w'),ensure_ascii=False); json.dump({'task':'CHAT','messages':['mã 20229002 ở đây']},open(d+'/b.json','w'))
rc,o=scan(d); rec('TC-P3-03-23',rc==1 and 'Bùi Thanh Khải' in o.replace('Bùi','Bùi') and '20229002' in o,f'thư mục mồi 2 tệp: rc={rc}; '+' | '.join(o.splitlines()[-6:]))
rec('TC-P3-03-24',nlines>=30*3+30 and all(len(x)>=4 for x in [KHAI]),f'roster {nlines} dòng (30 SV × 3 biến thể tên + 30 MSSV + 30 email + 2 canary SĐT/CCCD = 182); quét dùng grep -ioFf (không phân biệt hoa-thường); 0 trùng trên toàn bộ dump {len(os.listdir(DUMP))} payload: '+scan(DUMP)[1].splitlines()[-1])
# ---------- 03-27 providers/test không có nội dung người dùng
reset(); fp='01a1df2d-a8ed-7389-b1d8-d6805e510bed'; s,d=A('admin','POST',f'/admin/llm/providers/{fp}/test'); its=pay()
rec('TC-P3-03-27',s in(200,204) and all(len(' '.join(i['messages']))<200 for i in its) and not re.search(r'Khải|Giang|2022',flat(its)),f'POST providers/{{id}}/test → {s}; payload={[(i["task"],i["messages"]) for i in its]}')
json.dump(R,open('/tmp/qc3e/w2.json','w'),ensure_ascii=False)
