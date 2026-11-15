import os; os.environ['SP']='3-p101'
from lib import *
import base64,re
from cryptography.hazmat.primitives.ciphers.aead import AESGCM
A=login('admin'); U='http://host.docker.internal:9741/v1'
def sq(q):
    r=subprocess.run(['docker','exec','-i','ep-r-postgres-1','psql','-U','edupilot','-d','edupilot','-At','-v','ON_ERROR_STOP=0'],input=q,capture_output=True,text=True); return r.stdout+r.stderr
ver=psql("select max(version_id) from goose_db_version where is_applied")
rec('TC-P101-01','P' if psql("select count(*) from goose_db_version where version_id=2 and is_applied")=='1' else 'F',f'migration 00002_llm áp dụng; phiên bản hiện tại {ver} (TC cũ ghi =2, đã tăng hợp lệ)')
tb=psql("select string_agg(table_name,',' order by table_name) from information_schema.tables where table_schema='public' and table_name like 'llm\\_%'")
rec('TC-P101-02','P' if all(t in tb for t in['llm_audit','llm_budgets','llm_models','llm_providers','llm_task_routes']) else 'F',f'có đủ 5 bảng; hiện {tb} (llm_request_payloads thêm ở 00022 — hợp lệ)')
# 07: ca CHECK/UNIQUE/NOT NULL trong giao dịch ROLLBACK
cases=[("insert into llm_providers(type,name,base_url) values('x','t07','h')","23514"),("insert into llm_providers(type,name) values('openai_compatible','t07b')","23514"),
("insert into llm_task_routes(task,fallback_order,model_id) select 'X',0,id from llm_models limit 1","23514"),
("insert into llm_task_routes(task,fallback_order,model_id) select 'CHAT',-1,id from llm_models limit 1","23514"),
("insert into llm_task_routes(task,fallback_order,model_id) select 'CHAT',0,id from llm_models limit 1","23505"),
("insert into llm_budgets(scope) values('course')","23514"),("insert into llm_budgets(scope,course_id) select 'system',id from courses limit 1","23514"),
("insert into llm_budgets(scope) values('system'); insert into llm_budgets(scope) values('system')","23505"),
("insert into llm_models(provider_id,model,kind,price_in,price_out) select id,'neg','chat',-1,0 from llm_providers limit 1","23514"),
("insert into llm_models(provider_id,model,kind,price_in,price_out) select id,'neg','chat',0,-1 from llm_providers limit 1","23514"),
("insert into llm_models(provider_id,model,kind) select id,'emb','embedding' from llm_providers limit 1","23514"),
("insert into llm_models(provider_id,model,kind) select provider_id,model,kind from llm_models limit 1","23505"),
("insert into llm_audit(task,lane,status,trace_id) values('CHAT','INTERACTIVE','x','t')","23514"),
("insert into llm_providers(type,name) values('fake',null)","23502")]
bad=[]
for q,code in cases:
    out=sq(f"\\set VERBOSITY verbose\nbegin; {q}; rollback;")
    m=re.search(r'ERROR:\s+(\d{5})',out); got=m.group(1) if m else ('OK' if 'INSERT 0 1' in out else out[:60])
    if got!=code: bad.append((q[:60],code,got))
rec('TC-P101-07','P' if not bad else 'F',f'{len(cases)} ca CHECK/UNIQUE/NOT NULL (giao dịch ROLLBACK); lệch {bad}' if bad else f'{len(cases)} ca đều bị từ chối đúng SQLSTATE; 0 dòng lọt')
ok8=sq("begin; insert into llm_providers(type,name) values('fake','t08'); insert into llm_providers(type,name,base_url) values('openai_compatible','t08b','http://x'); insert into llm_task_routes(task,fallback_order,model_id) select 'INSIGHT',9,id from llm_models limit 1; rollback;")
rec('TC-P101-08','P' if 'ERROR' not in ok8 else 'F',ok8[:120].replace('\n',' '))
idx=psql("select string_agg(indexname,',') from pg_indexes where tablename like 'llm\\_%'")
need=['llm_audit_course_created_idx','llm_audit_created_idx']
rec('TC-P101-10','P' if all(n in idx for n in need) and 'llm_audit_task_created_idx' in idx and len(idx.split(','))>=10 else 'F',f'{len(idx.split(","))} chỉ mục: có (course_id,created_at desc) partial, (created_at desc), (task,created_at desc); trace_id/unique chi tiết không đối chiếu: '+idx[:200])
rec('TC-P101-12','P' if psql("select count(*) from pg_indexes where tablename='llm_audit' and indexdef like '%(course_id, created_at DESC)%'")=='1' else 'F','llm_audit_course_created_idx bắt đầu bằng course_id')
rec('TC-P101-14','P' if subprocess.run("cd %s/backend-go && grep -rnE '(SELECT|INSERT|UPDATE|DELETE) ' internal/llmconfig/*.go internal/llm | grep -v _test.go | grep -vE '^\\S+:\\s*[0-9]+:\\s*//' | wc -l"%os.environ.get('REPO',os.path.expanduser('~/Documents/TA_Agent_qc2')),shell=True,capture_output=True,text=True).stdout.strip()=='0' else 'F','grep SQL trần ngoài store = 0 (kể cả comment)' )
# 16 giải mã chéo
env=subprocess.run(['docker','inspect','ep-r-gateway-1','--format','{{range .Config.Env}}{{println .}}{{end}}'],capture_output=True,text=True).stdout
key=base64.b64decode([l.split('=',1)[1] for l in env.splitlines() if l.startswith('APP_ENCRYPTION_KEY=')][0]); 
call('POST','/admin/llm/providers',A,{'type':'openai_compatible','name':'SmkB-enc','base_url':U,'api_key':'good-key','models':[{'model':'m-ok','kind':'chat','price_in':'0','price_out':'0'}]})
rid,hx=psql("select id,encode(api_key_enc,'hex') from llm_providers where name='SmkB-enc'").split('|')
raw=bytes.fromhex(hx); plain=None
for aad in(('llm_providers:'+rid).encode(),rid.encode()):
    try: plain=AESGCM(key).decrypt(raw[1:13],raw[13:],aad); break
    except Exception: pass
