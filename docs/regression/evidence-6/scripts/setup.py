from lib import *
tid=psql("select id from users where email='teacher@edupilot.local'"); taid=psql("select id from users where email='ta@edupilot.local'")
out={}
for code,jc in (('E6C1','QKTWXZH'),('E6C2','QKTWXZM')):
    ex=psql(f"select id from courses where class_code='{code}'")
    if not ex:
        s,d=A('admin','POST','/admin/courses',{'subject_code':'QCE6','class_code':code,'name':'QC nhóm E '+code,'semester':'2026-2027-HK1','capacity':50,'teacher_id':tid,'ta_ids':[taid],'join_code':jc})
        print(code,s,d); ex=d['id'] if s in(200,201) else ''
    out[code]=ex
print(out)
for code,jc in (('E6C1','QKTWXZH'),('E6C2','QKTWXZM')):
    for u in (['sv.gioi','sv.kha'] if code=='E6C1' else ['sv.moi']):
        s,d=A(u,'POST','/courses/join',{'code':jc}); print(code,u,s,json.dumps(d,ensure_ascii=False)[:200])
json.dump(out,open('/tmp/qc2e/courses.json','w'))
