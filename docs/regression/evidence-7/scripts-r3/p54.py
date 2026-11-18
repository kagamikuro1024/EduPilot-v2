from lib3 import *
import statistics,datetime as dt
fx=json.load(open('/tmp/qc3e/r4fx.json')); fm=json.load(open('/tmp/qc3e/fx_me.json')); L=fx['L']; M=fm['M']
def dk(*a): return subprocess.run(['docker',*a],capture_output=True,text=True)
# ---- P5-04-09 : lớp 1.000 SV (LOAD1000, GV teacher@edupilot.local)
LD=psql("select id from courses where class_code='LOAD1000'"); Q=f'/courses/{LD}'; tk=login('teacher')
ns=psql(f"select count(*) from enrollments where course_id='{LD}' and role_in_course='STUDENT' and status='ACTIVE'")
ts=[]
for i in range(30):
    t=time.time(); s,t_,h=raw('GET',f'{Q}/students?limit=50',tk); ts.append((time.time()-t)*1000); assert s==200,(s,t_[:100])
p95=sorted(ts)[int(0.95*len(ts))-1]; sid=json.loads(t_)['items'][0]['student']['id']
pt=[]
for i in range(5):
    t=time.time(); s,t_,h=raw('GET',f'{Q}/students/{sid}/profile',tk); pt.append((time.time()-t)*1000)
nstd=psql("select count(*) from study_time_daily where course_id='%s'"%LD)
rec('TC-P5-04-09',p95<=300 and max(pt)<=3000,f'lớp LOAD1000 ({ns} SV ACTIVE, GV teacher): GET students?limit=50 ×30 (qua caddy HTTPS, tuần tự): trung vị {statistics.median(ts):.0f} ms, p95 {p95:.0f} ms, max {max(ts):.0f} ms (≤ 300); hồ sơ 360 ×5: {[round(x) for x in pt]} ms (≤ 3.000); (study_time_daily của lớp này = {nstd} hàng)')
# ---- P5-04-P1 : phân quyền chéo lớp
r1=[A('qc3ta','GET',f'/courses/{LD}/students')[0], A('qc3ta','GET',f'/courses/{LD}/students/{sid}/profile')[0], A('qc3t2','GET',f'/courses/{M}/students')[0], A('qc3t2','GET',f'/courses/{M}/students/{uid("qc3me01")}/profile')[0], A('qc3t','GET',f'/courses/{M}/students')[0]]
rec('TC-P5-04-P1',r1[:4]==[403]*4 and r1[4]==200,f'TA lớp khác (qc3ta, không thuộc LOAD1000) GET students → {r1[0]}, profile → {r1[1]}; teacher lớp khác (qc3t2, không thuộc QC3M5) GET students → {r1[2]}, profile → {r1[3]}; teacher đúng lớp → {r1[4]}')
# ---- P5-05-P1 : SV đọc observations của chính mình
Qm=f'/courses/{M}'; me=uid('qc3me01'); A('qc3t','POST',f'{Qm}/students/{me}/observations',{'body':'Ghi chú riêng QC3 P1','tags':[]})
r2=[A('qc3me01','GET',f'{Qm}/students/{me}/observations')[0],A('qc3me01','GET',f'{Qm}/students/{me}/profile')[0],A('qc3me01','POST',f'{Qm}/students/{me}/observations',{'body':'x','tags':[]})[0],A('qc3me02','GET',f'{Qm}/students/{me}/observations')[0],A('qc3t','GET',f'{Qm}/students/{me}/observations')[0]]
txt=json.dumps([A('qc3me01','GET','/me/today')[1],A('qc3me01','GET',f'{Qm}/me/attendance')[1],A('qc3me01','GET',f'{Qm}/me/participation')[1]])
rec('TC-P5-05-P1',r2[:4]==[403]*4 and r2[4]==200 and 'Ghi chú riêng QC3' not in txt,f'SV qc3me01 đọc observations của CHÍNH MÌNH → {r2[0]}; đọc profile → {r2[1]}; ghi observation → {r2[2]}; SV khác đọc → {r2[3]}; GV → {r2[4]}; nội dung ghi chú không xuất hiện ở /me/today, me/attendance, me/participation')
# ---- P5-03-E2 : Redis mất dữ liệu giữa ngày
ICT=dt.timezone(dt.timedelta(hours=7)); day=dt.datetime.now(ICT).strftime('%Y%m%d'); X=fx['X']; u=fx['S'][5]; usr=uid(u)
def rget(k): v=rds('GET',k); return int(v) if v else 0
k=f'ep:study:{X}:{usr}:{day}:CALENDAR'; rds('SET',k,'150','EX','691200'); rds('SADD','ep:study:dirty',k)
def dbv(): return psql(f"select coalesce(sum(active_seconds),0) from study_time_daily where student_id='{usr}' and course_id='{X}' and area='CALENDAR' and date=(now() at time zone 'Asia/Ho_Chi_Minh')::date")
for _ in range(80):
    v1=dbv()
    if v1=='150': break
    time.sleep(2)
