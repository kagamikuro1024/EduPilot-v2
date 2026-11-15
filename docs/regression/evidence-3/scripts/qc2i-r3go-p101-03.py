# TC-P101-03: so cột 5 bảng llm_* (DB tạm ở v2 + DB ep-r hiện hành) với DDL ở SRS 5.2
import re,subprocess,sys
sql=open('docs/specs/FEAT-llm-gateway/SRS.md').read().split('### 5.2')[1].split('```sql')[1].split('```')[0]
TYPE={'uuid':'uuid','text':'text','bytea':'bytea','boolean':'boolean','integer':'integer','timestamptz':'timestamp with time zone','jsonb':'jsonb','numeric':'numeric'}
exp={}
for m in re.finditer(r'create table (\w+) \((.*?)\n\);',sql,re.S):
    t=m.group(1); exp[t]={}
    for ln in m.group(2).split('\n'):
        ln=ln.split('--')[0].strip().rstrip(',')
        mm=re.match(r'(\w+) (uuid|text|bytea|boolean|integer|timestamptz|jsonb|numeric)(?:\(\d+,\d+\))?(.*)',ln)
        if not mm or mm.group(1) in('constraint',): continue
        rest=mm.group(3)
        exp[t][mm.group(1)]=(TYPE[mm.group(2)],'NO' if ('not null' in rest or 'primary key' in rest) else 'YES','default' in rest)
def dump(db):
    out=subprocess.run(['docker','exec','-i','ep-r-postgres-1','psql','-U','edupilot','-d',db,'-At','-F','|','-c',"select table_name,column_name,data_type,is_nullable,column_default is not null from information_schema.columns where table_schema='public' and table_name in ('llm_providers','llm_models','llm_task_routes','llm_audit','llm_budgets')"],capture_output=True,text=True).stdout
    d={}
    for l in out.strip().split('\n'):
        t,c,ty,nu,df=l.split('|'); d.setdefault(t,{})[c]=(ty,nu,df=='t')
    return d
def fkq(db):
    return subprocess.run(['docker','exec','-i','ep-r-postgres-1','psql','-U','edupilot','-d',db,'-At','-c',"select table_name||'->'||ccu.table_name from information_schema.table_constraints tc join information_schema.constraint_column_usage ccu using(constraint_name,constraint_schema) where tc.constraint_type='FOREIGN KEY' and tc.table_name like 'llm\\_%'"],capture_output=True,text=True).stdout.strip()
bad=0
for db in sys.argv[1:]:
    got=dump(db); print('==',db)
    for t,cols in exp.items():
        g=got.get(t,{})
        for c,(ty,nu,df) in cols.items():
            if c not in g: print('THIẾU',t,c); bad+=1; continue
            if g[c][0]!=ty or g[c][1]!=nu: print('LỆCH',t,c,'SRS',ty,nu,'DB',g[c]); bad+=1
            if df and not g[c][2]: print('THIẾU default',t,c); bad+=1
        extra=set(g)-set(cols)
        if extra: print('cột thêm ngoài SRS',t,sorted(extra))
    ids=[(t,got[t]['id']) for t in exp if 'id' in got.get(t,{})]
    print('cột SRS',sum(len(v) for v in exp.values()),'| id uuid',all(i[1][0]=='uuid' and i[1][2] for i in ids),'| created_at/updated_at có:',[ (t,'created_at' in got[t],'updated_at' in got[t]) for t in exp if t in got and t!='llm_audit'])
    print('FK llm_*:',fkq(db).replace('\n',', '))
print('BAD',bad)
