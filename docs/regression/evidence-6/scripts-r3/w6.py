from w3lib import *
import fx,subprocess
sh=lambda c,t=900: subprocess.run(c,shell=True,capture_output=True,text=True,timeout=t)
def gt(pkg,run,tags=''):
    o=sh(GO+f"go test {tags} {pkg} -run '{run}' -v -count=1 2>&1 | grep -E '^(--- |ok|FAIL|PASS)' | head -20").stdout; return o
# ---- Go-level: 02-41, 03-09, 02-50, 03-33
o=gt('./internal/privacy','TestQCStreamCutEverywhere|TestUnmaskStream$'); rec('TC-P3-02-41','FAIL' not in o and 'TestQCStreamCutEverywhere' in o and 'PASS' in o,'cắt chuỗi "Xin chào [[SV_1]], điểm của bạn…" tại MỌI vị trí rune (TestQCStreamCutEverywhere: ghép == Unmask(toàn), 0 khung chứa nửa placeholder): '+o.replace('\n',' | ')[:200])
o=gt('./internal/llm','TestStructuredUnmaskAfterSchema|TestMaskCoversAllPayloadParts'); rec('TC-P3-03-09','FAIL' not in o and o.count('--- PASS')>=2,'đường Structured ở sprint 6 (Threads UTILITY, chấm bài): gọi qua cổng llm.Structured: '+o.replace('\n',' | ')[:200])
o=gt('./internal/llm','TestMaskNoCourseFailsClosed|TestMaskOnceBeforeFallback|TestAuditPIIMaskedCount'); 
rec('TC-P3-02-50-go','FAIL' not in o,o.replace('\n',' | ')[:200])
b=sh(GO+"go test ./internal/llm -bench 'BenchmarkGatewayMask' -run '^$' -benchtime=200x 2>&1 | grep -E 'Benchmark|ok|FAIL'").stdout
import re
ns=[float(x) for x in re.findall(r'([\d.]+) ns/op',b)]
rec('TC-P3-03-33',bool(ns) and all(x/1e6<=5 for x in ns),'go test -bench BenchmarkGatewayMask -benchtime=200x: '+b.replace('\n',' | ')[:300]+f' (≤ 5 ms/op: {[round(x/1e6,3) for x in ns]} ms)')
# ---- 03-12 / 03-13: hai provider fake, fake-1 lỗi (khoá sai), fake-2 trả lời
def mkprov(name,key):
    s,d=A('admin','POST','/admin/llm/providers',{'type':'fake','name':name,'api_key':key,'models':[{'model':'fake-chat','kind':'chat','price_in':'0','price_out':'0'}]})
    assert s in(200,201),(s,d); return d['id'],d['models'][0]['id'],d
p1,m1,_=mkprov('QC3-fake1','bad-key'); p2,m2,_=mkprov('QC3-fake2','good-key')
try:
    A('admin','POST','/_test/llm/fake',{'valid_key':'good-key'})
    rt={r['task']:r for r in A('admin','GET','/admin/llm/routes')[1]['items']}
    s,d=A('admin','PUT','/admin/llm/routes',{'task':'CHAT','chain':[m1,m2],'version':rt['CHAT']['version']}); print('route',s)
    msg=f'{KHAI}, {GIANG} 20229002 sv.kha@edupilot.local giải thích RSA'
    reset(); s,d=A('admin','POST','/_test/llm/chat',{'task':'CHAT','prompt':msg,'course_id':C1,'session_id':'qc3-312'}); its=pay()
    chat_its=[i for i in its if i['task']=='CHAT']
    same=len(chat_its)==2 and chat_its[0]['messages']==chat_its[1]['messages']
    ok=s==200 and same and not any(x in flat(chat_its) for x in (KHAI,GIANG,'20229002','sv.kha@')) and d.get('fallback_index')==1
    rec('TC-P3-03-12',ok,f'fake-1 (khoá sai→AUTH) rồi fake-2: HTTP {s}, fallback_index={d.get("fallback_index")}, {len(chat_its)} payload, hai payload giống hệt={same}, payload={chat_its[0]["messages"] if chat_its else None}')
    au=psql("select pii_masked_count||'|'||attempts||'|'||fallback_index||'|'||status from llm_audit where task='CHAT' and provider like 'QC3-fake%' order by created_at desc limit 3")
    rec('TC-P3-03-13',au.splitlines()[0].startswith('4|') or au.splitlines()[0].split('|')[0]=='4',f'4 thực thể (2 tên+MSSV+email); llm_audit (pii_masked_count|attempts|fallback_index|status) = {au.splitlines()[:2]} → không nhân đôi theo số lần thử')
finally:
    A('admin','POST','/_test/llm/fake',{'valid_key':''})
    json.dump(R,open('/tmp/qc3e/w6.json','w'),ensure_ascii=False)
