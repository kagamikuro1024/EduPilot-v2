# TC-GATEP1-12 / 15 / 17 trên stack cô lập smk3 (gw1 :18201): 2 nhà `fake`, đổi CHAT, chat ngay sau mỗi lần; fallback (a) tắt, (b) lỗi còn bật; mọi nhà chết + passages
from smk3iso import *
def mkfake(name,model):
    ex=[p for p in ic('GET','/admin/llm/providers')[1]['items'] if p['name']==name]
    if ex: return ex[0]
    s,j,_=ic('POST','/admin/llm/providers',{'type':'fake','name':name,'models':[{'model':model,'kind':'chat','price_in':'1000','price_out':'1000'}]}); assert s==201,(s,j); return j
F1=mkfake('SmkG-F1','fake-g1'); F2=mkfake('SmkG-F2','fake-g2')
print('--- 12: đổi CHAT giữa hai nhà fake')
rows=[]
for i,(P,m) in enumerate([(F1,'fake-g1'),(F2,'fake-g2'),(F1,'fake-g1'),(F2,'fake-g2'),(F1,'fake-g1'),(F2,'fake-g2')]):
    t=time.time(); iput('CHAT',[mid(P,m)]); s,j,ms,_=ichat('CHAT'); dt=time.time()-t
    rows.append((P['name'],j.get('provider'),j.get('model'),j.get('fallback_index'),round(dt,3)))
print(rows)
aud=ipsql("select provider||'/'||model||'/'||fallback_index from llm_audit where task='CHAT' order by created_at desc limit 6").split('\n')[::-1]
print('llm_audit 6 dòng cuối (cũ→mới):',aud)
ok12=all(r[0]==r[1] and r[4]<=1.0 for r in rows) and aud==[f'{r[1]}/{r[2]}/{r[3]}' for r in rows]
print('KQ12','P' if ok12 else 'F')
print('--- 15a: tắt nhà chính (F1) trong chuỗi [F1,F2]')
iput('CHAT',[mid(F1,'fake-g1'),mid(F2,'fake-g2')])
s0,j0,_,_=ichat('CHAT'); print('trước khi tắt: provider',j0.get('provider'),'fi',j0.get('fallback_index'))
print('tắt F1',iprov_set('SmkG-F1',enabled=False)[0]); time.sleep(1.5)
s,j,ms,_=ichat('CHAT'); print('(a) sau khi tắt: ',s,'provider',j.get('provider'),'fallback_index',j.get('fallback_index'))
pr=[p for p in ic('GET','/admin/llm/providers')[1]['items'] if p['name']=='SmkG-F1'][0]; print('   hàng F1 enabled=',pr['enabled'],'circuit=',pr.get('circuit'))
iprov_set('SmkG-F1',enabled=True); time.sleep(1.5)
print('--- 15b: nhà chính LỖI nhưng còn bật (stub 503), phụ F2')
V='http://host.docker.internal:9751/v1'; S3=imkprov('SmkG-S3',V,['m-503'],skip_verify=True)
iput('CHAT',[mid(S3,'m-503'),mid(F2,'fake-g2')],{'retries':0}); time.sleep(1.5)
s,j,ms,_=ichat('CHAT'); print('(b)',s,'provider',j.get('provider'),'fallback_index',j.get('fallback_index'))
print('   llm_audit',ipsql("select provider||'/fi='||fallback_index||'/'||status from llm_audit where task='CHAT' order by created_at desc limit 1"))
lg=[l for l in gwlogs('30s') if '"level":"WARN"' in l and ('from' in l and 'to' in l) and 'fallback' in l.lower()]
print('   log WARN fallback có from/to:',len(lg)); [print('    ',l[:300]) for l in lg[:2]]
print('--- 17: mọi nhà chết + passages')
iput('CHAT',[mid(S3,'m-503')],{'retries':0}); time.sleep(1.5)
ps=[{'text':f'Đoạn {i}: hạn nộp bài trước 23:59','source':'syllabus.pdf','page':i,'score':0.9-i/10} for i in range(5)]
s,j,ms,_=ichat('CHAT','hạn nộp bài?',passages=ps); print(s,'degraded',j.get('degraded'),ms,'ms'); print(repr(j.get('text')))
bad=[w for w in ['provider','fallback','trace','RAG','PII','LLM','prompt','embedding'] if w.lower() in (j.get('text') or '').lower()]
print('từ kỹ thuật:',bad,'| số đoạn trích:',sum(1 for i in range(5) if f'Đoạn {i}' in (j.get('text') or '')))
