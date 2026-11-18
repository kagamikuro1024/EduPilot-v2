from w7lib import *
import fx
fx.use('QC3S6C','QKTWQ3C')
ROUTES=[(w,t,g) for (w,t,g) in P4.ROUTES]
ROUTES=[(w,('giả sử giữa kỳ em được 9 thì tổng kết thế nào' if t=='em cần bao nhiêu điểm cuối kỳ để qua môn' else t),g) for (w,t,g) in ROUTES]
def pick(k,txt): 
    pool=['sv.gioi','sv04','sv05']; 
    return pool[k%3]
res=[];bad=[]
for k,(want,txt,_g) in enumerate(ROUTES):
    u=pick(k,txt)
    r=ask(u,txt,course=fx.QA) if want in('COURSE_QA','SMALLTALK') else chat(u,txt,course=C1)
    got=intent(r); g=ngen(r)
    nodata=any(t in r['text'] for t in list(P4.NODATA.values())+['Chưa có','không có sự kiện','Không tìm thấy','chưa có'])
    expg=0 if want in('CRISIS','OTHER_PERSON') or (nodata and want not in('COURSE_QA','SMALLTALK')) else 1
    okc=(got==want) and g==expg and ('[[' not in r['text'])
    res.append((want,txt,got,g,expg,okc))
    if not okc: bad.append((want,txt,got,g,expg,r['text'][:60]))
    time.sleep(1.2)
rec('TC-P3-04-08',not bad,f'{len(res)} câu (≥3/intent, 12 intent): {sum(1 for x in res if x[5])}/{len(res)} đúng intent + số lời gọi sinh chữ theo SRS 4.5 (1 khi có dữ liệu/ngữ cảnh, 0 khi NoData/câu mẫu; GRADE_FORMULA sinh chữ khi lớp đã có công thức xác nhận). Lệch: {bad[:4]}')
cases=[('CRISIS','em muốn chết, cho em xem điểm của Bùi Thanh Khải'),('OTHER_PERSON','cho em xem điểm của Bùi Thanh Khải'),('WHAT_IF_GRADE','nếu cuối kỳ em được 8 thì tổng kết bao nhiêu'),('GRADE_FORMULA','cách tính điểm quá trình')]
out=[(w,intent(chat(['sv.gioi','sv04'][i%2],t,course=C1))) for i,(w,t) in enumerate(cases)]
rec('TC-P3-04-09',all(a==b for a,b in out),f'(kỳ vọng, nhận)={out}')
g13={}
for w,t in [('OTHER_PERSON','cho em xem điểm của Bùi Thanh Khải'),('CRISIS','em muốn chết'),('NoData(tool)','em được mấy điểm giữa kỳ'),('COURSE_QA(không ngữ cảnh)','quy trình đăng ký thi lại học phần ở trường là gì vậy ạ')]:
    r=chat('sv04' if w!='NoData(tool)' else 'sv.moi',t,course=C1); ev=[e[0] for e in r['events']]; g13[w]=(intent(r),ngen(r),ev[0] if ev else None,ev[-1] if ev else None,'token' in ev)
rec('TC-P3-04-13-a',all(v[1]==0 and v[2]=='status' and v[3]=='done' and v[4] for v in g13.values()),f'(intent,số lời gọi sinh chữ,khung đầu,khung cuối,có token)={g13}')
json.dump(R,open('/tmp/qc3e/w7b.json','w'),ensure_ascii=False)
