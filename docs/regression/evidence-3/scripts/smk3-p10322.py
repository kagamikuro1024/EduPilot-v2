from smk3iso import *
V2='http://host.docker.internal:9752/v1'
T2=imkprov('SmkI-T2',V2,['m-ok','m-503','m-500'])
print(iput('GRADING',[mid(T2,'m-503')],{'retries':0})[0])
urllib.request.urlopen(urllib.request.Request('http://localhost:9752/__reset',data=b'{}',method='POST'))
for i in range(6): r=ichat('GRADING',gw='gw1'); print(i,r[0],r[1].get('code'),r[2],'ms')
time.sleep(1); pid=T2['id']
print('khoá Redis:',ird('GET','ep:llm:cb:'+pid)[:200],'| keys',ird('--scan','--pattern','ep:llm:cb:*').split())
print('providers circuit gw2:',[(p['name'],p['circuit']) for p in ic('GET','/admin/llm/providers',gw='gw2')[1]['items'] if p['name']=='SmkI-T2'])
n0=stub(9752)['n']; r=ichat('GRADING',gw='gw2'); print('gọi qua gw2:',r[0],r[1].get('code'),r[2],'ms; stub nhận thêm',stub(9752)['n']-n0)
print(ipsql("select status,error_kind,attempts from llm_audit where task='GRADING' order by created_at desc limit 2"))
