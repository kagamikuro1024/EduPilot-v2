from fx import *
import fx,re
fx.use('QC3P8A','QKTWP8A')
A1=fx.QA; d=json.load(open('/tmp/qc3e/p8d.json')); dK,dL,dH=d['dK'],d['dL'],d['dH']
Q0='Hãy giải thích chế độ mã hoá khối GCM và CBC khác nhau thế nào QC30'
v=vec([Q0])[0]
def ins(doc,aud,n,txt,start):
    for i in range(n): psql(f"insert into content_chunks (document_id,course_ids,audience,ord,text,embedding) values ('{doc}',array['{A1}'::uuid],'{aud}',{start+i},'{txt} {i}','{v}')")
ins(dK,'GRADING',60,'Đáp án CANARY-7Q2X câu một của đề giữa kỳ, chế độ GCM',100)
ins(dH,'STAFF',5,'Ghi chú nội bộ chỉ nhân viên CBC GCM',100)
ins(dL,'ALL',9,'Bài giảng hợp lệ về chế độ mã hoá khối GCM CBC',100)
r=fx.ask('sv.gioi',Q0,pre_ctx=False)
cits=psql(f"select citations from chat_messages where session_id='{r['sid']}' and role='ASSISTANT'")
gen=[it for it in r['payload'] if it['task']!='EMBEDDING']
pl=json.dumps(r['payload'],ensure_ascii=False)
print(r['row'],len(r['text']))
print(cits[:1500]); print('canary in payload',pl.count('CANARY-7Q2X'),'in text',r['text'].count('CANARY'))
print(json.dumps(gen,ensure_ascii=False)[:2500])
