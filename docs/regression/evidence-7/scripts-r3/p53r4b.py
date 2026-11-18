from lib3 import *
import datetime as dt, threading
fx=json.load(open('/tmp/qc3e/r4fx.json')); X,Y,S=fx['X'],fx['Y'],fx['S']
a34,a36,a20,a43,a31,a32,a17,a28,a03,a15=S[:10]
ICT=dt.timezone(dt.timedelta(hours=7))
def dayoff(n=0): return (dt.datetime.now(ICT)+dt.timedelta(days=n)).strftime('%Y%m%d')
day=dayoff()
def tabid(n): return str(uuid.uuid5(uuid.NAMESPACE_DNS,n))
def hb(u,c,area='CHAT',tab='t1'): return A(u,'POST','/activity/heartbeat',{'course_id':c,'area':area,'tab_id':tabid(tab)})
def sver(u):
    for _ in range(8):
        g=A(u,'GET','/me/settings')[1]
        if isinstance(g,dict) and 'version' in g: return g['version']
        time.sleep(3)
    raise Exception('no version '+str(g))
def K(u,c,area='CHAT',d=None): return f"ep:study:{c}:{uid(u)}:{d or day}:{area}"
def rget(k): v=rds('GET',k); return int(v) if v else 0
def waitdb(sql,want,timeout=170):
    t0=time.time()
    while time.time()-t0<timeout:
        r=psql(sql)
        if r==want: return r
        time.sleep(4)
    return psql(sql)
def nextwin():
    t=time.time(); time.sleep(30-(t%30)+1.0)
def db(u,c,d=None,area='CHAT'): return psql(f"select coalesce(sum(active_seconds),0) from study_time_daily where student_id='{uid(u)}' and course_id='{c}' and area='{area}' and date='{d or dt.datetime.now(ICT).strftime('%Y-%m-%d')}'")
for u in S+['qc3t','qc3ta','admin']: login(u)
psql(f"update users set tracking_notice_ack_at=now() where email in ({','.join(repr(x+'@edupilot.local') for x in S if x!=a20)})")
psql(f"update users set tracking_notice_ack_at=null where email='{a20}@edupilot.local'")
for u in S: psql(f"delete from user_settings where false")
for u in S: rds('DEL',f'ep:trk:{uid(u)}')
RES={}
import sys
ids=','.join(repr(uid(x)) for x in S)
psql(f"delete from study_time_daily where student_id in ({ids})")
for u in S:
    for k in rds('keys',f'ep:study:*:{uid(u)}:*').split():
        rds('DEL',k); rds('SREM','ep:study:dirty',k); rds('HDEL','ep:study:flushed',k)
    for k in rds('keys',f'ep:hb:*:{uid(u)}*').split(): rds('DEL',k)
    s,_=A(u,'PUT','/me/settings',{'track_study_time':True,'version':sver(u)})
def run(name,fn):
    try: fn()
    except Exception as e:
        import traceback; traceback.print_exc(); RES[name+'_exc']=str(e)

# ---- 02 lược đồ ----
def c02():
    sid=uid(S[0])
    def ins(area,s,d='2026-01-01',n=1):
        v=f"insert into study_time_daily(course_id,student_id,date,area,active_seconds) values('{X}','{sid}','{d}','{area}',{s});"
        return sqlx(v*n)
    r=[ins('X',10),ins('CHAT',-1),ins('CHAT',86401),ins('CHAT',10,n=2),ins('CHAT',86400),ins('CHAT',0)]
    cons=psql("select conname||': '||pg_get_constraintdef(oid) from pg_constraint where conrelid='study_time_daily'::regclass and contype in('c','u') order by 1")
    ir=psql("select string_agg(column_name||' '||data_type||' '||is_nullable,'; ' order by ordinal_position) from information_schema.columns where table_name='insight_reports'")
    ic=psql("select conname||': '||pg_get_constraintdef(oid) from pg_constraint where conrelid='insight_reports'::regclass and contype in('c','f') order by 1")
    ix=psql("select indexdef from pg_indexes where tablename='insight_reports' and indexname like '%course_kind%'")
    ok=[x[0] for x in r]==['23514','23514','23514','23505','OK','OK'] and 'course_id' in cons.split('study_time_daily_key')[1].split('\n')[0] and 'KNOWLEDGE_GAP' in ic and 'payload' in ir and 'created_at DESC' in ix
    rec('TC-P5-03-02',ok,f"area='X' → {r[0][0]}; -1 → {r[1][0]}; 86401 → {r[2][0]}; trùng (course,student,date,area) → {r[3][0]}; biên 86400/0 → {r[4][0]}/{r[5][0]}; ràng buộc: {cons.replace(chr(10),' | ')}; insight_reports cột: {ir}; check/FK: {ic.replace(chr(10),' | ')}; {ix}")
