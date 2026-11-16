# TC-P102-49 / TC-P103-46: /_test/llm/chat và /_test/llm/stats — binary thường 404 (ep-r), binary testroutes (gateway cô lập smk3-gw1, mini.sh): 401 không token, 403 SV/GV/TA, 200 ADMIN; keys stats không có prompt/text
from smk3iso import *
import lib
adm=itok('gw1'); 
def f(role): return lib.forge(adm,sec='edupilot-dev',role=role)
def eplogin():
    for i in range(6):
        subprocess.run(['docker','exec','ep-r-redis-1','sh','-c',"redis-cli --scan --pattern 'ep:rl:auth*' | xargs -r redis-cli del"],capture_output=True)
        try: return lib.login('admin')
        except Exception as e: time.sleep(3)
    raise
out=[]
for path,m,body in (('/_test/llm/chat','POST',{'task':'CHAT','prompt':'x'}),('/_test/llm/stats','GET',None)):
    r={}
    for name,tok in (('none',None),('STUDENT',f('STUDENT')),('TEACHER',f('TEACHER')),('TA',f('TA')),('ADMIN',adm)):
        s,j,_=lib._call(m,path,tok,body,None,None,True,IB['gw1']); r[name]=s
    # binary thường: ep-r (token ADMIN thật)
    s,j,_=lib._call(m,path,eplogin(),body)
    r['ep-r ADMIN (binary thường)']=s
    print(path,r); out.append((path,r))
s,j,_=ic('GET','/_test/llm/stats'); print('keys stats',sorted(j.keys()),'prompt/text trong stats:',any(k in json.dumps(j) for k in ('"prompt"','"text"')))
