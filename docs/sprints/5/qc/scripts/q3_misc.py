from q3lib import *
import time
T=login('teacher'); TA=login('ta'); SV=login('sv.gioi'); AD=login('admin'); SVK=login('sv.kha')
st=json.load(open('/tmp/q3/state2.json')); C1,C2,P,qid=st['C1'],st['C2'],st['P'],st['qid']; Q=f'/courses/{C1}/questions'
def wait(job,t=90):
    for _ in range(t):
        s,j,_=call('GET','/jobs/'+job,T)
        if j.get('status') in ('DONE','FAILED','SUCCEEDED','COMPLETED','ERROR'): return j
        time.sleep(1)
    return j
s,j,_=call('POST',Q+'/suggest',T,dict(kind='MCQ',topic='Mật mã đối xứng',difficulty='MEDIUM',count=5),idem()); print('35 suggest',s,codes(j)); job=j.get('job_id')
r=wait(job) if job else {}; print('35 job',r.get('status'),json.dumps(r.get('result') or r.get('error'),ensure_ascii=False)[:300])
s,l,_=call('GET',Q+'?origin=AI_DRAFT&limit=100',T); its=l.get('items',[]); print('35 câu AI:',len(its),{(i['origin'],i['review_status']) for i in its}, 'created_by là GV:', {i.get('created_by') for i in its})
if its:
    one=its[0]; print('38 SV đọc câu AI chưa duyệt:',call('GET',f'/courses/{C1}/questions/{one["id"]}',SV)[0])
s,j,_=call('POST',Q+'/suggest',T,dict(kind='MCQ',topic='x',count=5)); print('35 thiếu Idempotency-Key',s)
s,j,_=call('POST',Q+'/suggest',T,dict(kind='MCQ',topic='x',count=0),idem()); print('35 count 0',s,codes(j)); s,j,_=call('POST',Q+'/suggest',T,dict(kind='MCQ',topic='x',count=21),idem()); print('35 count 21',s,codes(j))
s,j,_=call('POST',Q+'/suggest',SV,dict(kind='MCQ',topic='x',count=1),idem()); print('44 SV suggest',s)
# AI tests for the code question
s,j,_=call('POST',Q+'/suggest',T,dict(kind='CODE_TESTS',question_id=qid,count=5),idem()); print('40 suggest tests',s,codes(j)); r=wait(j.get('job_id')) if j.get('job_id') else {}; print('40 job',r.get('status'),json.dumps(r.get('result') or r.get('error'),ensure_ascii=False)[:260])
s,l,_=call('GET',P+'/testcases?limit=100',T); ai=[x for x in l.get('items',[]) if x.get('source')=='AI_DRAFT']; print('40 test AI:',len(ai),{(x.get('approved')) for x in ai}, 'tổng',len(l.get('items',[])))
s,g,_=call('GET',P,T); print('41 total (chỉ duyệt) =',g['code']['tests'])
# 43 filters
for q in ['review_status=PENDING','topic=M%E1%BA%ADt%20m%C3%A3','difficulty=HARD','type=CODE','origin=MANUAL','archived=true','q=c%E1%BB%99ng','q=C%E1%BB%98NG','q='+'x'*100,'q='+'x'*101,'limit=0','limit=101','limit=5']:
    s,l,_=call('GET',Q+'?'+q,T); print('43',q[:30],s,len(l.get('items',[])) if s==200 else codes(l))
s,l,_=call('GET',Q+'?limit=2',T); c=l.get('next_cursor'); s2,l2,_=call('GET',Q+'?limit=2&cursor='+(c or 'x'),T); print('43 cursor',bool(c),s2,'rác:',call('GET',Q+'?cursor=rac',T)[0])
# 44 permission matrix on a sample of ops
ops=[('GET',Q),('POST',Q),('GET',P),('PUT',P),('POST',P+'/archive'),('POST',P+'/duplicate'),('PUT',P+'/review'),('PUT',P+'/code'),('GET',P+'/testcases'),('POST',P+'/testcases'),('POST',P+'/testcases/approve'),('POST',P+'/reference/verify'),('POST',Q+'/suggest')]
B={'POST'+Q:dict(type='TRUE_FALSE',title='t',topic='x',stem='s',value=True)}
res={}
for who,tok in [('SV',SV),('ADMIN',AD),('none',None),('SVkha',SVK)]:
    codes_=[]
    for m,pth in ops:
        s,j,_=call(m,pth,tok,{} if m!='GET' else None,idem()); codes_.append(s)
    res[who]=codes_
print('44 ma trận (SV, ADMIN, không token) cho 13 thao tác:',res)
# lớp khác: GV lớp 761987 gọi qua đường dẫn lớp C2
print('44 câu của C1 qua C2',call('GET',f'/courses/{C2}/questions/{qid}',T)[0],call('PUT',f'/courses/{C2}/questions/{qid}',T,{})[0],call('POST',f'/courses/{C2}/questions/{qid}/duplicate',T,{})[0])
# 46 duplicate
s,j,_=call('POST',P+'/duplicate',T,{}); print('46 nhân bản',s,j.get('title'),j.get('review_status'),j.get('origin'),(j.get('code') or {}).get('tests_version'),(j.get('code') or {}).get('reference_verified_version'),(j.get('code') or {}).get('tests'))
# 53 today
for w,tok in [('GV',T),('TA',TA),('SV',SV)]:
    s,j,_=call('GET','/me/today',tok); print('53',w,[ (a.get('kind'),a.get('title'),a.get('href')) for a in j.get('actions',[]) if 'QUESTION' in a.get('kind','')])
