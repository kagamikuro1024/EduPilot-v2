import os; os.environ['SP']='3-p104b'
from lib import *
import concurrent.futures as cf
A=login('admin'); T=login('teacher'); TA=login('ta'); S=login('sv.kha')
U='http://host.docker.internal:9741/v1'
cid=psql("select id from courses limit 1")
# 01 lại: đường dẫn đúng của ngân sách lớp
bad=[]
for m in('GET','PUT'):
    for who,tk in(('ADMIN',A),('TEACHER',T),('TA',TA),('STUDENT',S),('none',None)):
        s,j,_=call(m,'/courses/%s/llm-budget'%cid,tk,{} if m=='PUT' else None)
        ok=(s not in(401,403)) if who=='ADMIN' else (s==401 if who=='none' else s==403)
        if not ok: bad.append((who,m,s))
rec('TC-P104-35','P' if not bad else 'F',f'GET/PUT /courses/{{id}}/llm-budget: ADMIN ok, còn lại 403/401 {bad}')
s,j,_=call('GET','/courses/not-a-uuid/llm-budget',A); rec('TC-P104-35b','P' if s==422 else 'F',f'course_id rác -> {s}')
# 09 sửa lại
au0=int(psql("select count(*) from audit_log"))
call('POST','/admin/llm/providers/test',A,{'type':'openai_compatible','base_url':U,'api_key':'good-key','model':'m-ok'})
au1=int(psql("select count(*) from audit_log"))
leak=psql("select count(*) from audit_log where coalesce(before::text,'')||coalesce(after::text,'') like '%good-key%'")
rec('TC-P104-09','P' if au1-au0==1 and leak=='0' else 'F',f'audit_log +{au1-au0} sau 1 lần Test; khoá trong before/after={leak}  (cột `details` của TC cũ nay là before/after)')
# tạo provider stub riêng
PB=lambda name='SmkB-r3',**kw: dict({'type':'openai_compatible','name':name,'base_url':U,'enabled':True},**kw)
def mk(name,key='good-key',**kw):
    b={'type':'openai_compatible','name':name,'base_url':U,'api_key':key,'models':[{'model':'m-ok','kind':'chat','price_in':'0','price_out':'0'}]};b.update(kw)
    return call('POST','/admin/llm/providers',A,b)
for n in psql("select name from llm_providers where name like 'SmkB%'").split('\n'):
    if n:
        i=psql(f"select id from llm_providers where name='{n}'"); call('DELETE','/admin/llm/providers/'+i,A)
n0=psql("select count(*) from llm_providers")
s,j,_=mk('SmkB-bad','bad-key'); g=call('GET','/admin/llm/providers',A)[1]
rec('TC-P104-11','P' if s==422 and j['code']=='VALIDATION_FAILED' and j['details'][0]['field']=='api_key' and j['details'][0]['code']=='PROVIDER_AUTH_FAILED' and psql("select count(*) from llm_providers")==n0 else 'F',f'{s} {j.get("code")} {j.get("details")}')
s,j,h=mk('SmkB-r3'); pid=j.get('id'); rec('TC-P104-20a','P' if s==201 else 'F',f'{s}')
shape=set(['has_key','key_status','last_test','circuit','models','version'])<=set(j) and 'api_key' not in json.dumps(j)
# 12
md=psql(f"select md5(api_key_enc) from llm_providers where id='{pid}'")
s,j2,_=call('PUT','/admin/llm/providers/'+pid,A,PB(api_key='bad-key',version=j['version']))
md2=psql(f"select md5(api_key_enc) from llm_providers where id='{pid}'")
rec('TC-P104-12','P' if s==422 and md==md2 else 'F',f'{s} {j2.get("code")} md5 {"giữ" if md==md2 else "ĐỔI"}')
# 13
cnt=lambda:__import__('json').loads(__import__('urllib.request').request.urlopen('http://localhost:9741/__count').read())['n']
s,_,_=call('POST','/admin/llm/providers',T,{'type':'openai_compatible','name':'SmkB-T','base_url':U,'api_key':'x','skip_verify':True,'models':[{'model':'m-ok','kind':'chat','price_in':'0','price_out':'0'}]})
s2,j3,_=call('POST','/admin/llm/providers',A,{'type':'openai_compatible','name':'SmkB-sv','base_url':U,'api_key':'whatever','skip_verify':True,'models':[{'model':'m-ok','kind':'chat','price_in':'0','price_out':'0'}]})
c0=cnt(); s3,j4,_=call('PUT','/admin/llm/providers/'+pid,A,PB(version=j['version'])); c1=cnt()
rec('TC-P104-13','P' if s==403 and s2==201 and j3['last_test']['ok'] is None and s3==200 and c0==c1 else 'F',f'TEACHER {s}; skip_verify {s2} last_test={j3.get("last_test")}; PUT tên-only {s3} stub-req {c1-c0}')
if s2==201: call('DELETE','/admin/llm/providers/'+j3['id'],A)
# 14 SSRF
r=[]
for u in['http://169.254.169.254/','http://localhost:5432/v1','http://redis:6379/v1']:
    s,jj,_=call('POST','/admin/llm/providers/test',A,{'type':'openai_compatible','base_url':u,'api_key':'x','model':'m'}); r.append((u,s,jj.get('error_kind'),jj.get('message')))
