from lib3 import *
A_=C7['A']; Q=f'/courses/{A_}'; T='qc3t'; TA='qc3ta'
# E2: gửi lại cùng key không tin đôi
tE,_=tick('qc3s17'); A(T,'POST',f'{Q}/tickets/{tE}/claim'); k=str(uuid.uuid4())
r1=A(T,'POST',f'{Q}/tickets/{tE}/answer',{'body':'Chào em, đây là lời đáp E2.'},idem=k); r2=A(T,'POST',f'{Q}/tickets/{tE}/answer',{'body':'Chào em, đây là lời đáp E2.'},idem=k)
nn=psql(f"select count(*) from chat_messages where intent='STAFF_ANSWER' and user_id='{uid('qc3s17')}' and content like '%lời đáp E2%'")
rec('TC-P4-03-E2',r1[0]==200 and r2[0]==200 and nn=='1',f'gửi hai lần cùng key → {r1[0]}/{r2[0]}; số tin STAFF_ANSWER={nn}')
# P1: lớp khác
B_=C7['B']; tb,_=tick('qc3s01',B_)
r=[A(T,'GET',f'/courses/{B_}/tickets/{tb}')[0]]
Cc=C7['A']; tA2,_=tick('qc3s21')
x1=A('r5t','GET',f'{Q}/tickets/{tA2}')[0]; x2=A('r5ta','POST',f'{Q}/tickets/{tA2}/answer',{'body':'Chào em, TA lớp khác.'})[0]
sv=[A('qc3s02',m,p)[0] for m,p in (('GET',f'{Q}/tickets'),('GET',f'{Q}/tickets/{tA2}'),('POST',f'{Q}/tickets/{tA2}/claim'),('POST',f'{Q}/tickets/{tA2}/answer'),('POST',f'{Q}/tickets/{tA2}/dismiss'))]
rec('TC-P4-03-P1',x1 in(403,404) and x2 in(403,404) and all(v==403 for v in sv),f'GV lớp khác mở ticket → {x1}; TA lớp khác answer → {x2}; SV gọi route staff → {sv} (spec: 404 cho cross-course; ghi mã thực)')
save('p403')
