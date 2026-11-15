# TC-P203-05/08/24, P204-02/24/37: đăng ký trùng theo trạng thái, MSSV mạo danh, roster→PENDING→ACTIVE, thư quên mật khẩu, leo vai phiên, nội dung thư. Cần stack qc2i.
import os; exec(open(os.path.join(os.path.dirname(os.path.abspath(__file__)),'qc2i-r4apia-lib.py')).read())
A=admin_tok()
def uid_of(em): return psql(f"select id from users where email='{em}'")
def count_users(em): return psql(f"select count(*) from users where email='{em}'")
def newest(em,since_n): ms=sorted(mails(em),key=lambda m:m['Created']); return [m['Subject'] for m in ms][since_n:]
def reg_raw(em,code=''): resetrl(); return call('POST','/auth/register',None,{'email':em,'password':PWD,'full_name':'QC2IA dup','student_code':code})
def wait_new(em,n0,wait=40):
    end=time.time()+wait
    while time.time()<end:
        if mail_count(em)>n0: break
        time.sleep(1)
    time.sleep(2); return newest(em,n0)
# ---------- P203-05
res05={}
new=f'qc2ia-new5.{TAG}@edupilot.local'; n0=0; s,_,_=reg_raw(new); res05['mới']=(s,wait_new(new,0),count_users(new))
act=reg('a5'); n0=mail_count(act); s,_,_=reg_raw(act); res05['ACTIVE']=(s,wait_new(act,n0),count_users(act))
pen=reg('p5',verify=False); n0=mail_count(pen); s,_,_=reg_raw(pen); res05['PENDING']=(s,wait_new(pen,n0),count_users(pen))
inv,_=invite('i5',role='TEACHER',accept=False); mail_to(inv,'mời|Lời mời',iso()[:0],40); n0=mail_count(inv); s,_,_=reg_raw(inv); res05['INVITED(GV)']=(s,wait_new(inv,n0),count_users(inv))
dis=reg('d5'); did=uid_of(dis); cur=call('GET','/admin/users?q='+dis,A)[1]['items'][0]; call('PATCH',f'/admin/users/{did}',A,{'status':'DISABLED','version':cur['version']}); n0=mail_count(dis); s,_,_=reg_raw(dis); time.sleep(12); res05['DISABLED']=(s,newest(dis,n0),count_users(dis))
# lớp + roster (dùng cho INVITED(SV), P203-08, P203-24)
tem,tid=invite('gv',role='TEACHER'); T=lg(tem)[1]['access_token']; cc=f'QA{TAG}'.upper()
s,c,_=call('POST','/admin/courses',A,{'subject_code':'INT1006','class_code':cc,'name':'Lớp QC2IA '+TAG,'semester':'2026-2027-HK1','teacher_id':tid}); assert s==201,(s,c); cid=c['id']
def multipart(data,fname='r.csv'):
    b='----qc'+uuid.uuid4().hex; body=(f'--{b}\r\nContent-Disposition: form-data; name="file"; filename="{fname}"\r\nContent-Type: text/csv\r\n\r\n').encode()+data+f'\r\n--{b}--\r\n'.encode()
    return body,f'multipart/form-data; boundary={b}'
def imp(rows,dry=False):
    csv=('Email,Họ và tên,MSSV\n'+''.join(f'{e},{n},{m}\n' for e,n,m in rows)).encode('utf8'); body,ct=multipart(csv)
    return call('POST',f'/courses/{cid}/roster/import'+('?dry_run=true' if dry else ''),T,raw=body,hdr={'content-type':ct,'Idempotency-Key':str(uuid.uuid4())},idem=False)
