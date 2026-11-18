from lib3 import *
import threading,random,datetime as dt
fx=json.load(open('/tmp/qc3e/r4fx.json')); X,Y,S=fx['X'],fx['Y'],fx['S']
ICT=dt.timezone(dt.timedelta(hours=7)); day=dt.datetime.now(ICT).strftime('%Y%m%d')
def tabid(n): return str(uuid.uuid5(uuid.NAMESPACE_DNS,n))
def hb(u,c,area='CHAT',tab='t1'): return A(u,'POST','/activity/heartbeat',{'course_id':c,'area':area,'tab_id':tabid(tab)})
# ---- 40: đua ghi danh vs đổi cờ
al='ABCDEFGHJKMNPQRSTUVWXYZ23456789'; tid=uid('qc3t'); res=[]
stu=[f'qc3at{i}' for i in range(100,125)]
for x in stu:
    login(x); time.sleep(1.5)

def run(i):
    code=''.join(random.choice(al) for _ in range(7)); cc='QC3R'+str(random.randint(10000,99999))
    s,d=A('admin','POST','/admin/courses',{'subject_code':'QC3','class_code':cc,'name':'QC3 race','semester':'2026-2027-HK1','capacity':50,'teacher_id':tid,'ta_ids':[],'join_code':code})
    if s not in(200,201): return ('CREATE',s,str(d)[:300])
    cid=d['id']; g=A('qc3t','GET',f'/courses/{cid}')[1]; ver=g['settings']['version']; out={}
    def a(): out['join']=A(stu[i%25],'POST','/courses/join',{'code':code})[0]
    def b():
        time.sleep(random.choice([0,0.05,0.1,0.15,0.25]))
        s2,d2=A('qc3t','PUT',f'/courses/{cid}/settings',{'allow_study_time_opt_out':False,'version':ver}); out['put']=(s2,(d2 or {}).get('code') if isinstance(d2,dict) else None)
    ts=[threading.Thread(target=a),threading.Thread(target=b)]; [t.start() for t in ts]; [t.join() for t in ts]
    flag=psql(f"select allow_study_time_opt_out from courses where id='{cid}'"); enr=psql(f"select status from enrollments where course_id='{cid}' and role_in_course='STUDENT'")
    # thứ tự: mốc ghi danh so với mốc cập nhật courses
    ordr=''
    return (i,out,flag,enr,ordr)
for i in range(50): res.append(run(i)); 
bad=[];c1=c2=0
for r in res:
    if r[0]=='CREATE': bad.append(r); continue
    i,out,flag,enr,ordr=r
    if out['join']!=200 or enr!='ACTIVE': bad.append(('join',r)); continue
    if out['put'][0]==200 and flag=='f': c1+=1   # PUT thắng, SV vào sau luật mới
    elif out['put'][0]==409 and flag=='t': c2+=1  # SV vào trước, PUT 409
    else: bad.append(('inconsistent',r))
rec('TC-P5-03-40',len(bad)==0 and c1+c2==50,f'50 lượt song song (lớp mới mỗi lượt, 25 SV luân phiên, PUT trễ ngẫu nhiên 0–250 ms): (1) PUT thắng (200, cờ=false) rồi SV vào lớp dưới luật mới: {c1}; (2) SV vào trước, PUT 409 COURSE_HAS_STUDENTS (cờ giữ true): {c2}; lượt lệch/khác: {len(bad)} {json.dumps(bad[:2],ensure_ascii=False)[:300]}')
save('p53r4c')
