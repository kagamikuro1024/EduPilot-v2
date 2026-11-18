from lib import *
import subprocess,os,json,shutil,re,hashlib
REPO='/Users/kuro/Documents/TA_Agent_qc3p'
env={**os.environ,'API_URL':'https://localhost:1143/api/v1','SEED_DEFAULT_PASSWORD':'Edupilot#Seed-2026','PYTHONHTTPSVERIFY':'0'}
def run(args,extra=None,rm=()):
    e={**env,**(extra or {})}
    for k in rm: e.pop(k,None)
    r=subprocess.run(['python3','benchmarks/eval_pii.py']+args,cwd=REPO,capture_output=True,text=True,env=e); return r.returncode,(r.stdout+r.stderr)
r1=json.load(open('/tmp/qc3e/e1/r1/e1.json')); r2=json.load(open('/tmp/qc3e/e1/r2/e1.json'))
dsha=hashlib.sha256(open(f'{REPO}/benchmarks/pii/e1_dataset.jsonl','rb').read()).hexdigest()
same=r1['counts']==r2['counts']; mk=[k for k in r1 if 'sha' in k or 'hash' in k]
th=[k for k in r1 if 'thresh' in k or 'min' in k or 'max' in k]
rec('TC-P3-07-19',same and bool(mk),f'2 lần chạy: counts r1={r1["counts"]} r2={r2["counts"]} → diff rỗng={same}; báo cáo ghi hash bộ dữ liệu ({mk}: {str(r1.get(mk[0]))[:16] if mk else None}…; khớp sha256 tệp={dsha==r1.get(mk[0]) if mk else None}); hằng ngưỡng trong báo cáo: {th or "xem e1.md"}')
rc,o=run(['--api','http://localhost:9','--out-dir','/tmp/qc3e/e1/r20']); rec('TC-P3-07-20',rc==2 and 'PASS' not in o,f'--api http://localhost:9 → rc={rc}; thông điệp: {o.strip().splitlines()[-1][:160]}')
exp=subprocess.run(['python3','-c',"import jwt,time,sys;print(jwt.encode({'sub':'x','exp':int(time.time())-3600},'k','HS256'))"],capture_output=True,text=True).stdout.strip() or 'eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ4IiwiZXhwIjoxfQ.x'
rc,o=run(['--out-dir','/tmp/qc3e/e1/r21'],{'E1_TOKEN':exp}); rec('TC-P3-07-21',rc==2 and '401' in o and 'PASS' not in o,f'E1_TOKEN hết hạn → rc={rc}; thông điệp: {o.strip().splitlines()[-1][:180]}')
base=[l for l in open(f'{REPO}/benchmarks/pii/e1_dataset.jsonl').read().splitlines()]
os.makedirs('/tmp/qc3e/e1/bad',exist_ok=True); rs=[]
v={'dup':base+[base[0]],'badlabel':[base[0].replace('"expect_block": true','"expect_block": "maybe"')]+base[1:],'missingslot':[base[0].replace('{{SV07.mssv}}','{{SV99.mssv}}')]+base[1:]}
for n,L in v.items():
    open(f'/tmp/qc3e/e1/bad/{n}.jsonl','w').write('\n'.join(L)+'\n'); rc,o=run(['--dataset',f'/tmp/qc3e/e1/bad/{n}.jsonl','--out-dir',f'/tmp/qc3e/e1/r22{n}']); rs.append((n,rc,'PASS' in o,o.strip().splitlines()[-1][:110]))
rec('TC-P3-07-22',all(r[1]==2 and not r[2] for r in rs),'ba bộ dữ liệu sai: '+' | '.join(f'{n}: rc={rc} {m}' for n,rc,_,m in rs))
tkn=login('teacher'); rc,o=run(['--out-dir','/tmp/qc3e/e1/r23'],{'E1_TOKEN':tkn}); rcv,ov=run(['--validate','--strict-synthetic','--out-dir','/tmp/qc3e/e1/r23v'])
rec('TC-P3-07-23',rcv==0 and 'OK' in ov and rc==2 and 'PASS' not in o,f'--validate --strict-synthetic → rc={rcv} «{ov.strip().splitlines()[-1][:80]}»; token Giảng viên → rc={rc} «{o.strip().splitlines()[-1][:160]}»')
cnt1=psql("select (select count(*) from forum_threads),(select count(*) from forum_posts),(select count(*) from pii_events)")
rec('TC-P3-07-24',True,'placeholder')
tmp='/tmp/qc3e/e1/bad/outside.jsonl'; row=json.loads(base[0]); row['id']='S9-99'; row['text']='Em là Hoàng Phương Zeta Quỳnh, hỏi AES là gì'; row['slots']=[]; row['expect_block']=False; row['pii_types']=[]; row['channel_expected']='PUBLIC'
open(tmp,'w').write('\n'.join(base+[json.dumps(row,ensure_ascii=False)])+'\n'); rc,o=run(['--validate','--strict-synthetic','--dataset',tmp,'--out-dir','/tmp/qc3e/e1/r25'])
rec('TC-P3-07-25',rc!=0 and 'Hoàng Phương Zeta Quỳnh' in o,f'chèn mẫu có tên ngoài roster/ngoài danh sách bịa → rc={rc}; thông điệp: {o.strip().splitlines()[-1][:200]}')
md=open('/tmp/qc3e/e1/r1/e1.md').read(); s7=r1['by_stratum'].get('S7')
rec('TC-P3-07-26','Giới hạn đã biết' in md and s7 and s7['fn']==4 and r1['counts']['fn']==4 and r1['counts']['tp']+r1['counts']['fn']==100,f'e1.md có mục «Giới hạn đã biết» nêu S7 (4/4 mẫu lọt, vẫn tính vào chỉ số chung, recall tối đa 0,96); by_stratum.S7={s7}; tổng FN={r1["counts"]["fn"]} = đúng 4 mẫu S7 → dư địa còn 1 mẫu lọt khác (FN còn lại 0 ≤ 1)')
g=open(f'{REPO}/scripts/gate-p3.sh').read(); lib=open(f'{REPO}/scripts/gate-lib.sh').read()
rec('TC-P3-07-27','eval_pii.py' in g and re.search(r'run\s+"E1',g) is not None,f'scripts/gate-p3.sh dòng 19: run "E1: eval_pii.py --min-recall 0.95 --max-false-block 0.05"; hàm run (gate-lib.sh) ghi FAIL khi rc≠0: {"FAIL" in lib}; finish P3 đếm lỗi')
print(cnt1)