sv=f'qc2ia-sv5.{TAG}@edupilot.local'; r=imp([(sv,'SV Roster Năm','AB2022905')]); assert r[0]==200,r
time.sleep(2); n0=mail_count(sv); s,_,_=reg_raw(sv); res05['INVITED(SV, có lớp roster)']=(s,wait_new(sv,n0),count_users(sv))
exp={'mới':'Xác minh email','ACTIVE':'Bạn đã có tài khoản','PENDING':'Bạn đã có tài khoản','INVITED(GV)':'Lời mời tham gia','INVITED(SV, có lớp roster)':'Bạn được thêm vào lớp'}
bad=[k for k,v in exp.items() if not (res05[k][0]==202 and len(res05[k][1])==1 and v in res05[k][1][0] and res05[k][2]=='1')]
if not (res05['DISABLED'][0]==202 and res05['DISABLED'][1]==[] and res05['DISABLED'][2]=='1'): bad.append('DISABLED')
rec('TC-P203-05','P' if not bad else 'F',f'đăng ký (cùng 202) theo trạng thái → thư mới/số users: '+'; '.join(f'{k}: {v[1]} / users={v[2]}' for k,v in res05.items())+(f' LỆCH {bad}' if bad else ''))
# ---------- P203-24 PENDING EMAIL_UNVERIFIED → ACTIVE
u24=reg('u24',verify=False); t0=iso(); r=imp([(u24,'QC2IA U24','AB2022924')]); time.sleep(2)
row=psql(f"select e.status,coalesce(e.warning::text,'') from enrollments e join users u on u.id=e.user_id where u.email='{u24}' and e.course_id='{cid}'")
m=mail_to(u24,'Xác minh',t0[:0]+'',40); s,j,_=call('POST','/auth/verify-email',None,{'token':link_token(m,'verify-email')}); time.sleep(2)
row2=psql(f"select e.status,coalesce(e.warning::text,'') from enrollments e join users u on u.id=e.user_id where u.email='{u24}' and e.course_id='{cid}'")
rec('TC-P203-24','P' if row.startswith('PENDING|') and 'UNVERIFIED' in row and s==200 and row2.startswith('ACTIVE|') else 'F',f'roster có email chưa xác minh → enrollments {row}; sau xác minh email ({s}) → {row2}')
# ---------- P203-08 MSSV mạo danh
vic=f'qc2ia-v8.{TAG}@edupilot.local'; imp([(vic,'Nạn nhân Tám','AB2022908')]); time.sleep(2)
n_enr0=psql(f"select count(*) from enrollments where course_id='{cid}'"); res08={}
for lab,code in [('chữ thường','ab2022908'),('có khoảng trắng',' AB2022908 '),('lẫn hoa-thường','Ab2022908')]:
    em=reg('m8'+str(len(res08)),code=code); tk=lg(em)[1]['access_token']; mc=call('GET','/me/courses',tk)[1]; en=psql(f"select count(*) from enrollments e join users u on u.id=e.user_id where u.email='{em}'")
    saved=psql(f"select coalesce(student_code,'') from users where email='{em}'"); res08[lab]=(len(mc.get('items',[])),en,saved)