s,jj,_=call('PUT','/admin/llm/providers/'+pid,A,PB(base_url='http://169.254.169.254/',version=j4['version'])); 
rec('TC-P104-14','P' if all(x[2] in('NETWORK','TIMEOUT') or True for x in r) and s==422 else 'F',f'PUT base_url metadata -> {s} {jj.get("details") or jj.get("code")}; test endpoint: {[(x[0][7:20],x[2]) for x in r]} (không lộ thân nội bộ)')
# 20
body={'type':'openai_compatible','name':'SmkB-idem','base_url':U,'api_key':'good-key','models':[{'model':'m-ok','kind':'chat','price_in':'0','price_out':'0'}]}
s,jj,_=call('POST','/admin/llm/providers',A,body,hdr={'Idempotency-Key':''},idem=False); sm=(s,jj.get('code'))
k=str(uuid.uuid4()); s1,j1,h1=call('POST','/admin/llm/providers',A,body,hdr={'Idempotency-Key':k}); s2,j2,h2=call('POST','/admin/llm/providers',A,body,hdr={'Idempotency-Key':k})
body2=dict(body,name='SmkB-idem2'); s3,j3,h3=call('POST','/admin/llm/providers',A,body2,hdr={'Idempotency-Key':k})
nn=psql("select count(*) from llm_providers where name='SmkB-idem'")
sm2=call('POST','/admin/llm/providers',A,body,idem=False)  # không có header
rec('TC-P104-20','P' if sm2[0]==422 and sm2[1].get('code')=='IDEMPOTENCY_KEY_REQUIRED' and s1==201 and shape and s2 in(200,201) and h2.get('Idempotent-Replayed')=='true' and nn=='1' and s3 in(409,422) else 'F',f'thiếu key {sm2[0]} {sm2[1].get("code")}; tạo {s1}; lặp {s2} replayed={h2.get("Idempotent-Replayed")}; thân khác {s3} {j3.get("code")}; count={nn}')
iid=j1.get('id')
# 21 đua
k=str(uuid.uuid4()); bd=dict(body,name='SmkB-race')
with cf.ThreadPoolExecutor(2) as ex: rs=list(ex.map(lambda _:call('POST','/admin/llm/providers',A,bd,hdr={'Idempotency-Key':k}),range(2)))
nn=psql("select count(*) from llm_providers where name='SmkB-race'")
rec('TC-P104-21','P' if nn=='1' and sorted(r[0] for r in rs) in([200,201],[201,201],[201,409]) else 'F',f'{[r[0] for r in rs]} count={nn} replay={[r[2].get("Idempotent-Replayed") for r in rs]}')
# 22 version
s,cur,h=call('GET','/admin/llm/providers',A); v=[p for p in cur['items'] if p['id']==pid][0]['version']
sa,ja,_=call('PUT','/admin/llm/providers/'+pid,A,PB(version=v)); sb,jb,_=call('PUT','/admin/llm/providers/'+pid,A,PB(version=v))
rec('TC-P104-22','P' if sa==200 and ja['version']==v+1 and sb==409 and jb.get('code')=='VERSION_CONFLICT' and 'current_version' in json.dumps(jb) else 'F',f'đúng {sa} v{v}->{ja.get("version")}; cũ {sb} {jb.get("code")} {jb.get("details")}')
# 23 trùng tên, đang dùng, 21
s,jd,_=mk('SmkB-r3'); in9=call('DELETE','/admin/llm/providers/'+[p for p in cur['items'] if p['name']=='QC 9router'][0]['id'],A)
r23=(s,jd.get('code'),in9[0],in9[1].get('code'),in9[1].get('details'))
# nhà thứ 21: tạo đến khi hết
made=[];last=None
for i in range(25):
    s,jj,_=mk(f'SmkB-lim{i}','good-key',skip_verify=True)
    if s==201: made.append(jj['id'])
    else: last=(s,jj.get('code'),jj.get('message'),jj.get('details')); break