# ---- 20/21 ----
def c20():
    s,_=hb(a20,X); k=rget(K(a20,X)); n=psql(f"select count(*) from study_time_daily where student_id='{uid(a20)}'")
    keys=rds('keys',f'ep:study:*:{uid(a20)}:*'); isnull=psql("select tracking_notice_ack_at is null from users where email='"+a20+"@edupilot.local'")
    rec('TC-P5-03-20',s==204 and k==0 and keys=='' and n=='0' and isnull=='t',f'SV chưa ack (tracking_notice_ack_at IS NULL = {isnull}) gửi nhịp CHAT bằng API → {s}; khoá Redis ep:study:…:{a20}: {keys or "không có"}; study_time_daily = {n} hàng (hook cũng không gửi: xem ui-p503 TC-P5-03-18/20)')
def c21():
    r={}
    for u in ('qc3t','qc3ta','admin'):
        s,_=hb(u,X); r[u]=(s,rds('keys',f'ep:study:*:{uid(u)}:*'))
    tot=rds('keys',f'ep:study:total:{uid("qc3t")}:*')
    rec('TC-P5-03-21',all(v[0]==204 and v[1]=='' for v in r.values()) and tot=='',f'TEACHER/TA/ADMIN gửi nhịp → {[(u,v[0]) for u,v in r.items()]} (204 im lặng, không 403); khoá đếm ep:study:*:{{uid}}:* của cả ba: không có; khoá tổng: không có (hộp thoại không hiện: ui-p503)')
# ---- 28 ----
def c28():
    s,d=A(a28,'GET',f'/courses/{X}/me/study-time')
    s2,d2=A(a28,'GET',f'/courses/{X}/me/study-time?weeks=8')
    rec('TC-P5-03-28',s==200 and d['weeks']==[] and d['this_week_minutes']==0 and d['tracking']=={'enabled':True,'locked':False},f'SV mới (đã ack, chưa nhịp) GET me/study-time → {s} weeks={d["weeks"]} this_week={d["this_week_minutes"]} tracking={d["tracking"]}; weeks=8 → {s2}; TestAllCourseRoutesGuarded (go test -tags testroutes ./internal/integration) PASS xem evidence-7/r4/static-p5.log; route `GET /courses/{{id}}/me/study-time` qua h.Guard(auth.Member) (activityhttp/handler.go:30)')
# ---- 42 ----
def c42():
    s,d=A('admin','GET','/me/settings'); sv=uid(a28)
    s2,d2=A('admin','GET',f'/me/settings?user_id={sv}&student_id={sv}')
    before=psql(f"select coalesce(track_study_time::text,'(chưa có hàng)') from user_settings where user_id='{sv}'")
    s3,d3=A('admin','PUT','/me/settings',{'track_study_time':False,'user_id':sv,'student_id':sv,'version':d['version']})
    after=psql(f"select coalesce(track_study_time::text,'(chưa có hàng)') from user_settings where user_id='{sv}'")
    s4,d4=A('admin','PUT',f'/me/settings?user_id={sv}',{'track_study_time':False,'version':d['version']})
    after2=psql(f"select coalesce(track_study_time::text,'(chưa có hàng)') from user_settings where user_id='{sv}'")
    own=psql(f"select track_study_time from user_settings where user_id='{uid('admin')}'")
    rec('TC-P5-03-42',s==200 and d==d2 and before==after==after2,f'ADMIN GET /me/settings → {s}; kèm ?user_id=<SV>&student_id → giống hệt (={d==d2}); PUT thân có user_id/student_id → {s3}, PUT ?user_id=<SV> → {s4}; track_study_time của SV: trước {before!r}, sau {after!r}/{after2!r} (không đổi); của ADMIN (hàng riêng): {own!r}; khoá GET: {list(d)[:10]}')