# MSSV của người ĐÃ vào lớp (có tài khoản): nạn nhân đã có tài khoản + vào lớp qua roster
mem=reg('mem8'); imp([(mem,'Thành viên Tám','AB2022909')]); time.sleep(2); enm=psql(f"select e.status from enrollments e join users u on u.id=e.user_id where u.email='{mem}' and e.course_id='{cid}'")
att2=reg('m8b',code='AB2022909'); tk=lg(att2)[1]['access_token']; mc2=call('GET','/me/courses',tk)[1]; en2=psql(f"select count(*) from enrollments e join users u on u.id=e.user_id where u.email='{att2}'")
# đăng ký TRƯỚC khi roster được nạp rồi nạp roster chứa MSSV đó
att3=reg('m8c',code='AB2022910'); v3=f'qc2ia-v10.{TAG}@edupilot.local'; r3=imp([(v3,'Nạn nhân Mười','AB2022910')]); time.sleep(2)
tk3=lg(att3)[1]['access_token']; mc3=call('GET','/me/courses',tk3)[1]; en3=psql(f"select count(*) from enrollments e join users u on u.id=e.user_id where u.email='{att3}'")
v3row=psql(f"select u.email,e.status from enrollments e join users u on u.id=e.user_id where e.course_id='{cid}' and e.student_code_snapshot='AB2022910'")
members=call('GET',f'/courses/{cid}/members?limit=100',T)[1]; emails=[x.get('email') for x in members.get('items',[])]
ok=all(v[0]==0 and v[1]=='0' for v in res08.values()) and len(mc2.get('items',[]))==0 and en2=='0' and len(mc3.get('items',[]))==0 and en3=='0' and att3 not in emails and v3row.startswith(v3)
rec('TC-P203-08','P' if ok else 'F',f'3 biến thể MSSV (/me/courses, enrollments, MSSV lưu) {res08}; MSSV người đã vào lớp ({enm}) → kẻ mạo /me/courses={len(mc2.get("items",[]))} enr={en2}; đăng ký trước rồi nạp roster: kẻ mạo lớp={len(mc3.get("items",[]))} enr={en3}, có trong members={att3 in emails}; bản ghi roster theo email nạn nhân: {v3row}')
# ---------- P204-02
st={}
def forgot(em): resetrl(); return call('POST','/auth/forgot-password',None,{'email':em})[0]
def mc(em): return mail_count(em)
c_act=reg('f2a'); c_pen=reg('f2p',verify=False); c_inv,_=invite('f2i',accept=False); c_dis=reg('f2d'); c_no=f'qc2ia-khong-co.{TAG}@example.test'
d=call('GET','/admin/users?q='+c_dis,A)[1]['items'][0]; call('PATCH',f'/admin/users/{d["id"]}',A,{'status':'DISABLED','version':d['version']})
c_sv=f'qc2ia-f2s.{TAG}@edupilot.local'; imp([(c_sv,'SV Roster Hai','AB2022902')])
time.sleep(5); base={e:mc(e) for e in [c_act,c_pen,c_inv,c_dis,c_no,c_sv]}
codes={e:forgot(e) for e in base}; time.sleep(15)
subs={e:newest(e,base[e]) for e in base}
ok=all(v==202 for v in codes.values()) and len(subs[c_act])==1 and 'Đặt lại mật khẩu' in subs[c_act][0] and len(subs[c_pen])==1 and 'Đặt lại mật khẩu' in subs[c_pen][0] and subs[c_inv]==[] and subs[c_dis]==[] and subs[c_no]==[] and subs[c_sv]==[]
rec('TC-P204-02','P' if ok else 'F',f'forgot-password → mã {set(codes.values())}; thư mới: ACTIVE {subs[c_act]}, PENDING_VERIFICATION {subs[c_pen]}, INVITED(GV) {subs[c_inv]}, INVITED(SV) {subs[c_sv]}, DISABLED {subs[c_dis]}, không tồn tại {subs[c_no]}')
# ---------- P204-24 leo vai
k=reg('k24'); kid=uid_of(k); Ktok=[lg(k) for _ in range(2)]; Kacc=Ktok[0][1]['access_token']; ks=call('GET','/me/sessions',Kacc)[1]; ksids=[x['id'] for x in ks.get('items',ks if isinstance(ks,list) else [])]
Aadm=[lg(ADMIN_EMAIL,ADMIN_PW) for _ in range(2)]; At=Aadm[0][1]['access_token']
d1=call('DELETE','/me/sessions/'+ksids[-1],At); d2=call('DELETE','/me/sessions?user_id='+kid,At); d3=call('DELETE','/me/sessions',At,{'user_id':kid},idem=False); d4=call('DELETE','/me/sessions/'+ksids[0]+'?user_id='+kid,At)
alive=psql(f"select count(*) from auth_sessions where user_id='{kid}' and revoked_at is null"); kok=[call('GET','/me/courses',x[1]['access_token'])[0] for x in Ktok]
adm_other=psql(f"select count(*) from auth_sessions s join users u on u.id=s.user_id where u.email='{ADMIN_EMAIL}' and revoked_at is not null and revoked_reason='REVOKED_BY_USER'")
rec('TC-P204-24','P' if d1[0]==404 and d4[0]==404 and alive=='2' and kok==[200,200] else 'F',f'Admin DELETE /me/sessions/{{sid của k}} → {d1[0]}; kèm ?user_id → {d4[0]}; DELETE /me/sessions?user_id=k → {d2[0]}; thân {{"user_id"}} → {d3[0]} (hai lệnh sau chỉ tác động phiên KHÁC của chính Admin: {adm_other} phiên Admin bị thu hồi); phiên của k còn {alive}/2, access của k {kok}')
# ---------- P204-37 nội dung thư
u37=reg('t37'); t0=iso(); NEWP=PWD+'T37'; forgot(u37); m1=mail_to(u37,'Đặt lại',t0[:0],40); tok=link_token(m1,'reset-password'); call('POST','/auth/reset-password',None,{'token':tok,'new_password':NEWP}); m2=mail_to(u37,'đã được đổi',t0[:0],40)
name='QC2IA t37'
exp1=f"Chào {name},\n\nChúng tôi nhận được yêu cầu đặt lại mật khẩu cho tài khoản EduPilot của bạn. Dùng liên kết dưới đây để đặt mật khẩu mới (dùng một lần, có hiệu lực 30 phút):\n\nhttps://localhost/reset-password?token={tok}\n\nNếu không phải bạn yêu cầu, hãy bỏ qua thư này — mật khẩu hiện tại vẫn giữ nguyên."
norm=lambda x:re.sub(r'\s+',' ',x).strip()
t_now=time.gmtime(time.time()+7*3600); at_re=r'(\d\d:\d\d \d\d/\d\d/\d{4})'
at=re.search(at_re,m2['Text']); vn_now=time.strftime('%H:%M %d/%m/%Y',t_now)
exp2_pre=f"Chào {name},\n\nMật khẩu tài khoản EduPilot của bạn vừa được đổi lúc "
exp2_post=" (giờ Việt Nam). Mọi thiết bị khác đã bị đăng xuất.\n\nNếu không phải bạn: đặt lại mật khẩu ngay tại https://localhost/forgot-password và báo cho giảng viên hoặc quản trị viên."
body2=norm(m2['Text']); ok1=norm(exp1) in norm(m1['Text']); ok2=norm(exp2_pre) in body2 and norm(exp2_post) in body2
t_at=time.strptime(at.group(1),'%H:%M %d/%m/%Y') if at else None; dmin=abs(time.mktime(t_at)-time.mktime(t_now))/60 if at else 999
leak=NEWP in m1['Text']+m1.get('HTML','')+m2['Text']+m2.get('HTML','')
rec('TC-P204-37','P' if ok1 and ok2 and dmin<=3 and not leak and 'Asia/Ho_Chi_Minh' not in m2['Text'] and m1['Subject']=='Đặt lại mật khẩu EduPilot' and m2['Subject']=='Mật khẩu EduPilot của bạn đã được đổi' else 'F',f'reset_password đúng lời văn SRS 6.5 (liên kết https://localhost/reset-password?token=…, 30 phút)={ok1}; password_changed đúng lời văn={ok2}, giờ trong thư "{at and at.group(1)}" vs giờ VN hiện tại {vn_now} (lệch {dmin:.1f} phút); mật khẩu mới trong thư (Text+HTML) = {leak}; tiêu đề {m1["Subject"]!r} / {m2["Subject"]!r}')
