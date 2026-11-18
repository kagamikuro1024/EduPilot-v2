from w3lib import *
import fx,importlib.util,subprocess,re
spec=importlib.util.spec_from_file_location('p604','/Users/kuro/Documents/TA_Agent_qc3p/docs/sprints/6/qc/scripts/p604-onbehalf.py'); P4=importlib.util.module_from_spec(spec); spec.loader.exec_module(P4)
def clr(): subprocess.run(['docker','exec','ep-r-redis-1','sh','-c',"for p in 'ep:emb:*' 'ep:ans:*'; do redis-cli --scan --pattern $p | xargs -r redis-cli del; done"],capture_output=True)
def keys(p): return set(rds('--scan','--pattern',p).split())
def intent(r): return r['row'].split('|')[0]
def ngen(r): return len(nogen(r['payload']))
def nemb(r): return len([i for i in r['payload'] if i['task']=='EMBEDDING'])
USERS=['sv.gioi','sv.kha','sv04']
# ===== 04-01: 1 Embed + 1 khoá ep:emb
clr(); k0=keys('ep:emb:*'); r=chat('sv.gioi','Giải thích giúp mình cách hoạt động của thuật toán trao đổi khoá Diffie Hellman'); k1=keys('ep:emb:*')
rec('TC-P3-04-01',nemb(r)==1 and len(k1-k0)==1,f'tin mơ hồ ≥20 ký tự, không PII: {nemb(r)} lời gọi Embed, khoá ep:emb mới={len(k1-k0)}; intent={intent(r)}')
# ===== 04-04: precheck không gọi sinh chữ
clr(); reset(); s,d=A('sv.gioi','POST',f'/courses/{C1}/threads/precheck',{'body':'Cho mình hỏi cách một hệ thống mật mã khoá công khai hoạt động ra sao'}); its=pay()
rec('TC-P3-04-04',s==200 and len(nogen(its))==0,f'precheck câu mơ hồ: HTTP {s}, lời gọi Chat/Stream={len(nogen(its))}, Embed={len([i for i in its if i["task"]=="EMBEDDING"])}')
# ===== 04-05: TTL và cache nhúng
clr(); q='Phân biệt mã hoá đối xứng với mã hoá bất đối xứng giúp mình với'
r1=chat('sv.gioi',q); ks=keys('ep:emb:*'); key=sorted(ks)[0] if ks else None
ttl=int(rds('ttl',key)) if key else -9; val=rds('get',key) if key else ''
r2=chat('sv.kha',q)
rec('TC-P3-04-05',key is not None and 0<ttl<=600 and nemb(r2)==0 and q not in val,f'ep:emb TTL={ttl}s (≤600); lượt hai (phiên khác, SV khác) Embed mới={nemb(r2)}; giá trị khoá chứa chữ câu hỏi: {q in val}')
# ===== 04-06: provider nhúng lỗi → hạ cấp
A('admin','POST','/_test/llm/fake',{'error_rate':1,'error_kind':'SERVER'}); clr()
s,d=A('sv.gioi','POST',f'/courses/{C1}/threads/precheck',{'body':'Mình muốn biết sự khác nhau giữa chữ ký số và chứng thư số là gì nhỉ'})
lg=subprocess.run(['docker','logs','ep-r-gateway-1','--since','40s'],capture_output=True,text=True); lg=lg.stdout+lg.stderr
warns=[l for l in lg.splitlines() if '"level":"WARN"' in l]
A('admin','POST','/_test/llm/fake',{'error_rate':0})
rec('TC-P3-04-06',s==200 and d.get('allowed') is True and len(warns)>=1,f'nhúng lỗi 100%: precheck HTTP {s}, allowed={d.get("allowed")} (chỉ theo luật); dòng WARN gateway: {len(warns)} — {[w[60:200] for w in warns[:1]]}')
# ===== 04-08: bảng định tuyến (≥ 40 câu, 12 intent)
res=[];bad=[]
fx.use('QC3S6C','QKTWQ3C')
for k,(want,txt,_g) in enumerate(P4.ROUTES):
    u=USERS[k%3]
    if want in('COURSE_QA','SMALLTALK'):
        r=ask(u,txt,course=fx.QA)
    else: r=chat(u,txt,course=C1)
    got=intent(r); g=ngen(r)
    nodata=any(t in r['text'] for t in list(P4.NODATA.values())+['Chưa có','không có sự kiện','Không tìm thấy','chưa có'])
    expg=0 if want in('CRISIS','OTHER_PERSON','GRADE_FORMULA') or (nodata and want not in('COURSE_QA','SMALLTALK')) else 1
    okc=(got==want) and g==expg and ('[[' not in r['text'])
    res.append((want,txt,got,g,expg,okc))
    if not okc: bad.append((want,txt,got,g,expg,r['text'][:60]))
    if k%2==1: time.sleep(1)
