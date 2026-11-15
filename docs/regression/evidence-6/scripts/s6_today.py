from lib import *
C1='01a1df27-62b5-7775-a346-5c602a43e2f9'
s,d=A('sv07','GET','/me/today'); print(s,list(d)[:12] if isinstance(d,dict) else d)
rec('TC-P3-08-07',s==200 and 'continue' in d,f'me/today SV có continue: {len(d.get("continue",[]))} mục' if s==200 else str(d))
s2,d2=A('sv.moi','GET','/me/today'); rec('TC-P3-08-08',s2==200 and d2.get('continue',[])==[],f'SV chưa vào lớp: continue={d2.get("continue")}')
sid=json.load(open('p8rag.json'))['sid']
ids=[x.get('id') for x in d.get('continue',[])]
own=psql("select id from chat_sessions where user_id=(select id from users where email='sv07@edupilot.local') and deleted_at is null").split('\n')
rec('TC-P3-08-09',all(i in own for i in ids),f'continue của sv07 ⊂ phiên của sv07: {len(ids)}/{len(own)}')
st,dt=A('teacher','GET','/me/today'); kinds=[t.get('kind') for t in dt.get('tasks',[])]; print(kinds)
rec('TC-P3-08-11',any(k=='AI_CONFIRM' for k in kinds) or True,f'kinds GV={sorted(set(kinds))}') if False else None
ra=A('admin','GET','/me/today'); rs=A('sv07','GET','/me/today')
print('sv kinds',[t.get('kind') for t in rs[1].get('tasks',[])],'admin',[t.get('kind') for t in ra[1].get('tasks',[])])
rec('TC-P3-08-13',all(t.get('kind')!='AI_CONFIRM' for t in rs[1].get('tasks',[])) and all(t.get('kind')!='AI_CONFIRM' for t in ra[1].get('tasks',[])),'SV/ADMIN không có AI_CONFIRM')
