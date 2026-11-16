# P103-37/38/39/41 + P104-34: ngân sách trên stack cô lập. Giá 125.000 đ/Mtok (in+out) → mỗi lời gọi stub (5+3 token) tốn đúng 1,0 đ.
from smk3iso import *
V1='http://host.docker.internal:9751/v1'; V2='http://host.docker.internal:9752/v1'
def mkp(name,u,model,price):
    ex=[p for p in ic('GET','/admin/llm/providers')[1]['items'] if p['name']==name]
    if ex: return ex[0]
    s,j,_=ic('POST','/admin/llm/providers',{'type':'openai_compatible','name':name,'base_url':u,'api_key':'good-key','models':[{'model':model,'kind':'chat','price_in':str(price),'price_out':str(price)}]}); assert s==201,(s,j); return j
B1=mkp('SmkI-B1',V1,'m-ok',125000); B2=mkp('SmkI-B2',V2,'m-ok',1000)   # B2 rẻ hơn nhiều: 0,008 đ/lời gọi
for t in ('CHAT','CLASSIFY','GRADING'): print(t,iput(t,[mid(B1,'m-ok')])[0])
# vô hiệu hoá mọi mô hình rẻ khác (nhà cũ T1,T2,S1,S2 đã giá 1.000.000 hoặc tắt) — chỉ B2 rẻ nhất đang bật
cheap=ic('GET','/admin/llm/providers')[1]['items']
print([(p['name'],p['enabled'],[(m['model'],m['price_in'],m['price_out'],m['enabled']) for m in p['models'][:1]]) for p in cheap])
def bud(): return ic('GET','/admin/llm/budget')[1]
def bput(d,m,ver=None,raw=None):
    b=bud(); v=b['version'] if ver is None else ver
    body={'daily_limit':d,'monthly_limit':m,'version':v}
    return ic('PUT','/admin/llm/budget',body,hdr={'If-Match':'W/"v%d"'%v} if v>0 else None)
# làm sạch bộ đếm: audit + khoá Redis ngân sách
ipsql("delete from llm_audit"); [ird('DEL',k) for k in ird('--scan','--pattern','ep:llm:budget:*').split()]; ipsql("delete from outbox where topic='llm.budget.warn'")
print('bud sau reset',bud())
r=bput('100','1000'); print('PUT 100/1000',r[0],r[1])
ob0=int(ipsql("select count(*) from outbox where topic='llm.budget.warn'"))
states=[]; wlog0=len([l for l in gwlogs('10m') if 'ngân sách' in l and 'WARN' in l])
for k in range(1,111):
    r=ichat('CHAT'); b=bud(); states.append((k,r[0],r[1].get('provider'),b['spent_today'],b['pct_today'],b['state']))
    if k in(1,78,79,80,81,84,85,90,99,100,101): print(states[-1], 'outbox',ipsql("select count(*) from outbox where topic='llm.budget.warn'"))
first=lambda st:next((s for s in states if s[5]==st),None)
print('first warn',first('warn'),'first exhausted',first('exhausted'))
ob1=int(ipsql("select count(*) from outbox where topic='llm.budget.warn'")); print('outbox warn events',ob1-ob0, ipsql("select payload from outbox where topic='llm.budget.warn' order by created_at limit 3"))
print('tiền: spent_today=',bud()['spent_today'],'audit sum=',ipsql("select sum(cost_est) from llm_audit"),'số dòng',ipsql("select count(*) from llm_audit"))
# ở ≥100%
b=bud(); print('state',b['state'],b['pct_today'])
g=ichat('GRADING'); print('GRADING(BATCH) khi hết',g[0],json.dumps(g[1],ensure_ascii=False)[:300])
for t in ('CHAT','CLASSIFY'):
    r=ichat(t); print(t,'khi hết ->',r[0],r[1].get('provider'),r[1].get('model'),r[1].get('text','')[:40])
time.sleep(2); print(ipsql("select task,provider,model,status,cost_est from llm_audit order by created_at desc limit 4"))
json.dump(states,open('/tmp/smk34/budget-states.json','w'))
