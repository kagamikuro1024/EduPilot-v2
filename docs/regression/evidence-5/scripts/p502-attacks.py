#!/usr/bin/env python3
"""QC TC-PE02-33…45: chạy 20 mã tấn công của QC (attacks/*.c|cpp) thẳng vào go-judge (chỉ biên dịch + chạy theo SRS 4.5.2),
độc lập với mã `internal/judge` của dev. Cần: container `qc5-judge` (ảnh deploy/judge, mạng internal `qc5_judge_net`) + `qc5-cli` (curl) cùng mạng.
Dùng: python3 p502-attacks.py <thư-mục-attacks> [tên-lọc]"""
import json, os, subprocess, sys, time, glob
TOK = open('/tmp/qc5-jtok').read().strip()
def post(path, body, method='POST', t=60):
    r = subprocess.run(['docker','exec','-i','qc5-cli','curl','-s','-m',str(t),'-X',method,'-H','Authorization: Bearer '+TOK,'-H','Content-Type: application/json','-d','@-','http://qc5-judge:5050'+path],input=json.dumps(body) if body is not None else '',capture_output=True,text=True)
    try: return json.loads(r.stdout)
    except Exception: return {'_raw': r.stdout[:200], '_err': r.stderr[:200]}
def compile_(src, cpp):
    name, args = ('a.cc',['/usr/bin/g++','-O2','-std=c++17','-pipe','-o','a','a.cc']) if cpp else ('a.c',['/usr/bin/gcc','-O2','-std=c11','-pipe','-o','a','a.c','-lm'])
    t=time.time()
    r = post('/run',{'cmd':[{'args':args,'env':['PATH=/usr/bin:/bin'],'files':[{'content':''},{'name':'stdout','max':4096},{'name':'stderr','max':8192}],'cpuLimit':10*10**9,'clockLimit':20*10**9,'memoryLimit':512<<20,'procLimit':50,'copyIn':{name:{'content':src}},'copyOutCached':['a']}]})
    return r[0] if isinstance(r,list) else r, time.time()-t
def run(fid, inp='', cpu_ms=1000, mem_mb=256, out_kb=1024):
    cpu=cpu_ms*10**6
    t=time.time()
    r = post('/run',{'cmd':[{'args':['a'],'env':['PATH=/usr/bin:/bin'],'files':[{'content':inp},{'name':'stdout','max':out_kb*1024},{'name':'stderr','max':4096}],'cpuLimit':cpu,'clockLimit':3*cpu,'memoryLimit':mem_mb<<20,'procLimit':1,'copyIn':{'a':{'fileId':fid}}}]},t=40)
    return r[0] if isinstance(r,list) else r, time.time()-t
def rm(fid): post('/file/'+fid,None,'DELETE')
MAP={'Accepted':'AC','Time Limit Exceeded':'TLE','Memory Limit Exceeded':'MLE','Output Limit Exceeded':'OLE','Nonzero Exit Status':'RE','Signalled':'RE','Internal Error':'IE','File Error':'IE','Dangerous Syscall':'RE'}
if __name__=='__main__':
    d=sys.argv[1]; flt=sys.argv[2] if len(sys.argv)>2 else ''
    for f in sorted(glob.glob(d+'/a*.c')+glob.glob(d+'/a*.cpp')):
        n=os.path.basename(f)
        if flt and flt not in n: continue
        src=open(f).read(); c,ct=compile_(src,f.endswith('.cpp'))
        if c.get('status')!='Accepted':
            print(f'{n:28s} CE  ({c.get("status")}) {ct:5.1f}s  {(c.get("files",{}).get("stderr") or c.get("error") or "")[:60]!r}'); continue
        fid=c['fileIds']['a']
        r,rt=run(fid)
        out=(r.get('files',{}).get('stdout') or '')[:70].replace('\n','|')
        print(f'{n:28s} {MAP.get(r.get("status"),r.get("status")):4s} ({r.get("status")}) cpu={r.get("time",0)//10**6}ms mem={r.get("memory",0)>>20}MiB proc={r.get("procPeak")} wall={rt:4.1f}s out={out!r} err={r.get("error","")[:50]!r}')
        rm(fid)
