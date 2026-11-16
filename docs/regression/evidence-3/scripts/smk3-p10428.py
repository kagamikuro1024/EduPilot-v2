# TC-P104-28 (+ TC-P101-30 qua API): đổi mô hình EMBEDDING → reindex_required:true; đổi tham số khác / giữ nguyên mô hình → false
from smk3iso import *
V='http://host.docker.internal:9751/v1'
body={'type':'openai_compatible','name':'SmkI-E','base_url':V,'api_key':'good-key','models':[
 {'model':'emb-a','kind':'embedding','dims':1536,'price_in':'1000','price_out':'0'},{'model':'emb-b','kind':'embedding','dims':1536,'price_in':'1000','price_out':'0'}]}
ex=[p for p in ic('GET','/admin/llm/providers')[1]['items'] if p['name']=='SmkI-E']
if ex: P=ex[0]
else:
    s,P,_=ic('POST','/admin/llm/providers',body); print('create',s)
a,b=mid(P,'emb-a'),mid(P,'emb-b')
r1=iput('EMBEDDING',[a]); print('đặt emb-a',r1[0],{k:v for k,v in r1[1].items() if k in('reindex_required','version')})
r2=iput('EMBEDDING',[b]); print('đổi sang emb-b',r2[0],{k:v for k,v in r2[1].items() if k in('reindex_required','version')})
r3=iput('EMBEDDING',[b],{'timeout_s':20}); print('giữ emb-b, đổi tham số',r3[0],{k:v for k,v in r3[1].items() if k in('reindex_required','version')})
r4=iput('EMBEDDING',[a]); print('đổi lại emb-a',r4[0],{k:v for k,v in r4[1].items() if k in('reindex_required','version')})
ok=r2[1].get('reindex_required') is True and r4[1].get('reindex_required') is True and r3[1].get('reindex_required') in (False,None)
print('KQ','P' if ok else 'F')
