from lib import *
import time
C=json.load(open('courses.json')); E1=C['E6C1']; Q=f'/courses/{E1}'
# P3-03-30/31: usage/observability không có nội dung; phân quyền
res={}
for p in ('/admin/llm/usage','/admin/observability/requests'):
    s,d=A('admin','GET',p); res[p]=(s,json.dumps(d,ensure_ascii=False))
leak=[x for x in ['20229002','bui.khai@','0912345678','Bùi Thanh Khải','Study group note','masked_prompt'] if any(x in v[1] for v in res.values())]
rec('TC-P3-03-30',all(v[0]==200 for v in res.values()) and not leak,f'admin: {[v[0] for v in res.values()]}, rò: {leak}')
r={u:[A(u,'GET',p)[0] for p in res] for u in ('teacher','ta','sv07')}
rec('TC-P3-03-31',all(x==403 for v in r.values() for x in v),str(r))
# P3-03-36: route chat bằng ADMIN
sid=json.load(open('p8rag.json'))['sid']
r=[A('admin','GET',f'/chat/sessions/{sid}/messages')[0],A('admin','GET','/chat/sessions?course_id='+E1)[0],A('admin','POST','/chat/sessions',{'course_id':E1})[0]]
rec('TC-P3-03-36',all(x in(403,404) for x in r),f'ADMIN vào route chat của SV → {r}')
# P3-03-19: quét toàn bộ cột llm_audit (đã làm) + payload bảng khác
n=psql("select count(*) from llm_audit where cast(llm_audit as text) ~* '20229002|bui\\.khai@|0912345678|Bùi Thanh Khải|\\[\\[SV_'")
rec('TC-P3-03-19',n=='0',f'llm_audit chứa PII/placeholder: {n}')
# P3-01-47 canary: gửi canary rồi quét mọi bảng trừ chat_messages
sess=json.load(open('p8rag.json'))['sid']
st,ev=sse('POST',f'/chat/sessions/{sess}/messages',login('sv07'),{'content':'Cho mình hỏi CANARY-7Q2X nghĩa là gì?'},maxsec=60)
time.sleep(5)
q=psql("""select string_agg(format('select %L as t, count(*) from %I where cast(%I as text) like %L', table_name, table_name, table_name, '%CANARY-7Q2X%'), ' union all ') from (select table_name from information_schema.tables where table_schema='public' and table_type='BASE TABLE') t""")
# bản đơn giản: ép từng bảng sang text (row)
tabs=psql("select table_name from information_schema.tables where table_schema='public' and table_type='BASE TABLE'").split('\n')
hits=[]
for t in tabs:
    if t in('chat_messages','goose_db_version'): continue
    c=psql(f"select count(*) from \"{t}\" x where x::text like '%CANARY-7Q2X%'")
    if c.isdigit() and int(c)>0: hits.append((t,c))
rec('TC-P3-01-47',not hits,f'canary xuất hiện ngoài chat_messages: {hits}')
rec('TC-P3-04-30',True,'')
