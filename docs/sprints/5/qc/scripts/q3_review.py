from q3lib import *
import time
T=login('teacher'); TA=login('ta'); SV=login('sv.gioi'); AD=login('admin')
s,j,_=call('GET','/me/courses',T); C1=[c['course']['id'] for c in j['items'] if c['course']['class_code']=='761987'][0]; C2=[c['course']['id'] for c in j['items'] if c['course']['class_code']=='761988'][0]
Q=f'/courses/{C1}/questions'
REF='#include <stdio.h>\nint main(){int a,b; scanf("%d %d",&a,&b); printf("%d\\n",a+b); return 0;}\n'
def newcode():
    s,j,_=call('POST',Q,T,dict(type='CODE',title='Cộng hai số QC',topic='QC',stem='a+b')); return j['id'],j['version']
def get(P): return call('GET',P,T)[1]
def wait(job,t=90):
    for _ in range(t):
        s,j,_=call('GET','/jobs/'+job,T)
        if j.get('status') in ('DONE','FAILED','SUCCEEDED','COMPLETED','ERROR'): return j
        time.sleep(1)
    return j
qid,v=newcode(); P=Q+'/'+qid
s,j,_=call('PUT',P+'/code',T,dict(version=v,languages=['c11','cpp17'],reference=dict(language='c11',source=REF))); print('ref set',s,codes(j)); 
d=lambda: get(P)['code']
tv=[d()['tests_version']]
s,_,_=call('POST',P+'/testcases',T,dict(name='sample1',input='1 2\n',expected='3\n',is_sample=True,weight=1)); tv.append(d()['tests_version'])
s,j,_=call('POST',P+'/testcases',T,dict(name='h1',input='5 6\n',expected='11\n',weight=1)); h1=j.get('id'); tv.append(d()['tests_version'])
s,j,_=call('POST',P+'/testcases',T,dict(name='h2',input='10 20\n',expected='30\n',weight=2)); h2=j.get('id'); tv.append(d()['tests_version'])
print('14 tests_version sau thêm 3 test:',tv)
s,j,_=call('PUT',P+f'/testcases/{h1}',T,dict(expected='11\n')); print('14 PUT test',s,d()['tests_version'])
# TC-28: điều kiện duyệt
def ap(dec='APPROVE'):
    g=get(P); s,j,_=call('PUT',P+'/review',T,dict(decision=dec,version=g['version'])); return s,codes(j)
q0,_=newcode(); P0=Q+'/'+q0
s,j,_=call('PUT',P0+'/review',T,dict(decision='APPROVE',version=1)); print('28 không test',s,codes(j))
call('POST',P0+'/testcases',T,dict(name='s',input='1',expected='1',is_sample=True)); print('28 chỉ test mẫu',ap.__name__ and call('PUT',P0+'/review',T,dict(decision='APPROVE',version=get(P0)['version']))[0:2][0], codes(call('PUT',P0+'/review',T,dict(decision='APPROVE',version=get(P0)['version']))[1]))
q1,_=newcode(); P1=Q+'/'+q1; call('POST',P1+'/testcases',T,dict(name='h',input='1',expected='1',weight=0)); print('28 chỉ test ẩn weight 0',codes(call('PUT',P1+'/review',T,dict(decision='APPROVE',version=get(P1)['version']))[1]))
print('28 chưa verify',ap())
# verify
k=idem(); s,j,_=call('POST',P+'/reference/verify',T,{},k); job=j.get('job_id'); print('23 verify',s,job and 'job'); s2,j2,_=call('POST',P+'/reference/verify',T,{},k); print('25 cùng key → cùng job:',j2.get('job_id')==job); print('25 thiếu key',call('POST',P+'/reference/verify',T,{})[0])
r=wait(job); print('23 job',r.get('status'),json.dumps(r.get('result'))[:300])
c=d(); print('23 cờ',c['reference_verified_version'],'tests_version',c['tests_version'])
print('27 duyệt (GV)',ap('REQUEST'),ap('APPROVE'))
print('29 AI/SV: SV duyệt',call('PUT',P+'/review',SV,dict(decision='APPROVE',version=1))[0],'ADMIN',call('PUT',P+'/review',AD,dict(decision='APPROVE',version=1))[0])
# đổi test sau duyệt → DRAFT, cờ verify hết
call('PUT',P+f'/testcases/{h2}',T,dict(expected='31\n')); g=get(P); print('28 đổi test sau duyệt: review_status',g['review_status'],'cờ',g['code']['reference_verified_version'],'tv',g['code']['tests_version'])
s,j,_=call('POST',P+'/reference/verify',T,{},idem()); r=wait(j['job_id']); res=r.get('result') or {}; print('24 verify lệch: ok=',res.get('ok'),[ (p['name'],p['verdict']) for p in res.get('per_test',[])], 'cờ không ghi:',get(P)['code']['reference_verified_version'] is None)
call('PUT',P+f'/testcases/{h2}',T,dict(expected='30\n'))
s,j,_=call('POST',P+'/reference/verify',T,{},idem()); r=wait(j['job_id']); print('23 verify lại',(r.get('result') or {}).get('ok'))
print('27 TA duyệt',call('PUT',P+'/review',TA,dict(decision='REQUEST',version=get(P)['version']))[0], call('PUT',P+'/review',TA,dict(decision='APPROVE',version=get(P)['version']))[0])
print('27 sai version',call('PUT',P+'/review',T,dict(decision='REJECT',version=1))[0:2][0])
open('/tmp/q3/state2.json','w').write(json.dumps(dict(P=P,qid=qid,C1=C1,C2=C2)))
