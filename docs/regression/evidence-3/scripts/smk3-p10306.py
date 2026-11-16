from smk3iso import *
V1='http://host.docker.internal:9751/v1'
T3=imkprov('SmkI-T3',V1,['m-ok','m-slow10','m-hang','m-slow'])
iput('CHAT',[mid(T3,'m-hang')],{'retries':0}); iput('GRADING',[mid(T3,'m-ok')],{'retries':0}); time.sleep(1.5)
key='ep:llm:inflight:'+T3['id']; r=RC()
print('ZCARD trước',r.cmd('ZCARD',key))
def hold(i):
    try: ichat('CHAT',f'h{i}',gw='gw2')
    except Exception as e: pass
ths=[threading.Thread(target=hold,args=(i,),daemon=True) for i in range(3)]; [t.start() for t in ths]; time.sleep(2.5)
print('ZCARD khi gw2 giữ 3 chỗ',r.cmd('ZCARD',key),r.cmd('ZRANGE',key,'0','-1','WITHSCORES'))
t_kill=time.time(); subprocess.run(['docker','kill','smk3-gw2'],capture_output=True)
g=ichat('GRADING',gw='gw1'); print('gw1 phục vụ GRADING cùng nhà khi 3 chỗ treo:',g[0],g[1].get('provider'),g[2],'ms')
last=None; t_zero=None
while time.time()-t_kill<80:
    n=r.cmd('ZCARD',key)
    if n!=last: print('  t+%.1fs ZCARD=%s'%(time.time()-t_kill,n)); last=n
    if n==0: t_zero=time.time()-t_kill; break
    time.sleep(1)
print('thu hồi chỗ sau',t_zero)