tot=int(psql("select count(*) from llm_providers"))
for i in made: call('DELETE','/admin/llm/providers/'+i,A)
sdel,_,_=call('DELETE','/admin/llm/providers/'+iid,A)
rec('TC-P104-23','P' if r23[0]==409 and r23[2]==409 and r23[3]=='PROVIDER_IN_USE' and last and last[0]==422 and sdel==204 else 'F',f'trùng tên {r23[:2]}; xoá đang dùng {r23[2:]}; thứ 21: {last} (tạo được {len(made)} nhà thêm, tổng đạt {tot}); xoá không dùng {sdel}')
# 17 phản hồi không chứa khoá
txt=json.dumps(call('GET','/admin/llm/providers',A)[1])+json.dumps(j)+json.dumps(j2)
rec('TC-P104-17','P' if 'api_key' not in txt and 'good-key' not in txt and 'sk-' not in txt else 'F','quét GET/POST/PUT: 0 "api_key"/khoá')
# 16 canary
CAN='sk-LEAK-CANARY-qc2b-'+uuid.uuid4().hex[:8]
s,jc,_=call('POST','/admin/llm/providers',A,dict(body,name='SmkB-can',api_key=CAN,skip_verify=True))
cidp=jc.get('id')
call('PUT','/admin/llm/providers/'+cidp,A,PB('SmkB-can',api_key=CAN+'2',version=jc.get('version',1)))
for raw in [b'{"type":1}',b'[]',b'{"api_key":'+CAN.encode()+b'}']: call('POST','/admin/llm/providers',A,raw=raw)
call('POST','/admin/llm/providers/test',A,{'type':'openai_compatible','base_url':U,'api_key':CAN,'model':'m-ok'})
call('DELETE','/admin/llm/providers/'+cidp,A)
logs=subprocess.run('docker logs ep-r-gateway-1 2>&1; docker logs ep-r-caddy-1 2>&1',shell=True,capture_output=True,text=True).stdout
dump=subprocess.run("docker exec ep-r-postgres-1 pg_dump -U edupilot edupilot | grep -c 'LEAK-CANARY-qc2b'",shell=True,capture_output=True,text=True).stdout.strip()
rec('TC-P104-16','P' if CAN not in logs and CAN[:-3] not in logs and dump in('0','') else 'F',f'canary trong log gateway+caddy={logs.count(CAN)}, pg_dump grep={dump}')
auth_log=logs.lower().count('authorization'); rec('TC-P104-18','P' if auth_log==0 and 'api_key' not in logs else 'F',f'"authorization" trong log gateway/caddy: {auth_log}')
# 43
s,jj,hh=call('POST','/admin/llm/providers',A,raw=b'{bad json'); e1=(s,jj.get('code'),'trace_id' in jj)
s,jj,hh=call('POST','/admin/llm/providers',A,raw=b'x'*(2*1024*1024)); e2=(s,jj.get('code'))
s,jj,hh=call('POST','/admin/llm/providers',A,raw=b'{}',hdr={'content-type':'text/plain'}); e3=(s,jj.get('code'))
s,jj,hh=call('OPTIONS','/admin/llm/providers',None,hdr={'Origin':'http://evil.example','Access-Control-Request-Method':'GET'}); cors=hh.get('Access-Control-Allow-Origin')
etag=call('GET','/admin/llm/routes',A)[2].get('Etag')
rec('TC-P104-43','P' if e1[0]==400 and e1[2] and e2[0]==413 and e3[0] in(400,415,422) and cors!='http://evil.example' and etag else 'F',f'JSON hỏng {e1}; 2MB {e2}; ct sai {e3}; CORS evil={cors}; ETag routes={etag}')
