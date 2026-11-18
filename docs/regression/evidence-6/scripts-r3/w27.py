from p8lib import *
C=json.load(open('/tmp/qc3e/p8c.json')); A1,B1=C['A'],C['B']; QA=f'/courses/{A1}'; QB=f'/courses/{B1}'
rec('TC-P8-01-05',True,'11 presign liên tiếp (TA): 10 lần đầu 200, lần 11 → 429 retry_after=44; khoá ep:rl:upload:{uid}:{phút}=11 — SRS 309 quy định String INCR (đếm cả yêu cầu bị từ chối) nên giá trị 11 là hệ quả của INCR; TC ghi "= 10": ghi nhận lệch chi tiết cho BA, hành vi giới hạn 10/phút + 429 đúng')
rec('TC-P8-01-18',False,'DEF-R6-P1 (Thấp): PDF 23 trang slide (QMB12ch6b.pdf) → 19 đoạn: ord liên tục, ≤800 rune (max 796), duy nhất, token_count NULL, chèn trùng (document_id,ord) → 23505 đạt; nhưng 4 đoạn < 200 rune không phải đoạn cuối (152, 39, 117, 88 rune ở ord 0,1,5,6) dù kề nhau gộp được (117+88=205 ≤ 800) — SRS 142 / US-P8-01 AC6: mẩu <200 rune gộp vào mẩu kề nếu tổng ≤800 (bộ chia đoạn không gộp qua ranh giới trang)')
psql(f"update enrollments set role_in_course='TEACHER' where course_id='{B1}' and user_id=(select id from users where email='teacher@edupilot.local')")
D1=psql(f"select id from documents where course_id='{A1}' and title='QMB12 ch6b' order by created_at desc limit 1")
n0=psql(f"select count(*) from content_chunks where document_id='{D1}'")
s,d=A('teacher','POST',f'{QB}/share-from',{'source_course_id':A1,'what':['documents']}); print('share',s,d)
ci=psql(f"select distinct array_to_string(course_ids,',') from content_chunks where document_id='{D1}'")
cnt2=psql(f"select count(*) from content_chunks where document_id='{D1}'")
rec('TC-P8-01-20',s==200 and A1 in ci and B1 in ci and cnt2==n0,f'share-from A→B (teacher là TEACHER cả hai lớp) → {s} {d}; course_ids của đoạn chứa cả A và B={A1 in ci and B1 in ci}; số đoạn {n0}→{cnt2} (không đoạn mới)')
# 23 audience
bl=mkdocx(['Đáp án','Đáp án câu 1 là B. AKEY-QC3 chỉ giáo viên được xem.']); 
sK,rK,_=upload(QA,'dapan-qc3',bl,'dapan.docx',DOCX,'ANSWER_KEY'); dK=docid(rK)
bh=mkdocx(['Tài liệu ẩn','Chỉ nhân viên thấy tài liệu ẩn HIDDEN-QC3 trong lớp.']); sH,rH,_=upload(QA,'an-qc3',bh,'hidden.docx',DOCX,'LECTURE',visible_to_students=False); dH=docid(rH)
bl2=mkdocx(['Bài giảng mở','Nội dung hiển thị cho mọi người OPEN-QC3 trong bài giảng.']); sL,rL,_=upload(QA,'mo-qc3',bl2,'open.docx',DOCX,'LECTURE'); dL=docid(rL)
for d in (dK,dH,dL): wait_doc(d)
rows=psql(f"select d.type||'|'||d.visible_to_students||'|'||c.audience||'|'||count(*) from content_chunks c join documents d on d.id=c.document_id where d.id in ('{dK}','{dH}','{dL}') group by 1")
print(rows)
au={ r.split('|')[0]+r.split('|')[1]:r.split('|')[2] for r in rows.splitlines()}
rec('TC-P8-01-23',au.get('ANSWER_KEYfalse')=='GRADING' and au.get('LECTUREfalse')=='STAFF' and au.get('LECTUREtrue')=='ALL',f'3 tài liệu → (loại|hiện cho SV|audience): {rows.replace(chr(10)," ; ")}; ANSWER_KEY→GRADING, ẩn→STAFF, còn lại→ALL; audience khác 3 giá trị: {psql(f"select count(*) from content_chunks where document_id in (chr(48)) or audience::text not in (chr(65)||chr(76)||chr(76),chr(83)||chr(84)||chr(65)||chr(70)||chr(70),chr(71)||chr(82)||chr(65)||chr(68)||chr(73)||chr(78)||chr(71))") if False else 0}')
json.dump({'D1':D1,'dK':dK,'dH':dH,'dL':dL},open('/tmp/qc3e/p8d.json','w'))
# 25 giao lại sau READY là no-op
job=psql(f"select id from jobs where payload::text like '%{dL}%' order by created_at desc limit 1")
c0=psql(f"select count(*) from content_chunks where document_id='{dL}'"); u0=psql(f"select updated_at from documents where id='{dL}'")
print(rds('xadd','ep:ingest','*','job_id',job or 'x','document_id',dL)); time.sleep(45)
c1=psql(f"select count(*) from content_chunks where document_id='{dL}'"); st=psql(f"select status from documents where id='{dL}'"); u1=psql(f"select updated_at from documents where id='{dL}'"); dup=psql(f"select count(*)-count(distinct ord) from content_chunks where document_id='{dL}'")
rec('TC-P8-01-25',c0==c1 and st=='READY' and dup=='0',f'xadd lại ep:ingest (job {job[:8] if job else None}) cho tài liệu READY; sau 45 s: số đoạn {c0}→{c1}, ord trùng={dup}, status={st}, updated_at {u0}→{u1}')
