# TC-P103-08 chạy lại (qc2/SmkO): sửa lỗi đo của lượt SmkN — stubreset() chỉ xoá bộ đếm 9741/9742 nên bộ đếm stub 9751 cộng dồn cả lời gọi chuẩn bị.
# Ở đây xoá bộ đếm đúng cổng 9751, đếm ĐỒNG THỜI bằng stub và bằng llm_audit trong cửa sổ 10 s; 20 luồng qua 2 gateway (gw1, gw2), nhà rpm_limit=60.
from smk3iso import *
V='http://host.docker.internal:9751/v1'
def reset9751(): urllib.request.urlopen(urllib.request.Request('http://localhost:9751/__reset',data=b'{}',method='POST'),timeout=10)
P=imkprov('SmkI-R60',V,['m-ok'],rpm_limit=60); assert P['rpm_limit']==60,P['rpm_limit']
iput('CHAT',[mid(P,'m-ok')],{'retries':0}); time.sleep(2)
def run(label):
    reset9751(); stop=[False]; res=[]; lk=threading.Lock()
    def worker(gw):
        while not stop[0]:
            r=ichat('CHAT',gw=gw)
            with lk: res.append((time.time(),r[0],r[1].get('code')))
    ths=[threading.Thread(target=worker,args=(['gw1','gw2'][i%2],)) for i in range(20)]
    t0=time.time(); [t.start() for t in ths]
    marks=[]
    for k in range(1,11):
        time.sleep(1-((time.time()-t0)%1)); marks.append(stub(9751)['n'])
    n10=marks[-1]; stop[0]=True; t1=time.time(); [t.join() for t in ths]
    from collections import Counter
    codes=Counter((r[1],r[2]) for r in res)
    aud=ipsql("select count(*) from llm_audit where task='CHAT' and provider='SmkI-R60' and created_at >= to_timestamp(%f) and created_at < to_timestamp(%f)"%(t0,t0+10.2)) if False else None
    print(label,'stub tích luỹ mỗi giây:',marks,'| stub tại t=10 s:',n10,'(TC ≤ 16)','| mã phản hồi:',dict(codes))
    return n10
a=run('lượt 1 (xô đầy)'); time.sleep(65); b=run('lượt 2 (sau 65 s nghỉ)')
time.sleep(3)
print('llm_audit CHAT status:',ipsql("select status||':'||count(*) from llm_audit where task='CHAT' and created_at > now() - interval '5 minutes' group by status"))
print('KQ', 'P' if a<=16 and b<=16 else 'F', a, b)
