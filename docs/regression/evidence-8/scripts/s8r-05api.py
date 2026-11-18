import importlib.util,os,json,time,uuid
spec=importlib.util.spec_from_file_location('lib','s8flow-lib.py'); L=importlib.util.module_from_spec(spec); spec.loader.exec_module(L)
PART=L.REPO+'/docs/regression/evidence-8/part-s8r.tsv'
def T(tc,st,n):
    n=' '.join(str(n).split())[:400]; open(PART,'a').write(f"{tc}\t{st}\t{n}\n"); print(st,tc,n[:200],flush=True)
C=open('/tmp/qc2-s8f/QD8H2.id').read().strip(); Q=f'/courses/{C}'
# 05-16 / 05-17
s,d=L.api('gv.s8f','POST',Q+'/gradebook/finalize',{}); T('TC-P6-05-16','P' if s==409 and d.get('code')=='SCHEME_NOT_CONFIRMED' else 'F',f'POST finalize lớp DRAFT_READY (chưa CONFIRMED) → {s} {d.get("code")}')
s1,d1=L.api('gv.s8f','POST',Q+'/gradebook/finalize',{},key=False)
ta=L.api('ta','POST',Q+'/gradebook/finalize',{})[0]; sv=L.api('sv.gioi','POST',Q+'/gradebook/finalize',{})[0]
T('TC-P6-05-17','P' if s1 in (400,422) and ta==403 and sv==403 else 'F',f'thiếu Idempotency-Key → {s1} {d1.get("code") if isinstance(d1,dict) else ""}; TA → {ta}; STUDENT → {sv}')
# 05-25/26 Today theo vai
def has(u):
    s,d=L.api(u,'GET','/me/today'); return s,[a['title'] for a in (d.get('actions') or []) if a['kind']=='GRADE_SCHEME_UNCONFIRMED' and 'QD8H2' in a['title']] if isinstance(d,dict) else []
tas,tai=has('ta'); T('TC-P6-05-25','P' if tas==200 and len(tai)==1 else 'F',f'TA /me/today: {tas}, mục GRADE_SCHEME_UNCONFIRMED của QD8H2: {tai}')
r=[(u,)+has(u) for u in ('sv.gioi','admin')]
T('TC-P6-05-26','P' if all(x[1]==200 and not x[2] for x in r) else 'F',f'STUDENT/ADMIN: {[(x[0],x[1],len(x[2])) for x in r]} (không có mục GRADE_SCHEME_UNCONFIRMED)')
# 05-28 lớp lưu trữ
L.psql(f"update courses set status='ARCHIVED' where id='{C}'")
for k in L.psql("select 1").split(): pass
os.system("docker exec ep-r-redis-1 sh -c \"redis-cli --scan --pattern 'ep:today:*' | xargs -r redis-cli del\" >/dev/null")
ts,ti=has('gv.s8f'); L.psql(f"update courses set status='ACTIVE' where id='{C}'"); os.system("docker exec ep-r-redis-1 sh -c \"redis-cli --scan --pattern 'ep:today:*' | xargs -r redis-cli del\" >/dev/null")
T('TC-P6-05-28','P' if ts==200 and not ti else 'F',f'đổi QD8H2 sang ARCHIVED (SQL) rồi xoá cache Hôm nay: {ts}, mục công thức của lớp lưu trữ: {ti}; trả về ACTIVE')
