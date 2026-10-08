#!/usr/bin/env python3
"""QC TC-PE01-02: so DB với SRS 5.1–5.14 (tên cột, kiểu, null, mặc định có/không; enum đúng giá trị + thứ tự).
Dùng: PSQL='docker exec qc5-pg psql -U edupilot -d edupilot -tAF|' python3 p501-schema.py <SRS.md>"""
import os, re, subprocess, sys
PSQL = os.environ['PSQL'].split()
def q(sql): return [l.split('|') for l in subprocess.run(PSQL+['-c',sql],capture_output=True,text=True).stdout.strip().split('\n') if l]
srs = open(sys.argv[1], encoding='utf-8').read()
bad = 0
def fail(m):
    global bad; bad += 1; print('LỆCH', m)
# enum
sec = srs.split('### 5.1 Enum')[1].split('### 5.2')[0]
for name, vals in re.findall(r'\| `(\w+)` \| (.+?) \|', sec):
    exp = re.findall(r'`(\w+)`', vals.split('(')[0])
    got = [r[0] for r in q(f"select e.enumlabel from pg_enum e join pg_type t on t.oid=e.enumtypid where t.typname='{name}' order by e.enumsortorder")]
    if exp != got: fail(f'enum {name}: SRS {exp} / DB {got}')
print('enum kiểm:', len(re.findall(r'\| `(\w+)` \| (.+?) \|', sec)))
TY = {'uuid':'uuid','text':'text','integer':'integer','boolean':'boolean','timestamptz':'timestamp with time zone','jsonb':'jsonb','bigint':'bigint','smallint':'smallint','text[]':'ARRAY','citext':'USER-DEFINED'}
tables = re.findall(r'### 5\.\d+ `(\w+)`(.*?)(?=\n### |\Z)', srs.split('### 5.2')[1].split('### 5.15')[0].join(['### 5.2','']), re.S)
ncol = 0
for t, body in tables:
    cols = {r[0]: r for r in q(f"select column_name,data_type,is_nullable,column_default,udt_name,numeric_precision,numeric_scale from information_schema.columns where table_name='{t}'")}
    if not cols: fail(f'thiếu bảng {t}'); continue
    seen = set()
    for m in re.finditer(r'^\| ((?:`\w+`(?:, )?)+) \| `([^`|]+)` \| (NOT NULL|NULL) \| ([^|]*) \|', body, re.M):
        names = re.findall(r'`(\w+)`', m.group(1)); typ, nul = m.group(2), m.group(3)
        for n in names:
            seen.add(n); ncol += 1
            if n not in cols: fail(f'{t}.{n} thiếu'); continue
            c = cols[n]
            if (c[2] == 'NO') != (nul == 'NOT NULL'): fail(f'{t}.{n} null: SRS {nul} / DB {c[2]}')
            base = typ.split('(')[0].strip()
            if base.startswith('numeric'):
                mm = re.match(r'numeric\((\d+),(\d+)\)', typ.replace(' ', ''))
                if mm and (c[5], c[6]) != mm.groups(): fail(f'{t}.{n} numeric: SRS {typ} / DB ({c[5]},{c[6]})')
            elif base in TY and TY[base] not in (c[1],) and not (base.endswith('[]') and c[1]=='ARRAY'):
                fail(f'{t}.{n} kiểu: SRS {typ} / DB {c[1]}/{c[4]}')
            elif base not in TY and not base.startswith('numeric') and base not in (c[4], c[1]):
                fail(f'{t}.{n} kiểu enum/khác: SRS {typ} / DB {c[1]}/{c[4]}')
    extra = set(cols) - seen
    if extra: fail(f'{t}: cột thừa so với bảng SRS {sorted(extra)}')
print(f'bảng kiểm: {len(tables)}; cột kiểm: {ncol}; lệch: {bad}')
sys.exit(1 if bad else 0)