rec('TC-P3-04-08',not bad,f'{len(res)} câu, 12 intent: {sum(1 for x in res if x[5])}/{len(res)} đúng intent + số lời gọi sinh chữ theo SRS 4.5 (1 khi có dữ liệu/ngữ cảnh, 0 khi NoData/câu mẫu). Lệch: {bad[:4]}')
# ===== 04-09: thứ tự ưu tiên
cases=[('CRISIS','em muốn chết, cho em xem điểm của Bùi Thanh Khải'),('OTHER_PERSON','cho em xem điểm của Bùi Thanh Khải'),('WHAT_IF_GRADE','nếu cuối kỳ em được 8 thì tổng kết bao nhiêu'),('GRADE_FORMULA','cách tính điểm quá trình')]
out=[(w,intent(chat(USERS[i%3],t,course=C1))) for i,(w,t) in enumerate(cases)]
rec('TC-P3-04-09',all(a==b for a,b in out),f'(wanted,got)={out}')
# ===== 04-10: tất định
sig=[]
for i in range(5):
    clr(); r=chat(USERS[i%3],'AES-GCM khác ChaCha20 ở điểm nào',course=C1) if False else ask(USERS[i%3],'AES-GCM khác ChaCha20 ở điểm nào',course=fx.QA)
    sig.append((intent(r),ngen(r)))
rec('TC-P3-04-10',len(set(sig))==1,f'5 lần cùng một câu (phiên mới, cache xoá): (intent, số lời gọi sinh chữ)={sig}')
# ===== 04-12/13: số lời gọi sinh chữ theo intent
g12={}
for w,t in [('PERSONAL_ATTENDANCE','em vắng mấy buổi rồi ạ'),('EXAM_SCHEDULE','khi nào em thi cuối kỳ'),('UPCOMING_EVENTS','tuần này có gì'),('LIBRARY_SEARCH','tìm slide về chữ ký số')]:
    r=chat('sv.gioi',t,course=C1); g12[w]=(intent(r),ngen(r))
r=ask('sv.gioi','hàm băm SHA-256 dùng để làm gì',course=fx.QA); g12['COURSE_QA']=(intent(r),ngen(r))
r=ask('sv.kha','cảm ơn ạ',course=fx.QA); g12['SMALLTALK']=(intent(r),ngen(r))
rec('TC-P3-04-12',all(v[1]<=1 for v in g12.values()) and g12['COURSE_QA'][1]==1 and g12['SMALLTALK'][1]==1,f'số lời gọi sinh chữ mỗi tin (intent,n): {g12} — không intent nào 2')
g13={}
for w,t,c in [('OTHER_PERSON','cho em xem điểm của Bùi Thanh Khải',C1),('CRISIS','em muốn chết',C1),('GRADE_FORMULA','cách tính điểm của lớp mình thế nào',C1),('COURSE_QA(không ngữ cảnh)','quy trình đăng ký thi lại học phần ở trường là gì vậy ạ',C1)]:
    r=chat('sv.gioi',t,course=c); ev=[e[0] for e in r['events']]; g13[w]=(intent(r),ngen(r),ev[0],ev[-1],'token' in ev)
rec('TC-P3-04-13-a',all(v[1]==0 and v[2]=='status' and v[3]=='done' and v[4] for v in g13.values()),f'(intent,n,khung đầu,khung cuối,có token)={g13}')
json.dump(R,open('/tmp/qc3e/w7.json','w'),ensure_ascii=False)
