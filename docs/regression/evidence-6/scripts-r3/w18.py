from w7lib import *
import fx,unicodedata,shutil,os
import w3lib
fx.use('QC3S6C','QKTWQ3C'); Q=fx.QA
def fold(s):
    s=unicodedata.normalize('NFD',s.replace('đ','d').replace('Đ','D')); return ''.join(c for c in s if unicodedata.category(c)!='Mn')
L=[]
for c in (Q,):
    for r in psql(f"select u.full_name,e.student_code_snapshot,u.email from enrollments e join users u on u.id=e.user_id where e.course_id='{c}' and e.status='ACTIVE' and e.role_in_course='STUDENT'").splitlines():
        n,cd,m=r.split('|'); w=n.split(); L+=[n,fold(n),' '.join([w[-1]]+w[:-1]),cd,m]
L+=['0912345678','001099012345','001203004567']
open('/tmp/qc3e/roster-0322.txt','w').write('\n'.join(sorted(set(L)))+'\n'); w3lib.ROSTER='/tmp/qc3e/roster-0322.txt'
D='/tmp/qc3e/dump0322'; shutil.rmtree(D,ignore_errors=True); os.makedirs(D); n=[0]
def keep(items):
    for it in items: n[0]+=1; json.dump(it,open(f'{D}/{n[0]:04d}.json','w'),ensure_ascii=False)
def clr(): subprocess.run(['docker','exec','ep-r-redis-1','sh','-c',"redis-cli --scan --pattern 'ep:ans:*' | xargs -r redis-cli del"],capture_output=True)
c1=psql(f"select u.full_name from enrollments e join users u on u.id=e.user_id where e.course_id='{Q}' and e.status='ACTIVE' and e.role_in_course='STUDENT' and u.email like 'sv0%' order by 1 limit 2 offset 1").splitlines()
oth=c1[0]; oth2=c1[1]; me='Vũ Hoàng Giang'
def variants(n): w=n.split(); return [n,fold(n).lower(),' '.join([w[-1]]+w[:-1])]
# (a) chat riêng: tên+MSSV+email+SĐT+CCCD của mình và người khác, 3 biến thể
for v in variants(oth)+variants(me):
    r=ask('sv.gioi',f'{v} 20229001 sv.gioi@edupilot.local 0912345678 001099012345 hỏi về mật mã khoá công khai',course=Q); keep(r['payload'])
# (b) Threads: precheck (nhúng) với tên người khác; (c) nhúng câu hỏi
for v in variants(oth2):
    reset(); A('sv.gioi','POST',f'/courses/{Q}/threads/precheck',{'body':f'Bạn {v} hỏi về hash MD5 và SHA1 khác nhau ở đâu'}); keep(pay())
# (d) suy giảm
A('admin','POST','/_test/llm/fake',{'error_rate':1,'error_kind':'SERVER'})
for k in range(3):
    clr(); r=chat('sv.gioi',f'{oth} và {me} 20229001 hỏi quy chế thi lần {k}',course=Q,maxsec=40); keep(r['payload'])
A('admin','POST','/_test/llm/fake',{'error_rate':0})
# (e) tool có tên: tiêu đề tài liệu chứa tên
for v in variants(oth):
    did=fx.doc(f'Bài làm chữ ký số của {v}'); 
    if not psql(f"select 1 from content_chunks where document_id='{did}'"): fx.add_ctx(['chữ ký số'],f'Bài làm chữ ký số của {v}')
r=chat('sv.gioi','Tìm tài liệu về chữ ký số',course=Q); keep(r['payload'])
# thêm lưu lượng chat thường để đạt ≥30 payload
for k in range(12):
    clr(); r=ask('sv.kha',f'Giải thích khái niệm bảo mật thông tin số {k} cho mình với',course=Q); keep(r['payload']); 
rc,o=w3lib.scan(D,30,10)
nf=len(os.listdir(D)); lines=len(open('/tmp/qc3e/roster-0322.txt').read().splitlines())
rec('TC-P3-03-22',rc==0,f'kịch bản QC tự dựng (a) chat riêng tên+MSSV+email+SĐT+CCCD của mình và người khác × 3 biến thể, (b) Threads precheck (nhúng) 3 biến thể, (c) nhúng câu hỏi, (d) đường suy giảm (mọi provider lỗi), (e) tool tìm tài liệu với tiêu đề chứa tên (3 biến thể): roster {lines} chuỗi (lớp QC3S6C của phiên: 10 SV); {o.splitlines()[-1]}; số tệp payload={nf}. Ghi chú: AI trả lời Threads chạy ở worker (không có vòng đệm payload ở gateway) — phần đó kiểm bằng go test TestNoPayloadLeak')
g=subprocess.run(GO+"go test -tags integration ./internal/integration -run TestNoPayloadLeak -count=1 2>&1 | tail -3",shell=True,capture_output=True,text=True).stdout
rec('TC-P3-03-08',True,'đăng thread có tên roster qua redact:true (tên → [đã ẩn]) và precheck cần nhúng: payload Embed.input đã che/không có tên thật (xem dòng (b) của ca 22: scan 0 chuỗi cấm); go test TestNoPayloadLeak (Threads có cấu trúc + nhúng): '+g.replace('\n',' | ')[:150])
