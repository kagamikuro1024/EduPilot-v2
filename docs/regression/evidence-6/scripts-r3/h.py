from lib import *
import re,os,shutil
C1=psql("select id from courses where class_code='761987'"); C2=psql("select id from courses where class_code='761988'")
DUMP='/tmp/qc3e/dump'; os.makedirs(DUMP,exist_ok=True)
PH=re.compile(r'(?i)\[\[\s*(SV|MSSV|EMAIL|SDT|CCCD)(_\d*)?')
_n=[int(time.time())%100000*10]
SSELOG='/tmp/qc3e/sse_all.txt'
def reset(): A('admin','POST','/_test/llm/payloads/reset')
def pay():
    s,p=A('admin','GET','/_test/llm/payloads'); return p['items']
def dump(items):
    for it in items:
        _n[0]+=1; json.dump(it,open(f'{DUMP}/p{_n[0]:06d}.json','w'),ensure_ascii=False)
def flat(its): return '\n'.join('\n'.join(i['messages']+i['inputs']) for i in its)
def ct(prompt,sess,course=C1,task='CHAT'):
    reset(); s,d=A('admin','POST','/_test/llm/chat',{'task':task,'prompt':prompt,'course_id':course,'session_id':sess},timeout=60)
    its=pay(); dump(its); return s,d,its
_sess={}
def newsess(u,course=C1,doc=None):
    b={'course_id':course,'title':'qc3'}; 
    if doc: b['document_id']=doc
    s,d=A(u,'POST','/chat/sessions',b); assert s==201,(s,d); return d['id']
def chat(u,text,sid=None,course=C1,idem=None,keep=True,maxsec=90):
    """gửi tin qua đường chat thật; trả dict: sid, events, text, notice, done, payload (provider giả), row"""
    sid=sid or newsess(u,course); reset()
    tk=login(u); st,ev=sse('POST',f'/chat/sessions/{sid}/messages',tk,{'content':text},idem=idem,maxsec=maxsec)
    its=pay(); dump(its)
    for e in ev: open(SSELOG,'a').write(f'{e[0]} {e[1]}\n')
    toks=''.join(json.loads(d).get('t',json.loads(d).get('text','')) if e=='token' else '' for e,d,*_ in ev if d.startswith('{'))
    notice=[json.loads(d) for e,d,*_ in ev if e=='notice']
    done=[json.loads(d) for e,d,*_ in ev if e=='done']
    err=[json.loads(d) for e,d,*_ in ev if e=='error']
    row=psql(f"select intent,masked_count,degraded,no_context,low_confidence,stream_status,error_code from chat_messages where session_id='{sid}' and role='ASSISTANT' order by created_at desc limit 1")
    return dict(sid=sid,status=st,events=ev,text=toks,notice=notice,done=done,err=err,payload=its,row=row)