for kk in rds('keys',f'ep:study:*:{usr}:*').split()+rds('keys',f'ep:hb:*:{usr}*').split(): rds('DEL',kk)   # = Redis khởi động lại không lưu: mất mọi khoá đếm của SV
rds('HDEL','ep:study:flushed',k)
login(u); psql(f"update users set tracking_notice_ack_at=now() where id='{usr}'"); rds('DEL',f'ep:trk:{usr}')
tid=str(uuid.uuid5(uuid.NAMESPACE_DNS,'e2'))
s,_=A(u,'POST','/activity/heartbeat',{'course_id':X,'area':'CALENDAR','tab_id':tid}); newk=rget(k)
time.sleep(75); v2=dbv(); neg=psql("select count(*) from study_time_daily where active_seconds<0")
rec('TC-P5-03-E2',v1=='150' and v2=='150' and neg=='0' and newk==30,f'DB sau rollup = {v1}; mô phỏng Redis khởi động lại không lưu giữa ngày (xoá mọi khoá ep:study:*/ep:hb:* của SV; không FLUSHALL vì dùng chung stack): nhịp mới đếm lại từ {newk}; sau rollup kế (75 s) DB = {v2} (GREATEST: không giảm), hàng active_seconds<0 toàn bảng = {neg}')
# ---- P5-04-E1 : crm.risk bị kill giữa chừng
fd='/tmp/qc3e/gw.env'; env=dk('inspect','ep-r-gateway-1','--format','{{range .Config.Env}}{{println .}}{{end}}').stdout
open(fd,'w').write('\n'.join(l for l in env.split('\n') if l and not l.startswith('APP_ENV=') and not l.startswith('HTTP_ADDR='))+'\nAPP_ENV=dev\n')
def risk_rows(): return psql(f"select count(*)||'|'||count(distinct computed_at)||'|'||coalesce(max(computed_at)::text,'-') from student_risk where course_id='{L}'")
dk('rm','-f','ep-r-risk-qc3')
r=dk('run','--rm','--name','ep-r-risk-qc3','--network','ep-r_default','--env-file',fd,'edupilot-gateway-test','admin','crm-risk-run','--course','QC3L'); base=risk_rows(); o0=(r.stdout+r.stderr).strip()[-80:]
psql(f"create or replace function qc3_slow_risk() returns trigger as $$ begin if NEW.course_id='{L}' then perform pg_sleep(0.006); end if; return NEW; end $$ language plpgsql"); psql("drop trigger if exists qc3_slow_risk_t on student_risk"); psql("create trigger qc3_slow_risk_t before insert or update on student_risk for each row execute function qc3_slow_risk()")
try:
    p=subprocess.Popen(['docker','run','--rm','--name','ep-r-risk-qc3','--network','ep-r_default','--env-file',fd,'edupilot-gateway-test','admin','crm-risk-run','--course','QC3L'],stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True)
    time.sleep(3.5); mid=psql("select count(*) from pg_stat_activity where query ilike '%student_risk%' and state<>'idle' and pid<>pg_backend_pid()"); dk('kill','ep-r-risk-qc3'); p.wait(timeout=30); after_kill=risk_rows()
    time.sleep(1)
finally:
    psql("drop trigger if exists qc3_slow_risk_t on student_risk"); psql("drop function if exists qc3_slow_risk()")
r=dk('run','--rm','--name','ep-r-risk-qc3','--network','ep-r_default','--env-file',fd,'edupilot-gateway-test','admin','crm-risk-run','--course','QC3L'); final=risk_rows(); o2=(r.stdout+r.stderr).strip()[-80:]
dup=psql(f"select count(*) from (select student_id from student_risk where course_id='{L}' group by 1 having count(*)>1) x")
b=base.split('|'); a=after_kill.split('|'); f=final.split('|')
rec('TC-P5-04-E1',b[0]=='1000' and a==b and f[0]=='1000' and f[1]=='1' and f[2]>b[2] and dup=='0',f'lớp QC3L (1.000 SV): chạy đủ → {base} (số hàng|số computed_at khác nhau|mới nhất) [{o0}]; chạy lại với trigger làm ghi chậm 6 ms/hàng, kill container ở giây 3,5 (giao dịch đang ghi; pg_stat_activity đang chạy={mid}) → sau kill {after_kill} (nguyên trạng cũ: giao dịch một lớp được rollback, không hàng nào có computed_at mới = không dữ liệu nửa vời); chạy lại → {final} [{o2}] (1.000 hàng, một computed_at mới), hàng trùng={dup}; trigger đã gỡ')
save('p54')
