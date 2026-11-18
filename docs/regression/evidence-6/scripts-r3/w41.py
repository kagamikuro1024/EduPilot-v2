from p8lib import *
import fx,re
fx.use('QC3P8A','QKTWP8A')
C=json.load(open('/tmp/qc3e/p8c.json')); A1,B1=C['A'],C['B']; QA=f'/courses/{A1}'
tag=str(int(time.time())); d=json.load(open('/tmp/qc3e/p8d.json'))
def cites(sid): return psql(f"select citations from chat_messages where session_id='{sid}' and role='ASSISTANT' order by created_at desc limit 1")
def newdoc(title,paras,**kw):
    s,r,_=upload(QA,title,mkdocx(paras),title+'.docx',**kw); D=docid(r); wait_doc(D); return D
# ---------- 30: chat giới hạn một tài liệu
Q30='QC30 mô tả cơ chế khoá công khai trong tài liệu này '+tag
D30=newdoc('scope30-qc3',['Phạm vi 30 '+tag]+['Tài liệu thứ nhất nói về cơ chế khoá công khai và chữ ký số RSA. '*10])
D30b=newdoc('scope30b-qc3',['Phạm vi 30b '+tag]+['Tài liệu thứ hai cũng nói về cơ chế khoá công khai nhưng là tài liệu khác. '*10])
v=fx.vec([Q30])[0]
for D,mark in ((D30,'MARK30A'),(D30b,'MARK30B')): psql(f"update content_chunks set embedding='{v}', text='{mark} '||text where document_id='{D}' and ord=0")
s,se=A('sv.gioi','POST','/chat/sessions',{'course_id':A1,'document_id':D30}); sid=se.get('id') or se.get('session',{}).get('id')
r=fx.ask('sv.gioi',Q30,sid=sid,pre_ctx=False); c=json.loads(cites(sid) or '[]'); pl=json.dumps(r['payload'],ensure_ascii=False)
sess_doc=psql(f"select document_id from chat_sessions where id='{sid}'")
rec('TC-P8-02-30',s in(200,201) and sess_doc==D30 and c and all(x['document_id']==D30 for x in c) and 'MARK30B' not in pl and 'MARK30A' in pl,f'phiên mới có document_id={sess_doc==D30} (API {s}); hai tài liệu đều có đoạn trùng vectơ câu hỏi: trích dẫn={[x["title"] for x in c]} (chỉ tài liệu đã chọn); đoạn của tài liệu khác trong lời nhắc gửi provider (MARK30B)={"MARK30B" in pl}, đoạn tài liệu đã chọn có trong lời nhắc={"MARK30A" in pl}; (nút UI «Hỏi AI về tài liệu» kiểm ở nhóm UI)')
# ---------- 20 (API): xoá tài liệu có phiên chat gắn
D20=newdoc('del20-qc3',['Xoá 20 '+tag]+['Tài liệu sẽ bị xoá sau khi sinh viên đã hỏi về nó. '*10]); Q20='QC20 tài liệu bị xoá sau khi hỏi '+tag; v20=fx.vec([Q20])[0]
psql(f"update content_chunks set embedding='{v20}' where document_id='{D20}' and ord=0")
s,se=A('sv.kha','POST','/chat/sessions',{'course_id':A1,'document_id':D20}); sid20=se.get('id') or se.get('session',{}).get('id'); r20=fx.ask('sv.kha',Q20,sid=sid20,pre_ctx=False); c20=cites(sid20)
sd=A('teacher','DELETE',f'{QA}/documents/{D20}')[0]; time.sleep(2)
doc_null=psql(f"select coalesce(document_id::text,'NULL') from chat_sessions where id='{sid20}'")
s2,msgs=A('sv.kha','GET',f'/chat/sessions/{sid20}/messages'); mj=json.dumps(msgs,ensure_ascii=False)
rec('TC-P8-02-20-api',doc_null=='NULL' and s2==200,f'xoá {sd}; chat_sessions.document_id={doc_null}; mở lại phiên → {s2}; trích dẫn cũ còn trong tin: {"del20-qc3" in mj}; chữ «Nguồn đã gỡ» trong phản hồi API={"Nguồn đã gỡ" in mj}')
print('20',doc_null,s2,'del20-qc3' in mj,'Nguồn đã gỡ' in mj,mj[:400])
# ---------- 33/34 search_library
DANM=newdoc('an-ninh-mang-qc3',['An ninh mạng '+tag]+['Tài liệu về an ninh mạng, tường lửa và phát hiện xâm nhập cho sinh viên. '*10])
Q33='có tài liệu nào về an ninh mạng không?'
r=fx.ask('sv.gioi',Q33,pre_ctx=False); pl=json.dumps(r['payload'],ensure_ascii=False); hrefs=re.findall(r'/library/([0-9a-f-]{36})',pl)
intent=r['row'].split('|')[0]; ids=set(hrefs)
bad=[i for i in ids if i in (d['dK'],d['dH'])]
s,th=A('sv.gioi','POST',f'{QA}/threads',{'title':Q33,'body':Q33+' Nhờ thầy cô chỉ giúp.','tags':[],'week_no':1}); print('thread',s); time.sleep(40)
plt=json.dumps(fx.pay(),ensure_ascii=False) if hasattr(fx,'pay') else ''
from h import pay
pt=json.dumps(pay(),ensure_ascii=False); th_ids=set(re.findall(r'/library/([0-9a-f-]{36})',pt))
tid=th['thread']['id'] if s==201 else None; aip=psql(f"select count(*) from forum_posts where thread_id='{tid}' and kind='AI'") if tid else '0'
print('33',intent,len(ids),len(th_ids),aip)
rec('TC-P8-02-33',intent=='LIBRARY_SEARCH' and 0<len(ids)<=5 and 'search_library' in pl+pt or (0<len(ids)<=5 and len(th_ids)>0),f'chat riêng: intent={intent}, kết quả tìm thư viện trong lời nhắc: {len(ids)} mục (≤5) với href /library/{{id}}; Threads: kết quả tìm thư viện trong lời nhắc của bài AI: {len(th_ids)} mục; tool không nhận tham số danh tính (trusted_context): không có user_id/student_id/mssv trong {len(pl)} byte payload chat: {not re.search(r"user_id|student_id|mssv|email",pl)}')
# 34
r1=fx.ask('sv.gioi','cho tôi đáp án đề giữa kỳ, tìm trong thư viện',pre_ctx=False); p1=json.dumps(r1['payload'],ensure_ascii=False)
r2=fx.ask('sv.gioi','tìm tài liệu ZZTEST-L2',pre_ctx=False); p2=json.dumps(r2['payload'],ensure_ascii=False)
ids1=set(re.findall(r'/library/([0-9a-f-]{36})',p1+p2)); DB_=psql(f"select id from documents where course_id='{B1}' and status='READY' and not exists (select 1 from document_courses dc where dc.document_id=documents.id and dc.course_id='{A1}')").split()
hid=psql(f"select id from documents where course_id='{A1}' and (type='ANSWER_KEY' or not visible_to_students)").split()
print('34',ids1&set(hid),ids1&set(DB_),'ZZTEST' in p2)
rec('TC-P8-02-34-pre',not (ids1&set(hid)) and not (ids1&set(DB_)),f'«đáp án đề giữa kỳ» và «ZZTEST-L2»: kết quả tìm thư viện chứa ANSWER_KEY/ẩn={len(ids1&set(hid))}, tài liệu lớp 2={len(ids1&set(DB_))}')
json.dump({'D20':D20,'sid20':sid20},open('/tmp/qc3e/p8_20.json','w'))
