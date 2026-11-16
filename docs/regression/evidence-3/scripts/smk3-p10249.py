# TC-P102-49 / TC-P103-46: /_test/llm/chat và /_test/llm/stats. binary testroutes = ep-r (và smk3-gw1): 401 không token, 403 SV/GV/TA, 200 ADMIN; binary thường = image edupilot-gateway (smk3-gw3 :18203): 404 kể cả ADMIN
from smk3iso import *
import lib
def eplogin(u):
    for i in range(8):
        subprocess.run(['docker','exec','ep-r-redis-1','sh','-c',"redis-cli --scan --pattern 'ep:rl:auth*' | xargs -r redis-cli del"],capture_output=True)
        try: return lib.login(u)
        except Exception as e: time.sleep(3)
    raise
toks={'none':None,'STUDENT':eplogin('sv36'),'TEACHER':eplogin('teacher'),'TA':eplogin('ta'),'ADMIN':eplogin('admin')}
for path,m,body in (('/_test/llm/chat','POST',{'task':'CHAT','prompt':'x'}),('/_test/llm/stats','GET',None)):
    r={n:lib._call(m,path,t,body)[0] for n,t in toks.items()}
    s3,_,_=lib._call(m,path,itok('gw3') if False else None,body,None,None,True,IB['gw3'])
    s,j,_=lib._call('POST','/auth/login',body={'email':'admin@smk3.test','password':APW},base=IB['gw3']); a3=j.get('access_token')
    r3=lib._call(m,path,a3,body,None,None,True,IB['gw3'])[0]
    print(path,'testroutes (ep-r):',r,'| binary thường ADMIN:',r3,'| không token:',s3)
s,j,_=lib._call('GET','/_test/llm/stats',toks['ADMIN']); print('keys stats',sorted(j.keys()),'prompt/text:',any(k in json.dumps(j) for k in ('"prompt"','"text"')))
