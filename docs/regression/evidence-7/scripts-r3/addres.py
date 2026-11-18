import json,sys
def add(d):
    try: r=json.load(open('/tmp/qc3e/res7c.json'))
    except Exception: r={}
    r.update(d); json.dump(r,open('/tmp/qc3e/res7c.json','w'),ensure_ascii=False,indent=0)
