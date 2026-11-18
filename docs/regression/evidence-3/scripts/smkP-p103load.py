#!/usr/bin/env python3
"""smkP: đo độc lập Scheduler (TC-P103-11/12/50, GATEP1-16): stack cô lập smk3-gw1 :18201 (fake 5-15 s, LLM_MAX_CONCURRENCY=10, BATCH_SHARE 0,5).
Pha 0 mốc: 25 CHAT (≤5 đồng thời) không BATCH, đo TTFT (stream: tới event token đầu) + queue_wait. Pha A (TC-12): chỉ 200 BATCH, lấy mẫu inflight_batch mỗi 25 ms; rồi 10 CHAT tuần tự sau khi BATCH chạy hàng loạt. Pha B (TC-11): 200 BATCH + 25 CHAT (≤5 đồng thời).
Số tự đo ghi ra JSON /tmp/smkP-p103.json (rồi chép vào evidence-3/logs/)."""
import json, time, threading, http.client, urllib.request, statistics, sys
H='localhost'; P=18201; PW='Zx9#Quartz-Lantern-2026'
def tok():
    r=urllib.request.Request(f'http://{H}:{P}/api/v1/auth/login',data=json.dumps({'email':'admin@smk3.test','password':PW}).encode(),headers={'content-type':'application/json','Idempotency-Key':str(time.time_ns())})
    return json.loads(urllib.request.urlopen(r).read())['access_token']
T=tok(); HD={'Authorization':'Bearer '+T,'Content-Type':'application/json'}
def post(task,prompt,stream=False):
    c=http.client.HTTPConnection(H,P,timeout=900); t0=time.time(); c.request('POST','/api/v1/_test/llm/chat',json.dumps({'task':task,'prompt':prompt,'stream':stream}),HD); r=c.getresponse()
    if not stream:
        b=r.read(); c.close(); j={}
        try: j=json.loads(b)
        except Exception: pass
        return r.status,j.get('queue_wait_ms'),None,time.time()-t0
    ttft=None; qw=None; ev=None
    for line in r:
        line=line.decode().strip()
        if line.startswith('event:'): ev=line[6:].strip()
        elif line.startswith('data:'):
            if ev=='token' and ttft is None: ttft=(time.time()-t0)*1000
            if ev=='done':
                try: qw=json.loads(line[5:]).get('queue_wait_ms')
                except Exception: pass
    c.close(); return r.status,qw,ttft,time.time()-t0
def inflight():
    c=http.client.HTTPConnection(H,P,timeout=10); c.request('GET','/api/v1/_test/llm/stats?lanes=1',headers=HD); r=c.getresponse(); j=json.loads(r.read()); c.close(); return sum((j.get('inflight_batch') or {}).values())
def p95(a): a=sorted(a); return a[max(int(len(a)*0.95)-1,0)] if a else None
class Sampler(threading.Thread):
    def __init__(s): super().__init__(daemon=True); s.stop=False; s.s=[]
    def run(s):
        while not s.stop:
            try: s.s.append((time.time(),inflight()))
            except Exception: pass
            time.sleep(0.025)
def chats(n,conc,gap=0.1,stream=True):
    out=[]; lk=threading.Lock(); sem=threading.Semaphore(conc); ths=[]; marks=[]
    def one(i):
        with sem:
            marks.append(time.time()); r=post('CHAT',f'hoi {i}',stream); marks.append(time.time())
            with lk: out.append(r)
    for i in range(n):
        t=threading.Thread(target=one,args=(i,)); t.start(); ths.append(t); time.sleep(gap)
    [t.join() for t in ths]; return out,marks
def summ(rs):
    ok=[r for r in rs if r[0]==200]; return {'n':len(rs),'ok':len(ok),'status':sorted({r[0] for r in rs}),'wait_p95_ms':p95([r[1] for r in ok if r[1] is not None]),'wait_max_ms':max([r[1] or 0 for r in ok] or [0]),'ttft_p50_ms':round(statistics.median([r[2] for r in ok if r[2]]),0) if any(r[2] for r in ok) else None,'ttft_p95_ms':round(p95([r[2] for r in ok if r[2]]) or 0,0)}
R={}
print('pha 0 (mốc)',flush=True); base,_=chats(25,5); R['baseline']=summ(base); print(R['baseline'],flush=True)
def batch(n):
    ths=[]; res=[]
    def one(i): res.append(post('INSIGHT',f'batch {i}')[0])
    for i in range(n): t=threading.Thread(target=one,args=(i,)); t.start(); ths.append(t)
    return ths,res
print('pha A (TC-12: chỉ BATCH)',flush=True); sp=Sampler(); sp.start(); ths,res=batch(200); time.sleep(8)
first,_=chats(10,1,gap=0.5); R['A_first_chats_after_batch_running']=summ(first); [t.join() for t in ths]; sp.stop=True; time.sleep(0.2)
pk=max(v for _,v in sp.s) if sp.s else None; R['A_batch_peak']=pk; R['A_samples']=len(sp.s); R['A_batch_status']={str(k):res.count(k) for k in set(res)}; print({k:R[k] for k in R if k.startswith('A_')},flush=True)
time.sleep(30)
print('pha B (TC-11: 200 BATCH + 25 CHAT ≤5 đồng thời)',flush=True); sp=Sampler(); sp.start(); ths,res=batch(200); time.sleep(0.5); mix,marks=chats(25,5); lo,hi=min(marks)-2,max(marks)+2
[t.join() for t in ths]; sp.stop=True; time.sleep(0.2)
R['B_mixed']=summ(mix); R['B_batch_peak_all']=max(v for _,v in sp.s); R['B_batch_peak_when_interactive']=max([v for t,v in sp.s if lo<=t<=hi] or [0]); R['B_batch_status']={str(k):res.count(k) for k in set(res)}
b=R['baseline']; m=R['B_mixed']; R['B_wait_p95_delta_ms']=(m['wait_p95_ms'] or 0)-(b['wait_p95_ms'] or 0); R['B_ttft_p95_ratio']=round(m['ttft_p95_ms']/b['ttft_p95_ms'],3) if b['ttft_p95_ms'] else None; R['B_ttft_p50_ratio']=round(m['ttft_p50_ms']/b['ttft_p50_ms'],3) if b['ttft_p50_ms'] else None
print(json.dumps(R,ensure_ascii=False,indent=1)); open('/tmp/smkP-p103.json','w').write(json.dumps(R,ensure_ascii=False,indent=1))