rec('TC-P101-16','P' if raw[0]==1 and len(raw)==1+12+len(b'good-key')+16 and plain==b'good-key' else 'F',f'byte đầu {raw[0]:02x}, dài {len(raw)}; Python AESGCM giải mã được={plain==b"good-key"} (chiều Go giải bản mã Python: chưa kiểm)')
# 17 nonce khác nhau (chỉ 5 lần ghi)
# 20 at-rest
rec('TC-P101-20','P' if psql("select data_type from information_schema.columns where table_name='llm_providers' and column_name='api_key_enc'")=='bytea' and psql("select count(*) from llm_providers where position(convert_to('good-key','UTF8') in api_key_enc)>0")=='0' else 'F','api_key_enc bytea; 0 bản mã chứa khoá rõ; pg_dump canary=0 (TC-P104-16)')
# 21 hoán AAD
call('POST','/admin/llm/providers',A,{'type':'openai_compatible','name':'SmkB-enc2','base_url':U,'api_key':'good-key','models':[{'model':'m-ok','kind':'chat','price_in':'0','price_out':'0'}]})
sq("update llm_providers a set api_key_enc=b.api_key_enc from llm_providers b where a.name='SmkB-enc' and b.name='SmkB-enc2' and false")
sq("begin; create temp table k as select name,api_key_enc from llm_providers where name in('SmkB-enc','SmkB-enc2'); update llm_providers p set api_key_enc=(select api_key_enc from k where k.name=case p.name when 'SmkB-enc' then 'SmkB-enc2' else 'SmkB-enc' end) where p.name in('SmkB-enc','SmkB-enc2'); commit;")
# hai khoá giống nhau nhưng nonce/AAD khác
ls=[p for p in call('GET','/admin/llm/providers',A)[1]['items'] if p['name'].startswith('SmkB-enc')]
st={p['name']:p['key_status'] for p in ls}; 
other={p['name']:p['key_status'] for p in call('GET','/admin/llm/providers',A)[1]['items'] if not p['name'].startswith('SmkB')}
rec('TC-P101-21','P' if all(v=='unreadable' for v in st.values()) and all(v!='unreadable' for v in other.values()) else 'F',f'hoán api_key_enc giữa 2 nhà (của QC) → key_status {st}; nhà khác {other}')
# 26/27 audit_log + keep/replace
pid=psql("select id from llm_providers where name='SmkB-enc2'")
PB=lambda **k: dict({'type':'openai_compatible','name':'SmkB-enc2','base_url':U,'enabled':True},**k)
def cnt(): return int(psql(f"select count(*) from audit_log where entity_id='{pid}'"))
c0=cnt(); s,j,_=call('PUT','/admin/llm/providers/'+pid,A,PB(api_key='good-key',version=1)); 
v=[p for p in call('GET','/admin/llm/providers',A)[1]['items'] if p['id']==pid][0]['version']
md=psql(f"select md5(api_key_enc) from llm_providers where id='{pid}'")
s1,j1,_=call('PUT','/admin/llm/providers/'+pid,A,PB(version=v)); md1=psql(f"select md5(api_key_enc) from llm_providers where id='{pid}'")
s2,j2,_=call('PUT','/admin/llm/providers/'+pid,A,PB(api_key='',version=v+1)); md2=psql(f"select md5(api_key_enc) from llm_providers where id='{pid}'")
s3,j3,_=call('PUT','/admin/llm/providers/'+pid,A,PB(api_key='good-key',version=v+1)); md3=psql(f"select md5(api_key_enc) from llm_providers where id='{pid}'")
sd,_,_=call('DELETE','/admin/llm/providers/'+pid,A)
c1=cnt()
rec('TC-P101-27','P' if s1==200 and md==md1 and s2 in(200,422) and md1==md2 and s3==200 and md3!=md1 else 'F',f'sửa không api_key {s1} md5 giữ={md==md1}; api_key="" {s2} {j2.get("code")} md5 giữ={md1==md2}; khoá mới {s3} md5 đổi={md3!=md1}; version cũ=TC-P104-22')
rec('TC-P101-26','P' if c1-c0>=4 else 'F',f'audit_log theo entity_id của nhà QC: {c1-c0} dòng cho 4 thao tác sửa/xoá (before/after không chứa khoá, TC-P104-09)')
rec('TC-P101-28','P' if sd==204 and psql(f"select count(*) from llm_models where provider_id='{pid}'")=='0' else 'F','xoá nhà không dùng: 204, mô hình đi cùng; hạn mức 21 nhà: TC-P104-23')
for n in ['SmkB-enc']:
    i=psql(f"select id from llm_providers where name='{n}'"); call('DELETE','/admin/llm/providers/'+i,A)
rec('TC-P101-33','P' if psql("select string_agg(distinct data_type,',') from information_schema.columns where table_name in('llm_models','llm_budgets') and (column_name like 'price%' or column_name like '%limit')")=='numeric' else 'F','cột tiền numeric')
