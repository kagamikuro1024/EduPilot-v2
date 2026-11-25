import sys,json; sys.path.insert(0,'/Users/kuro/Documents/TA_Agent_qc3/docs/regression/evidence-oc/scripts')
import oc
C=oc.psql("select id from courses where class_code='QCOC1'")
ex=oc.psql(f"select e.id||'|'||e.course_id from exams e where e.course_id<>'{C}' limit 1").split('|')
q=oc.psql(f"select id from question_bank where course_id<>'{C}' limit 1")
at=oc.psql(f"select a.id||'|'||a.exam_id from exam_attempts a where a.course_id<>'{C}' limit 1").split('|')
print('foreign exam',ex[0][:8],'question',q[:8],'attempt',at[0][:8])
out=[]
for who in ('teacher','ta','sv.gioi'):
    for label,m,p in (('GET exam','GET',f'/courses/{C}/exams/{ex[0]}'),('GET question','GET',f'/courses/{C}/questions/{q}'),('GET attempt','GET',f'/courses/{C}/exams/{at[1]}/attempts/{at[0]}'),('GET results','GET',f'/courses/{C}/exams/{ex[0]}/results'),('GET csv','GET',f'/courses/{C}/exams/{ex[0]}/results.csv')):
        s,d=oc.A(who,m,p); out.append((who,label,s)); print(who,label,s, (d or {}).get('code') if isinstance(d,dict) else '')
# trộn câu hỏi lớp khác vào bài thi lớp 1 (teacher): cần bài thi nháp của lớp 1
s,d=oc.A('teacher','POST',f'/courses/{C}/exams',{'title':'QC3 LT cách ly','opens_at':None})
print('create exam',s,str(d)[:120])
E=d['id']; ver=d.get('version',1)
s,d2=oc.A('teacher','PUT',f'/courses/{C}/exams/{E}/items',{'items':[{'question_id':q,'points':1}],'version':ver}); print('PUT items foreign q',s,json.dumps(d2,ensure_ascii=False)[:400])
