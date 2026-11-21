"""Ghi kết quả lượt qc3 (sprint 5) vào evidence-5/results-q2.jsonl (bản ghi cuối thắng), rồi `python3 make_report.py` sinh smoke-5.md."""
import json,subprocess,os,sys
ROOT='/Users/kuro/Documents/TA_Agent_qc3/docs/regression/evidence-5'
F=ROOT+'/results-q2.jsonl'
def R(tc,r,note):
    note=' '.join(str(note).split())
    open(F,'a').write(json.dumps(dict(tc=tc,r=r,note=note),ensure_ascii=False)+'\n'); print(r,tc,note[:160])
def sh(cmd,timeout=900,cwd=None,log=None):
    p=subprocess.run(cmd,shell=True,capture_output=True,text=True,timeout=timeout,cwd=cwd,errors='replace')
    out=p.stdout+p.stderr
    if log: open(ROOT+'/logs-r3/'+log,'w').write(out)
    return p.returncode,out
if __name__=='__main__': R(sys.argv[1],sys.argv[2],sys.argv[3])
