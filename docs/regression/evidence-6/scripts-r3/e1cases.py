from lib import *
import subprocess,os,json,hashlib,shutil,re
REPO='/Users/kuro/Documents/TA_Agent_qc3p'
env={**os.environ,'API_URL':'https://localhost:1143/api/v1','SEED_DEFAULT_PASSWORD':'Edupilot#Seed-2026','PYTHONHTTPSVERIFY':'0'}
def run(args,extra=None):
    e={**env,**(extra or {})}
    r=subprocess.run(['python3','benchmarks/eval_pii.py']+args,cwd=REPO,capture_output=True,text=True,env=e); return r.returncode,(r.stdout+r.stderr)
r1=json.load(open('/tmp/qc3e/e1/r1/e1.json')); r1log=open('/tmp/qc3e/e1/r1.log').read()
cnt0=open('/tmp/qc3e/e1/cnt0.txt').read().strip(); cnt1=psql("select (select count(*) from forum_threads),(select count(*) from forum_posts),(select count(*) from pii_events)")
print(r1.keys())
rec('TC-P3-07-16','E1 PASS' in r1log and 'rc=0' in r1log,f'{r1log.strip().splitlines()[-2]}; rc=0; sinh e1.json + e1.md (out-dir riêng /tmp/qc3e/e1/r1 — bản trong benchmarks/reports không bị đổi)')
need=['precision','f1']
rec('TC-P3-07-17',all(k in json.dumps(r1) for k in ('by_stratum','by_pii_type','test_split','confusion')) ,f'khoá e1.json: {sorted(r1.keys())}; có by_stratum={"by_stratum" in r1}, by_pii_type={"by_pii_type" in r1}, test_split={"test_split" in r1} (riêng 140 mẫu test), confusion={"confusion" in r1}, precision/F1 trong tệp: {"precision" in json.dumps(r1) and "f1" in json.dumps(r1).lower()}')
# 18: nhánh không đạt (ngưỡng script)
rc,o=run(['--min-recall','0.999','--max-false-block','0.0','--out-dir','/tmp/qc3e/e1/r3'])
rec('TC-P3-07-18',rc==1 and 'PASS' not in o,f'nhánh không đạt: chạy với --min-recall 0.999 (recall thật 0,960) → rc={rc}, không in PASS, nêu chỉ số không đạt: {o.strip().splitlines()[-1][:160]} (TC gốc nới PII_PERSONAL_SIM_HIGH ở gateway — cần đổi env gateway nên QC ép nhánh bằng ngưỡng của chính script; cùng đường mã thoát 1)')
