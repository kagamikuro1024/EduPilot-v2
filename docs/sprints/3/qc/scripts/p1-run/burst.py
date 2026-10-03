import sys,json,urllib.request,threading,time
port,n,workers=int(sys.argv[1]),int(sys.argv[2]),int(sys.argv[3]); tok=open('/tmp/qcp1/tok-ADMIN').read().strip()
ok=[0];bad=[0];lock=threading.Lock(); it=iter(range(n))
def w():
    while True:
        with lock:
            try: next(it)
            except StopIteration: return
        r=urllib.request.Request(f'http://localhost:{port}/api/v1/_test/llm/chat',data=b'{"task":"CLASSIFY","prompt":"x"}',headers={'Authorization':'Bearer '+tok,'Content-Type':'application/json'})
        try: urllib.request.urlopen(r,timeout=20); ok[0]+=1
        except Exception: bad[0]+=1
ts=[threading.Thread(target=w) for _ in range(workers)]; t=time.time(); [x.start() for x in ts]; [x.join() for x in ts]; print('ok',ok[0],'bad',bad[0],'in %.1fs'%(time.time()-t))