# ---- 36 ----
def c36():
    s,d=A(a36,'PUT','/me/settings',{'track_study_time':False,'version':sver(a36)})
    r=hb(a36,Y)[0]; k=rget(K(a36,Y)); st,sd=A(a36,'GET',f'/courses/{Y}/me/study-time')
    rec('TC-P5-03-36',s==200 and r==204 and k==30 and sd['tracking']=={'enabled':True,'locked':True},f'lớp Y (allow_study_time_opt_out=false): PUT /me/settings track_study_time=false → {s}; nhịp CHAT → {r}, ep:study:Y:…:CHAT = {k} (vẫn đo); GET me/study-time tracking={sd["tracking"]}; công tắc UI bật+khoá/tooltip: xem ui-p503 (TC-P5-06-22)')
# ---- 37 ----
def c37():
    tid=uid('qc3t'); r=[]
    for extra in ({'allow_study_time_opt_out':False},{'allow_study_time_opt_out':True}):
        b={'subject_code':'QC3','class_code':'QC3Z'+str(int(time.time())%100000),'name':'QC3 z37','semester':'2026-2027-HK1','capacity':50,'teacher_id':tid,'ta_ids':[],'join_code':None,**extra}
        b.pop('join_code'); s,d=A('admin','POST','/admin/courses',b); r.append((s,(d or {}).get('code') if isinstance(d,dict) else d, (d or {}).get('details') if isinstance(d,dict) else None))
    b={'subject_code':'QC3','class_code':'QC3Z'+str(int(time.time())%100000+7),'name':'QC3 z37b','semester':'2026-2027-HK1','capacity':50,'teacher_id':tid,'ta_ids':[]}
    s,d=A('admin','POST','/admin/courses',b); dflt=psql(f"select allow_study_time_opt_out from courses where id='{d['id']}'") if s in(200,201) else None
    # PUT /admin/courses/{id}
    s2,d2=A('admin','GET',f'/admin/courses/{X}'); 
    pb={k:v for k,v in (d2 or {}).items() if k in('subject_code','class_code','name','semester','capacity','teacher_id','version')} if isinstance(d2,dict) else {}
    pb['allow_study_time_opt_out']=False
    s3,d3=A('admin','PUT',f'/admin/courses/{X}',pb)
    still=psql(f"select allow_study_time_opt_out from courses where id='{X}'")
    rec('TC-P5-03-37',all(x[0]==422 for x in r) and dflt=='t' and (s3 in(422,400) or still=='t'),f'POST /admin/courses có trường allow_study_time_opt_out (false/true) → {[(x[0],x[1]) for x in r]} (từ chối, 422); POST không trường → {s}, courses.allow_study_time_opt_out của lớp mới = {dflt!r} (mặc định true); PUT /admin/courses/{{id}} có trường → {s3} {str(d3)[:80]}, giá trị lớp X sau đó = {still!r}; form «Mở lớp» không có trường: ui-p503')
# ---- 34 ----
def c34():
    u=a34; sid=uid(u); k=K(u,X)
    s0=hb(u,X)[0]; v1=rget(k); d1=waitdb(f"select coalesce(sum(active_seconds),0) from study_time_daily where student_id='{sid}' and course_id='{X}' and area='CHAT' and date=(now() at time zone 'Asia/Ho_Chi_Minh')::date",'30')
    nextwin()
    s1,g1=A(u,'PUT','/me/settings',{'track_study_time':False,'version':sver(u)})
    r2=hb(u,X)[0]; v2=rget(k); kk=rds('keys',f'ep:study:*:{sid}:*'); 
    time.sleep(70); d2=db(u,X)
    sx,sd=A(u,'GET',f'/courses/{X}/me/study-time')
    s3,_=A(u,'PUT','/me/settings',{'track_study_time':True,'version':sver(u)})
    r3=hb(u,X)[0]; v3=rget(k); d3=waitdb(f"select coalesce(sum(active_seconds),0) from study_time_daily where student_id='{sid}' and course_id='{X}' and area='CHAT' and date=(now() at time zone 'Asia/Ho_Chi_Minh')::date",'60')
    rec('TC-P5-03-34',all([s0==204,v1==30,d1=='30',s1==200,r2==204,v2==30,d2=='30',sx==200,sd['tracking']['enabled'] is False,s3==200,r3==204,v3==60,d3=='60']),f'lớp X (cho tắt): nhịp → {s0}, Redis {v1}, DB sau rollup {d1}; PUT track=false → {s1}; nhịp cùng khung → {r2}, Redis vẫn {v2} (không khoá mới), DB sau ≥70 s vẫn {d2}; dữ liệu cũ đọc được: me/study-time → {sx} tracking.enabled={sd["tracking"]["enabled"]} weeks={[w["minutes"] for w in sd["weeks"]]}; bật lại ({s3}) và nhịp NGAY trong cùng khung 30 s (nhịp tắt không chiếm khung) → {r3}, Redis {v3}, DB {d3}')
