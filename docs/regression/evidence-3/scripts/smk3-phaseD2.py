from smk3iso import *
# P104-34 + P103-39 (hai phạm vi)
b0=ic('GET','/admin/llm/budget')[1]; print('GET',json.dumps(b0))
print('keys',sorted(b0),'tiền là chuỗi:',all(isinstance(b0[k],str) for k in('daily_limit','monthly_limit','spent_today','spent_month')))
def bud(): return ic('GET','/admin/llm/budget')[1]
def bput(body,ver=None):
    b=bud(); v=b['version'] if ver is None else ver
    return ic('PUT','/admin/llm/budget',dict(body,version=v),hdr={'If-Match':'W/"v%d"'%v} if v>0 else None)
cs={}
for nm,body in [('daily>monthly',{'daily_limit':'100','monthly_limit':'50'}),('âm',{'daily_limit':'-1','monthly_limit':None}),('abc',{'daily_limit':'abc','monthly_limit':None}),('monthly âm',{'daily_limit':None,'monthly_limit':'-5'}),('ok 100000/2000000',{'daily_limit':'100000','monthly_limit':'2000000'}),('null',{'daily_limit':None,'monthly_limit':None})]:
    r=bput(body); cs[nm]=(r[0],r[1].get('code') or r[1].get('state'),r[1].get('daily_limit'),r[1].get('monthly_limit'))
for k,v in cs.items(): print(' ',k,v)
v=bud()['version']; r=ic('PUT','/admin/llm/budget',{'daily_limit':None,'monthly_limit':None,'version':v-1 if v>1 else 1},hdr={'If-Match':'W/"v%d"'%max(v-1,1)}); print(' version cũ',r[0],r[1].get('code'))
# hiệu lực ngay: đặt daily = spent_today + 1,5 → lời gọi kế BATCH vẫn qua? rồi exhausted
b=bud(); sp=float(b['spent_today'])
r=bput({'daily_limit':'%.2f'%(sp-1),'monthly_limit':None}); print('đặt trần dưới chi tiêu',r[0],r[1].get('state'),r[1].get('pct_today'))
g=ichat('GRADING'); print('GRADING ngay sau PUT (hết ngân sách?)',g[0],g[1].get('code'),(g[1].get('details') or {}))
bput({'daily_limit':None,'monthly_limit':None})
g=ichat('GRADING'); print('GRADING sau khi bỏ trần',g[0],g[1].get('provider'))
# ---- hai phạm vi
A=ic('GET','/auth/me')[1] if False else None
uid=ipsql("select id from users where email='admin@smk3.test'")
print(ipsql(f"insert into courses(subject_code,class_code,name,semester,join_code,created_by) values('INT1006','SMK3A','Lớp smk3 A','2026-2027-HK1','SKMQQQ7','{uid}') on conflict do nothing"))
cid=ipsql("select id from courses where class_code='SMK3A'"); print('course',cid)
s,j,_=ic('GET','/courses/%s/llm-budget'%cid); print('GET course budget',s,json.dumps(j)[:250])
v=j.get('version',0)
s,j,_=ic('PUT','/courses/%s/llm-budget'%cid,{'daily_limit':'5','monthly_limit':'1000','version':v},hdr={'If-Match':'W/"v%d"'%v} if v>0 else None); print('PUT course budget',s,json.dumps(j)[:250])
# chi tiêu cho lớp: 5 lời gọi 1,0 đ → 100 % của lớp; hệ thống chưa chạm (trần hệ thống bỏ)
iput('CHAT',[mid(imkprov('SmkI-B1',U1,['m-ok']),'m-ok')]) if False else None
rs=[]
for i in range(7):
    r=ichat('CHAT',course_id=cid); st=ic('GET','/courses/%s/llm-budget'%cid)[1]; rs.append((r[0],r[1].get('provider'),st.get('spent_today'),st.get('state')))
print('lớp:',rs)
g=ichat('GRADING',course_id=cid); print('GRADING lớp (hết):',g[0],(g[1].get('details') or {}),'| hệ thống:',bud()['state'])
g2=ichat('GRADING'); print('GRADING không course (hệ thống chưa cạn):',g2[0],g2[1].get('provider'))
print('outbox warn:',ipsql("select payload from outbox where topic='llm.budget.warn' order by created_at"))
time.sleep(2); print('audit sum',ipsql("select sum(cost_est),count(*) from llm_audit"),'| spent_month',bud()['spent_month'])
