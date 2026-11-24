import sys; sys.path.insert(0,'/tmp/qc3oc')
from routes import *
k=open(os.path.expanduser('~/.config/edupilot-team/secrets/opencode-go.key')).read().strip()
# TC-57: tạo QC3-OC (có kiểm tra) và Test ×10
t0=time.time(); s,d=A('admin','POST','/admin/llm/providers',{"type":"openai_compatible","name":"QC3-OC","base_url":"https://opencode.ai/inference/go/v1","api_key":k,"models":[{"model":"deepseek-v4.1-flash","kind":"chat","price_in":"0","price_out":"0"}]},timeout=100)
say('TC57 POST QC3-OC (có kiểm tra):',s,round(time.time()-t0,1),'s',(d or {}).get('code') if s!=201 else 'id '+d['id'][:8])
pid=psql("select id from llm_providers where name='QC3-OC'"); res=[]
for i in range(10):
    s,d=A('admin','POST',f'/admin/llm/providers/{pid}/test',{},timeout=100); res.append((d['ok'],d.get('error_kind'),d['latency_ms']))
say('TC57 Test ×10:',res)
say('TC57 TIMEOUT 10000?',any(r[1]=='TIMEOUT' and r[2]<=10050 for r in res),'| max latency',max(r[2] for r in res))
# TC-62: nhà chậm 20 s
s,d=A('admin','POST','/admin/llm/providers',{"type":"openai_compatible","name":"QC3-stub-ok","base_url":"http://host.docker.internal:9791/ok/v1","api_key":"stubkey","models":[{"model":"m1","kind":"chat","price_in":"0","price_out":"0"}]}); say('create stub-ok',s)
for n,mode,skip in (('QC3-stub-bad','bad',True),):
    s,d=A('admin','POST','/admin/llm/providers',{"type":"openai_compatible","name":n,"base_url":f"http://host.docker.internal:9791/{mode}/v1","api_key":"stubkey","skip_verify":skip,"models":[{"model":"mb","kind":"chat","price_in":"0","price_out":"0"}]}); say('create',n,s)
t0=time.time(); s,d=A('admin','POST','/admin/llm/providers',{"type":"openai_compatible","name":"QC3-stub-slow","base_url":"http://host.docker.internal:9791/slow/v1","api_key":"stubkey","models":[{"model":"ms","kind":"chat","price_in":"0","price_out":"0"}]},timeout=100); dt=round(time.time()-t0,1)
say('TC62 POST QC3-stub-slow (có kiểm tra, upstream 20 s):',s,dt,'s',(d or {}).get('code') if s!=201 else ('last_test='+str(d.get('last_test'))+' version '+str(d['version'])))
sid=psql("select id from llm_providers where name='QC3-stub-slow'"); say('  bản ghi:',psql("select count(*) from llm_providers where name='QC3-stub-slow'"))
if s==201:
    v=d['version']; t0=time.time(); s2,d2=A('admin','PUT',f'/admin/llm/providers/{sid}',{"type":"openai_compatible","name":"QC3-stub-slow","base_url":"http://host.docker.internal:9791/slow/v2","version":v},timeout=100)
    say('TC62 PUT đổi base_url (kiểm tra lại, upstream 20 s):',s2,round(time.time()-t0,1),'s',(d2 or {}).get('code') if s2!=200 else 'version '+str(d2['version'])+' last_test '+str(d2.get('last_test')))
    t0=time.time(); s3,d3=A('admin','POST',f'/admin/llm/providers/{sid}/test',{},timeout=100); say('TC62 Test API nhà chậm:',s3,d3,round(time.time()-t0,1),'s')