# ---- 43 ----
def c43():
    u=a43; sid=uid(u)
    sp,_=A(u,'PUT','/me/settings',{'track_study_time':False,'version':sver(u)})
    nextwin()
    rx=hb(u,X)[0]; ry=hb(u,Y)[0]; kx=rget(K(u,X)); ky=rget(K(u,Y))
    dY=waitdb(f"select coalesce(sum(active_seconds),0) from study_time_daily where student_id='{sid}' and course_id='{Y}'",'30'); dX=psql(f"select count(*) from study_time_daily where student_id='{sid}' and course_id='{X}'")
    sx,tx=A(u,'GET',f'/courses/{X}/me/study-time'); sy,ty=A(u,'GET',f'/courses/{Y}/me/study-time')
    st=A(u,'GET','/me/settings')[1]
    A(u,'PUT','/me/settings',{'track_study_time':True,'version':sver(u)})
    nextwin(); rx2=hb(u,X)[0]; kx2=rget(K(u,X))
    rec('TC-P5-03-43',all([sp==200,rx==204,ry==204,kx==0,ky==30,dY=='30',dX=='0',tx['tracking']=={'enabled':False,'locked':False},ty['tracking']=={'enabled':True,'locked':True},kx2==30]),f'SV ghi danh X (cho tắt) + Y (không cho tắt), track_study_time=false: nhịp CHAT cùng khung 30 s cho X rồi Y → {rx}/{ry}; Redis X={kx} (không đo, không chiếm khung), Y={ky}; study_time_daily sau rollup: Y={dY} s, hàng của X={dX}; GET X me/study-time tracking={tx["tracking"]}, Y={ty["tracking"]}; /me/settings nhóm thời gian học: {json.dumps(st.get("study_time",st),ensure_ascii=False)[:200]}; bật lại → nhịp kế ở X → {rx2}, Redis X={kx2} (đo)')
# ---- 31 ----
def c31():
    u=a31; sid=uid(u); tk=f'ep:study:total:{sid}:{day}'
    rds('SET',tk,'43170','EX','86400'); nextwin()
    r1=hb(u,X)[0]; t1=rget(tk); kx=rget(K(u,X))
    nextwin(); r2=hb(u,Y,'THREADS')[0]; t2=rget(tk); ky=rget(K(u,Y,'THREADS'))
    nextwin(); r3=hb(u,X,'CHAT','t9')[0]; t3=rget(tk)
    rec('TC-P5-03-31',all([r1==204,t1==43200,kx==30,r2==204,t2==43200,ky==0,r3==204,t3==43200]),f'ep:study:total:{{uid}}:{day} đặt 43.170: nhịp 1 (X/CHAT) → {r1}, total={t1}, ep:study:X…:CHAT={kx}; nhịp 2 (lớp Y/THREADS, khung sau) → {r2}, total={t2}, khoá Y/THREADS={ky} (không tạo/ không cộng); nhịp 3 (X/CHAT) → {r3}, total={t3}: trần tính trên tổng mọi lớp × khu vực')
# ---- 32 ----
def c32():
    u=a32
    t=time.time(); time.sleep(30-(t%30)+0.5)
    k=K(u,X); before=rget(k); st=[hb(u,X,'CHAT',f'tb{i}')[0] for i in range(6)]; delta=rget(k)-before
    acc=delta//30
    rec('TC-P5-03-32',all(s==204 for s in st) and 6-acc>=2,f'6 nhịp liên tiếp (tab_id khác nhau) ngay sau mốc khung → {st} (không có 429); Redis tăng {delta} ⇒ {acc} nhịp được tính, {6-acc} bị bỏ (≥2)')
