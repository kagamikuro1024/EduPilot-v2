# TC-P101-11 / TC-P104-32: 20.000 dòng llm_audit giả trong DB tạm smk3_p101, EXPLAIN 3 truy vấn usage (llm.sql) + thời gian
import subprocess,json,re
def sq(q,db='smk3_p101'):
    r=subprocess.run(['docker','exec','-i','ep-r-postgres-1','psql','-U','edupilot','-d',db,'-At','-v','ON_ERROR_STOP=1'],input=q,capture_output=True,text=True); return r.stdout+r.stderr
print(sq("""truncate llm_audit cascade;
insert into llm_audit(task,lane,provider,model,tokens_in,tokens_out,latency_ms,status,degraded,trace_id,course_id,created_at,cost_est)
select (array['CHAT','CLASSIFY','UTILITY','GRADING','QUESTION_GEN','INSIGHT'])[1+i%6],'INTERACTIVE','p','m',100+i%50,50+i%30,10+i%900,
 (array['ok','ok','ok','error','timeout'])[1+i%5], i%7=0,'t'||i,
 case when i%4=0 then null else (select ('00000000-0000-7000-8000-00000000000'||(i%3))::uuid) end,
 now()-((i%90)||' days')::interval-((i%1440)||' minutes')::interval, (i%17)*0.0123
from generate_series(1,20000) i;
analyze llm_audit; select count(*) from llm_audit;"""))
C="'00000000-0000-7000-8000-000000000001'::uuid"
W="created_at >= now()-interval '7 days' and created_at < now()"
Qs={
'theo task (course_id=NULL, 7 ngày)':f"select task as key,count(*),sum(tokens_in),sum(cost_est),percentile_cont(0.5) within group (order by latency_ms),count(*) filter (where degraded) from llm_audit where {W} group by task order by task",
'theo day (course_id=NULL, 7 ngày)':f"select to_char(created_at at time zone 'Asia/Ho_Chi_Minh','YYYY-MM-DD') k,count(*),sum(tokens_in) from llm_audit where {W} group by 1 order by 1",
'theo course_id (task, 30 ngày)':f"select task,count(*) from llm_audit where created_at >= now()-interval '30 days' and created_at < now() and course_id={C} group by task",
'theo task, từng task 7 ngày':f"select count(*) from llm_audit where task='GRADING' and {W}",
}
def scans(n,acc):
    if 'Relation Name' in n or 'Index Name' in n: acc.append((n['Node Type'],n.get('Relation Name'),n.get('Index Name')))
    for c in n.get('Plans',[]): scans(c,acc)
bad=0
for k,q in Qs.items():
    out=sq("explain (analyze, format json) "+q)
    j=json.loads(out); acc=[]; scans(j[0]['Plan'],acc)
    seq=[a for a in acc if a[0]=='Seq Scan' and a[1]=='llm_audit']
    print(k,'→',acc,'| exec ms',j[0]['Execution Time'],'| SEQ SCAN' if seq else '| không Seq Scan'); bad+=bool(seq)
print('SEQ',bad)
print(sq("select count(*) from llm_audit where created_at >= now()-interval '7 days'"))
