from w7lib import *
import fx
fx.use('QC3S6C','QKTWQ3C'); Q=fx.QA
uidG=psql("select id from users where email='sv.gioi@edupilot.local'")
# ===== 51: retry đặt lại mọi cột
QT='Bùi Thanh Khải phân tích rủi ro hệ thống thông tin doanh nghiệp'; ask('sv.gioi',QT,course=Q)
sid=newsess('sv.gioi',Q); uid=uidG
psql(f"insert into chat_messages(course_id,session_id,user_id,role,content,stream_status) values('{Q}','{sid}','{uid}','USER','{QT}','DONE')")
um=psql(f"select id from chat_messages where session_id='{sid}' and role='USER'")
mid=psql(f"insert into chat_messages(course_id,session_id,user_id,role,content,stream_status,error_code,citations,blocks,confidence,low_confidence,no_context,degraded,masked_count,intent,feedback,attempt,partial_content,reply_to) values('{Q}','{sid}','{uid}','ASSISTANT','cũ','FAILED','PROVIDER_ERROR','[{{\"n\":1}}]','[{{\"kind\":\"x\"}}]',0.5,true,true,true,3,'COURSE_QA','HELPFUL',1,'phần cũ','{um}') returning id").splitlines()[0]
clr0=subprocess.run(['docker','exec','ep-r-redis-1','sh','-c',"redis-cli --scan --pattern 'ep:ans:*' | xargs -r redis-cli del"],capture_output=True); A('admin','POST','/_test/llm/fake',{'stream_delay_ms':800})
import threading
rr={}
th=threading.Thread(target=lambda: rr.setdefault('s',A('sv.gioi','POST',f'/chat/messages/{mid}/retry')[0])); th.start(); time.sleep(1.5)
row=psql(f"select content||'|'||coalesce(partial_content,'<null>')||'|'||(completed_at is null)||'|'||coalesce(error_code,'<null>')||'|'||citations::text||'|'||blocks::text||'|'||coalesce(confidence::text,'<null>')||'|'||low_confidence||no_context||degraded||'|'||masked_count||'|'||coalesce(intent,'<null>')||'|'||coalesce(feedback::text,'<null>')||'|'||attempt from chat_messages where id='{mid}'")
th.join(); s=rr['s']; A('admin','POST','/_test/llm/fake',{'stream_delay_ms':20})
f=row.split('|')
g=subprocess.run(GO+"go test -tags integration ./internal/chat -run 'TestReapInterrupted|TestReapSkipsLive|TestReapDoesNotOverwriteLive|TestRetryResetsAllColumns|TestDrainMarksInterruptedAndReleasesBusy|TestUpdatedAtTrigger' -v -count=1 2>&1 | grep -E '^(--- |ok|FAIL)'",shell=True,capture_output=True,text=True).stdout
okc=f[0]=='' and f[2]=='true' and f[3]=='<null>' and f[4]=='[]' and f[5]=='[]' and f[6]=='<null>' and f[7]=='falsefalsefalse' and f[8]=='0' and f[9]=='<null>' and f[10]=='<null>' and f[11]=='2'
rec('TC-P3-05-51',s in(200,202) and okc and 'FAIL' not in g,f'tin FAILED có citations/blocks/confidence/cờ/masked/intent/feedback → retry {s}; ngay sau (đang STREAMING chậm): {row} (content rỗng, completed_at NULL, error_code NULL, citations=[], blocks=[], confidence NULL, 3 cờ false, masked_count=0, intent NULL, feedback NULL, attempt 1→2); partial_content={f[1]!r} (đã bắt đầu lượt mới); go test: {g.replace(chr(10)," | ")[:200]}')
json.dump(R,open('/tmp/qc3e/w15.json','w'),ensure_ascii=False)
