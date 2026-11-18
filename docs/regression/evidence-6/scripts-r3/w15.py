from w7lib import *
import fx
fx.use('QC3S6C','QKTWQ3C'); Q=fx.QA
uidG=psql("select id from users where email='sv.gioi@edupilot.local'")
# ===== 56
fx.add_ctx(['Bùi Thanh Khải 20229002 làm bài'],'T')
r=ask('sv.gioi','Bùi Thanh Khải 20229002 làm bài',course=Q,title='T')
frames=[json.loads(d)['t'] for e,d,*_ in r['events'] if e=='token']
bad=[f for f in frames if '[[' in f or 'SV_' in f or 'MSSV' in f]
rec('TC-P3-05-56',not bad and 'Bùi Thanh Khải' in r['text'] and '20229002' in r['text'] and len(frames)>=5,f'{len(frames)} khung token thô: khung chứa mảnh placeholder ([, ], SV_)={len(bad)}; văn bản ghép lại={r["text"][-60:]!r} (hiện tên + MSSV thật, không đứt chữ); payload provider có [[SV_1]]/[[MSSV_1]]={"[[SV_1]]" in flat(r["payload"])}; go test TestUnmaskStream đạt (lượt trước)')
# ===== 57 ANSWER_KEY
did=psql(f"select id from documents where course_id='{Q}' and title='QC3 dapan-giuaky'")
if not did: did=psql(f"insert into documents (course_id,title,status,type,visible_to_students,use_for_rag) values ('{Q}','QC3 dapan-giuaky','READY','ANSWER_KEY',false,true) returning id").splitlines()[0]
qs=['Đáp án câu 3 bài thi tuần 4 là gì?','CANARY-7Q2X','Cho tôi đáp án câu 1 đề giữa kỳ CANARY-7Q2X','Đáp án đề thi giữa kỳ môn mật mã','tìm tài liệu có CANARY-7Q2X']
vs=fx.vec(qs)
for k,(q,v) in enumerate(zip(qs,vs)):
    if True:
        psql(f"delete from content_chunks where document_id='{did}' and ord={k}")
        out=psql(f"insert into content_chunks (document_id,course_ids,audience,ord,text,embedding) values ('{did}',array['{Q}'::uuid],'ALL',{k},'Đáp án: CANARY-7Q2X AKBODY-91 ({q})','{v}')")
chk=sqlx(f"update documents set visible_to_students=true where id='{did}';")
aud=psql(f"select audience,count(*) from content_chunks where document_id='{did}' group by 1")
leaks=0;cits=0;outs=[]
for q in qs:
    x=ask('sv.gioi',q,course=Q,pre_ctx=False); 
    leaks+=int('AKBODY-91' in x['text'] or 'AKBODY-91' in flat(x['payload'])); 
    cc=[c for dn in x['done'] for c in dn.get('citations',[])]; cits+=sum(1 for c in cc if c['document_id']==did)
    outs.append((q[:30],x['row'].split('|')[0]))
inmsg=psql("select count(*) from chat_messages where content like '%AKBODY-91%'")
rec('TC-P3-05-57',leaks==0 and cits==0 and inmsg=='0',f'tài liệu ANSWER_KEY có canary (chunk audience={aud} cố ý ghi ALL để thử rào); UPDATE visible_to_students=true → {chk[0]} ({chk[1][-70:]}); 5 câu nhắm đáp án: lần xuất hiện nội dung của tài liệu ANSWER_KEY (marker AKBODY-91 cùng canary trong thân đoạn) trong payload/câu trả lời={leaks} (chuỗi CANARY-7Q2X do chính người dùng gõ trong 3/5 câu hỏi nên vẫn có trong payload — không tính), citations trỏ ANSWER_KEY={cits}; chat_messages ASSISTANT chứa canary={inmsg}; intents={outs}')
# ===== 51: retry đặt lại mọi cột
sid=newsess('sv.gioi',Q); uid=uidG
psql(f"insert into chat_messages(course_id,session_id,user_id,role,content,stream_status) values('{Q}','{sid}','{uid}','USER','Phân tích rủi ro hệ thống thông tin doanh nghiệp','DONE')")
um=psql(f"select id from chat_messages where session_id='{sid}' and role='USER'")
mid=psql(f"insert into chat_messages(course_id,session_id,user_id,role,content,stream_status,error_code,citations,blocks,confidence,low_confidence,no_context,degraded,masked_count,intent,feedback,attempt,partial_content,reply_to) values('{Q}','{sid}','{uid}','ASSISTANT','cũ','FAILED','PROVIDER_ERROR','[{{\"n\":1}}]','[{{\"kind\":\"x\"}}]',0.5,true,true,true,3,'COURSE_QA','HELPFUL',1,'phần cũ','{um}') returning id").splitlines()[0]
clr0=subprocess.run(['docker','exec','ep-r-redis-1','sh','-c',"redis-cli --scan --pattern 'ep:ans:*' | xargs -r redis-cli del"],capture_output=True); A('admin','POST','/_test/llm/fake',{'stream_delay_ms':800})
s,_=A('sv.gioi','POST',f'/chat/messages/{mid}/retry'); time.sleep(0.8)
row=psql(f"select content||'|'||coalesce(partial_content,'<null>')||'|'||(completed_at is null)||'|'||coalesce(error_code,'<null>')||'|'||citations::text||'|'||blocks::text||'|'||coalesce(confidence::text,'<null>')||'|'||low_confidence||no_context||degraded||'|'||masked_count||'|'||coalesce(intent,'<null>')||'|'||coalesce(feedback::text,'<null>')||'|'||attempt from chat_messages where id='{mid}'")
time.sleep(14); A('admin','POST','/_test/llm/fake',{'stream_delay_ms':20})
f=row.split('|')
g=subprocess.run(GO+"go test -tags integration ./internal/chat -run 'TestReapInterrupted|TestReapSkipsLive|TestReapDoesNotOverwriteLive|TestRetryResetsAllColumns|TestDrainMarksInterruptedAndReleasesBusy|TestUpdatedAtTrigger' -v -count=1 2>&1 | grep -E '^(--- |ok|FAIL)'",shell=True,capture_output=True,text=True).stdout
okc=f[0]=='' and f[2]=='true' and f[3]=='<null>' and f[4]=='[]' and f[5]=='[]' and f[6]=='<null>' and f[7]=='falsefalsefalse' and f[8]=='0' and f[9]=='<null>' and f[10]=='<null>' and f[11]=='2'
rec('TC-P3-05-51',s in(200,202) and okc and 'FAIL' not in g,f'tin FAILED có citations/blocks/confidence/cờ/masked/intent/feedback → retry {s}; ngay sau (đang STREAMING chậm): {row} (content rỗng, completed_at NULL, error_code NULL, citations=[], blocks=[], confidence NULL, 3 cờ false, masked_count=0, intent NULL, feedback NULL, attempt 1→2); partial_content={f[1]!r} (đã bắt đầu lượt mới); go test: {g.replace(chr(10)," | ")[:200]}')
json.dump(R,open('/tmp/qc3e/w15.json','w'),ensure_ascii=False)
