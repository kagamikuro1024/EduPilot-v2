import json
EV='/Users/kuro/Documents/TA_Agent_qc3/docs/regression/evidence-5/results-q2.jsonl'
def rec5(tc,ok,note):
    open(EV,'a').write(json.dumps({'tc':tc,'r':'P' if ok else 'F','note':note,'suite':'q3k6'},ensure_ascii=False)+'\n'); print('P' if ok else 'F',tc,note[:150])
