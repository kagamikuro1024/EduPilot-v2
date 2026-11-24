import sys,os; sys.path.insert(0,'/tmp/qc3oc')
from routes import *
k=open(os.path.expanduser('~/.config/edupilot-team/secrets/opencode-go.key')).read().strip()
P=provs()
def mk(name,base,key,model,skip=True):
    if name in P: return say('exists',name)
    s,d=A('admin','POST','/admin/llm/providers',{"type":"openai_compatible","name":name,"base_url":base,"api_key":key,"skip_verify":skip,"models":[{"model":model,"kind":"chat","price_in":"0","price_out":"0"}]})
    say('create',name,s,(d or {}).get('code') if s!=201 else d['id'])
mk('QC3-stub-ok','http://host.docker.internal:9791/ok/v1','stubkey','m1',skip=False)
mk('QC3-stub-404','http://host.docker.internal:9791/r404/v1','stubkey','m404')
mk('QC3-stub-403','http://host.docker.internal:9791/r403/v1','stubkey','m403')
k2=k
mk('QC3-badmodel','https://opencode.ai/inference/go/v1',k2,'deepseek-v9-khong-co')
