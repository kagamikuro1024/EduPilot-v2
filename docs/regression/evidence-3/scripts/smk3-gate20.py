# TC-GATEP1-20: chạy lại ma trận quyền 13×6 (TC-P104-01) + tấn công token (P104-02/03/04) trên ep-r hiện tại (qc2/SmkO)
import os; os.environ['SP']='3-gate20'
from lib import *
for _ in range(8):
    subprocess.run(['docker','exec','ep-r-redis-1','sh','-c',"redis-cli --scan --pattern 'ep:rl:auth*' | xargs -r redis-cli del"],capture_output=True)
    try: A=login('admin'); T=login('teacher'); TA=login('ta'); S=login('sv.kha'); break
    except Exception as e: time.sleep(3)
base=A
sec=secret()
exp=forge(A,exp=1700000000,iat=1699999000,nbf=1699999000)
bad=forge(A,sec='wrong-secret-xxxxxxxxxxxxxxxxxxxxxxxx')
none=forge(A,alg='none'); roleT=forge(T,role='ADMIN',sec='wrong-secret-xxxxxxxxxxxxxxxxxxxxxxxx')
su=forge(A,role='SUPERUSER'); nosub=forge(A,sub=None)
pid=call('GET','/admin/llm/providers',A)[1]['items'][2]['id']
cid=psql("select id from courses limit 1")
ops=[('GET','/admin/llm/providers',None),('POST','/admin/llm/providers',{}),('PUT','/admin/llm/providers/'+pid,{}),('DELETE','/admin/llm/providers/'+pid,None),
 ('POST','/admin/llm/providers/test',{'type':'fake','api_key':'x','model':'fake-chat'}),('POST','/admin/llm/providers/'+pid+'/test',{}),
 ('GET','/admin/llm/routes',None),('PUT','/admin/llm/routes',{}),('GET','/admin/llm/usage',None),('GET','/admin/llm/budget',None),('PUT','/admin/llm/budget',{}),
 ('GET','/courses/'+cid+'/llm-budget',None),('PUT','/courses/'+cid+'/llm-budget',{})]
reads={'/admin/llm/providers','/admin/llm/routes','/admin/llm/usage','/admin/llm/budget'}
before=psql("select (select count(*) from llm_providers),(select count(*) from llm_task_routes),(select count(*) from llm_budgets)")
bad_cases=[]
for m,p,b in ops:
    for who,tk in (('ADMIN',A),('TEACHER',T),('TA',TA),('STUDENT',S),('none',None),('expired',exp)):
        # ghi bằng ADMIN với thân rỗng/DELETE có thể có tác dụng -> dùng thân sai để chỉ kiểm quyền (DELETE ADMIN bỏ qua)
        if who=='ADMIN' and m in('DELETE',): continue
        s,j,h=call(m,p,tk,b if b is not None else None) if tk else call(m,p,None,b)
        code=(j or {}).get('code')
        if who=='ADMIN': ok= s not in(401,403)
        elif who=='TEACHER': ok=(s==200) if (m=='GET' and p in reads) else s==403
        elif who in('TA','STUDENT'): ok= s==403
        elif who=='none': ok= s==401
        else: ok= s==401 and code=='TOKEN_EXPIRED'
        if not ok: bad_cases.append((who,m,p,s,code))
after=psql("select (select count(*) from llm_providers),(select count(*) from llm_task_routes),(select count(*) from llm_budgets)")
print('rbac bad',bad_cases[:8],len(bad_cases),before,after)
rec('TC-P104-01','P' if not bad_cases and before==after else 'F', f'78 ca; lệch {len(bad_cases)} {bad_cases[:3]}')
# 02 chặn trước DB: 403 thân rác (TEACHER)
s,j,h=call('POST','/admin/llm/providers',T,raw=b'{garbage'); rec('TC-P104-02','P' if s==403 and set(['code','message','trace_id'])<=set(j) else 'F',f'{s} {list(j)}')
# 03 tấn công token
bad3=[]
for name,t in [('wrongsecret',bad),('alg-none',none),('role-edit',roleT),('SUPERUSER-role(valid sig)',su),('nosub',nosub),('Bearer empty',''),('Bearer a.b.c','a.b.c')]:
    for m,p,b in [('GET','/admin/llm/providers',None),('PUT','/admin/llm/routes',{})]:
        s,j,h=call(m,p,None,b,hdr={'authorization':'Bearer '+t})
        if s!=401: bad3.append((name,m,s,j.get('code')))
rec('TC-P104-03','P' if not bad3 else 'F',f'lệch: {bad3}' if bad3 else '14 ca đều 401')
# 04 leo thang
bad4=[]
for hd in [{'X-Role':'ADMIN'},{'X-User-Id':'x'}]:
    s,j,h=call('PUT','/admin/llm/routes',S,{},hdr=hd); 
    if s!=403: bad4.append(s)
