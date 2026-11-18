from fx import *
import fx
fx.use('QC3P8A','QKTWP8A')
A1=fx.QA; d=json.load(open('/tmp/qc3e/p8d.json'))
Q0='Hãy giải thích chế độ mã hoá khối GCM và CBC khác nhau thế nào QC30'
r=fx.ask('sv.gioi',Q0,pre_ctx=False)
print(r['row'],r['done'],r['err'])
emb=[i for it in r['payload'] if it['task']=='EMBEDDING' for i in it['inputs']]; print('emb',emb)
gen=[it for it in r['payload'] if it['task']!='EMBEDDING']; print(json.dumps(gen,ensure_ascii=False)[:1500])
print(psql(f"select citations from chat_messages where session_id='{r['sid']}' and role='ASSISTANT'")[:600])
