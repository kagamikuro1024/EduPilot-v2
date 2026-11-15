import sys, time, json, subprocess, concurrent.futures as cf, os
sys.path.insert(0,'.'); import importlib.util
sp=importlib.util.spec_from_file_location('p','p502-attacks.py'); p=importlib.util.module_from_spec(sp); sp.loader.exec_module(p)
# A15 write then read
for n in ['a15-leak-write.c','a15-leak-read.c']:
    c,_=p.compile_(open('attacks/'+n).read(),False); r,_=p.run(c['fileIds']['a']); print(n,r['status'],repr(r['files']['stdout'][:80])); p.rm(c['fileIds']['a'])
# token
def curl(extra,path,m='GET'):
    r=subprocess.run(['docker','exec','qc5-cli','curl','-s','-o','/dev/null','-w','%{http_code}','-X',m]+extra+['-d','{}','-H','Content-Type: application/json','http://qc5-judge:5050'+path],capture_output=True,text=True); return r.stdout
print('TC-07: /run no token',curl([],'/run','POST'),'wrong',curl(['-H','Authorization: Bearer wrong'],'/run','POST'),'good',curl(['-H','Authorization: Bearer '+p.TOK],'/run','POST'),'| /version no token',curl([],'/version'))
# TC-44: correct program in parallel with attacks
OK='#include <stdio.h>\nint main(){int a,b; if(scanf("%d %d",&a,&b)==2) printf("%d\\n",a+b); return 0;}\n'
c,_=p.compile_(OK,False); fid=c['fileIds']['a']
atk=[]
for n in ['a01-infinite-loop.c','a02-print-forever.c','a05-alloc-gradual-512m.c','a06-alloc-2g.c','a08-fork-bomb.c','a19-sleep-forever.c','a20-stdin-hang.c','a04-print-1gb.c']:
    cc,_=p.compile_(open('attacks/'+n).read(),False); atk.append(cc['fileIds']['a'])
def good(i):
    t=time.time(); r,_=p.run(fid,'2 3\n'); return (r['status'],r['files']['stdout'].strip(),time.time()-t)
def bad(f): return p.run(f)[0]['status']
with cf.ThreadPoolExecutor(12) as ex:
    fb=[ex.submit(bad,f) for f in atk*2]; fg=[ex.submit(good,i) for i in range(10)]
    g=[f.result() for f in fg]; b=[f.result() for f in fb]
print('TC-44: bài đúng song song với 16 ca tấn công:', [x[:2] for x in g][:3], '… tất cả AC:', all(x[0]=='Accepted' and x[1]=='5' for x in g), 'max chờ %.1fs'%max(x[2] for x in g))
print('       ca tấn công:', sorted(set(b)))
# TC-46: 60 bài bits/stdc++ × 10 test
SRC='#include <bits/stdc++.h>\nusing namespace std;\nint main(){int a,b; cin>>a>>b; cout<<a+b<<"\\n";}\n'
def submission(i):
    t0=time.time(); c,_=p.compile_(SRC+'//%d\n'%i,True)
    if c['status']!='Accepted': return ('CE',time.time()-t0)
    f=c['fileIds']['a']; ok=True
    for k in range(10):
        r,_=p.run(f,'%d %d\n'%(k,k+1)); ok&= r['status']=='Accepted' and r['files']['stdout'].strip()==str(2*k+1)
    p.rm(f); return ('AC' if ok else 'BAD',time.time()-t0)
t=time.time()
with cf.ThreadPoolExecutor(60) as ex: res=list(ex.map(submission,range(60)))
tot=time.time()-t; d=sorted(x[1] for x in res)
print('TC-46: 60 bài × (1 biên dịch + 10 test), parallelism=2: tổng %.1fs, p95 %.1fs, AC=%d, khác=%d'%(tot,d[int(len(d)*.95)-1],sum(1 for x in res if x[0]=='AC'),sum(1 for x in res if x[0]!='AC')))
