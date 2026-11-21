import subprocess,time,json,datetime,os,sys
HERE=os.path.dirname(os.path.abspath(__file__)); sys.path.insert(0,'/tmp/qc3e')
REPO=os.path.expanduser('~/Documents/TA_Agent_qc3p'); OUT=HERE+'/out'; os.makedirs(OUT,exist_ok=True)
def log(m): open('/tmp/qc2-global-changes.log','a').write(f"{datetime.datetime.now():%H:%M:%S} qc3: {m}\n")
sh=lambda c,**k: subprocess.run(c,shell=True,capture_output=True,text=True,**k)
print(sh('python3 /tmp/qc3e/win.py on').stdout[-60:],flush=True); t00=time.time(); log('k6: bắt đầu (trong khoá của win.py)')
try:
    for name,scen,lat in (('first_event','first_event','5000-15000'),('ttft','ttft','300-300'),('ttft_bad','ttft','3000-3000')):
        print(sh(f'{HERE}/k6gw.sh up {lat}').stdout.strip(),flush=True); t=time.time()
        r=sh(f'k6 run -q {HERE}/chat_qc3.js -e BASE=http://localhost:18096 -e SCENARIO={scen}',cwd=REPO)
        open(f'{OUT}/{name}.txt','w').write(f'rc={r.returncode}\n'+r.stdout+r.stderr); print(name,r.returncode,round(time.time()-t),flush=True)
finally:
    sh(f'{HERE}/k6gw.sh down'); print(sh('python3 /tmp/qc3e/win.py off').stdout[-60:]); log(f'k6 xong sau {round(time.time()-t00)} s')
