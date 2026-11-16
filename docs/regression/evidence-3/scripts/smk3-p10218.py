from smk3iso import *
V1='http://host.docker.internal:9751/v1'; V2='http://host.docker.internal:9752/v1'
SM=['m-ok','m-400','m-401','m-big','m-canary','m-ra2','m-ra30','m-slow10','m-503','m-500','m-midkill','m-prefail','m-slow']
T1=imkprov('SmkI-T1',V1,SM); T2=imkprov('SmkI-T2',V2,SM)
def sreset():
    for p in (9751,9752): urllib.request.urlopen(urllib.request.Request(f'http://localhost:{p}/__reset',data=b'{}',method='POST'),timeout=10)
for fb in ('m-ok','m-503'):
    print(iput('GRADING',[mid(T1,'m-400'),mid(T2,fb)],{'retries':2})[0]); sreset()
    r=ichat('GRADING',hdr=tp('c1'+'0'*30)); time.sleep(1)
    print('A=m-400, B=',fb,r[0],r[1].get('code'),'stubA',stub(9751),'stubB',stub(9752))
for kind in ('m-401',):
    iput('GRADING',[mid(T1,kind),mid(T2,'m-ok')],{'retries':2}); sreset(); r=ichat('GRADING'); print('A=',kind,r[0],r[1].get('code'),stub(9751)['log'],stub(9752)['log'])
