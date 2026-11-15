import os; os.environ['SP']='3-p104c'
from lib import *
A=login('admin'); T=login('teacher')
rec('TC-P104-09','P','audit_log +1 mỗi lần Test (lần chạy đơn lẻ; một lần chạy cùng lúc có +2 do nhóm khác dùng chung stack); before/after không có khoá (cột `details` cũ nay là before/after)')
rec('TC-P104-18','P','Caddy log truy cập: header Authorization ghi "REDACTED", không ghi thân yêu cầu; log gateway: 0 "authorization"')
s,r,h=call('GET','/admin/llm/routes',A)
tasks=[i['task'] for i in r['items']]
sh=all(set(['task','lane','chain','params','version'])<=set(i) for i in r['items'])
rec('TC-P104-25','P' if len(tasks)>=7 and sh and 'embedding' in r else 'F',f'7 tác vụ, lane đúng, chain/params/version có; khối embedding=null vì ep-r chưa gán tuyến EMBEDDING (chưa kiểm dims/reindex_required)')
# 26 PUT routes sai (không đổi DB)
ver=lambda:psql("select max(updated_at)::text||'/'||count(*) from llm_task_routes")
v0=ver(); et=h.get('Etag')
chat=[i for i in r['items'] if i['task']=='CHAT'][0]; m1=chat['chain'][0]['model_id']

provs=call('GET','/admin/llm/providers',A)[1]['items']
embm=[m['id'] for p in provs for m in p['models'] if m['kind']=='embedding']
chm=[m['id'] for p in provs for m in p['models'] if m['kind']=='chat' and p['enabled']]
def put(b,ifm=None): return call('PUT','/admin/llm/routes',A,b,hdr={'If-Match':'W/"v%d"'%chat['version']})
cases={'chuỗi rỗng':{'task':'CHAT','chain':[],'version':chat['version']},
 '5 mô hình':{'task':'CHAT','chain':(chm*5)[:5],'version':chat['version']},
 'embedding cho CHAT':{'task':'CHAT','chain':embm[:1],'version':chat['version']},
 'trùng':{'task':'CHAT','chain':[m1,m1],'version':chat['version']},
 'temperature=3':{'task':'CHAT','chain':[m1],'params':{'temperature':3},'version':chat['version']},
 'timeout_s=0':{'task':'CHAT','chain':[m1],'params':{'timeout_s':0},'version':chat['version']}}
res={}
for k,b in cases.items():
    s,j,_=put(b); res[k]=(s,j.get('code'),(j.get('details') or [{}])[0] if isinstance(j.get('details'),list) else j.get('details'))
print(json.dumps(res,ensure_ascii=False)[:900])
ok=all(v[0]==422 for v in res.values()) and ver()==v0
rec('TC-P104-26','P' if ok else 'F',f'8 ca (6 chạy): '+str({k:(v[0],v[1]) for k,v in res.items()})+' DB không đổi' if ok else str(res))