# ---- 17/16/33 ----
def c17():
    u=a17; sid=uid(u)
    d_prev=dayoff(-1); d_old=dayoff(-4)
    keys=[K(u,X,'CHAT',d_prev),K(u,X,'CHAT',day),K(u,X,'LIBRARY',d_old),K(u,X,'THREADS',dayoff(-6))]
    for k_,v in zip(keys,(30,30,30,30)):
        rds('SET',k_,str(v),'EX','691200'); rds('SADD','ep:study:dirty',k_)
    # nhịp thật để đo TTL
    nextwin(); hb(u,X,'PRACTICE'); kp=K(u,X,'PRACTICE'); ttl=int(rds('TTL',kp)); tt=int(rds('TTL',f'ep:study:total:{sid}:{day}'))
    time.sleep(75)
    rows=psql(f"select date||':'||area||'='||active_seconds from study_time_daily where student_id='{sid}' order by date,area").replace('\n',', ')
    d_prev_f=psql(f"select date from study_time_daily where student_id='{sid}' and area='CHAT' order by date").replace('\n',',')
    inD=[rds('SISMEMBER','ep:study:dirty',x) for x in keys]; inF=[rds('HEXISTS','ep:study:flushed',x) for x in keys]
    ok17=d_prev_f==f"{dt.datetime.now(ICT)-dt.timedelta(days=1):%Y-%m-%d},{dt.datetime.now(ICT):%Y-%m-%d}" and 0<ttl<=691200 and ttl>691200-120 and 0<tt<=691200
    rec('TC-P5-03-17',ok17,f'khoá tay {d_prev}:CHAT=30 (≈23:59) và {day}:CHAT=30 (≈00:01) → study_time_daily: {rows}; tách đúng hai hàng theo ngày ICT (CHAT: {d_prev_f}); nhịp thật PRACTICE: TTL khoá đếm={ttl}s, TTL khoá tổng={tt}s (≤ 691200 = 8 ngày); nhịp 23:59/00:01 thật không dựng được vì gateway dùng đồng hồ máy: đã đặt khoá tay theo ngày như TC')
    rec('TC-P5-03-33',inD[2:]==['0','0'] and inF[2:]==['0','0'] and inD[:2]==['1','1'],f'sau rollup: khoá ngày {dayoff(-4)} và {dayoff(-6)} (>1 ngày): SISMEMBER dirty={inD[2:]}, HEXISTS flushed={inF[2:]} (đã SREM/HDEL); khoá ngày gần (hôm qua/hôm nay): dirty={inD[:2]}, flushed={inF[:2]}; DB đã ghi các hàng khoá cũ: {rows}')
    # 16
    kk=K(u,X,'CHAT',day)
    # a17 CHAT hôm nay =30. tăng lên 90, rollup, rồi DEL, rồi đặt 30
    rds('SET',kk,'90','EX','691200'); rds('SADD','ep:study:dirty',kk); v1=waitdb(f"select coalesce(sum(active_seconds),0) from study_time_daily where student_id='{sid}' and area='CHAT' and date=(now() at time zone 'Asia/Ho_Chi_Minh')::date",'90')
    rds('DEL',kk); time.sleep(75); v2=db(u,X)
    rds('SET',kk,'30','EX','691200'); rds('SADD','ep:study:dirty',kk); time.sleep(75); v3=db(u,X)
    rec('TC-P5-03-16',v1=='90' and v2=='90' and v3=='90',f'DB giữ X={v1}; $RDS DEL khoá → rollup (≥75 s) → DB={v2} (không giảm, không âm); đặt lại khoá 30 (Redis "mất" tích luỹ rồi đếm lại) → rollup → DB={v3} (GREATEST giữ 90)')
for n,f in (('c20',c20),('c21',c21),('c36',c36)): run(n,f)
ths=[threading.Thread(target=run,args=(n,f)) for n,f in (('c34',c34),('c43',c43),('c31',c31),('c32',c32),('c17',c17))]
[t.start() for t in ths]; [t.join() for t in ths]
print(RES); save('p53r4b')
