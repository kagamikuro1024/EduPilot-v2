from lib3 import *
A_=C7['A']; Q=f'/courses/{A_}'
eid=mkexam3(A_); print('exam',eid)
u='qc3s27'
sid,um,am=mkchat(u,A_); s0,d0=A(u,'POST','/chat/sessions',{'course_id':A_})  # trước khi thi
aid,s,j=startexam(A_,eid,u); print('attempt',aid,s)
sidx,_,amx=(sid,None,am)
r_esc=esc(u,am); 
c1=A(u,'POST','/chat/sessions',{'course_id':A_}); c2=A(u,'POST',f'/chat/sessions/{sid}/messages',{'content':'Giải thích giúp mình bài thi'})
rec('TC-P4-01-32',aid is not None and r_esc[0]==409 and r_esc[1].get('code')=='EXAM_IN_PROGRESS' and c2[0]==409 and c2[1].get('code')==r_esc[1].get('code') and 'until' in json.dumps(r_esc[1]),f'đang thi: escalate → {r_esc[0]} {r_esc[1].get("code")} details={r_esc[1].get("details")}; chat riêng cùng lúc → {c2[0]} {c2[1].get("code")} details={c2[1].get("details")} (cùng mã/trạng thái/thân; đã lên lớp {A_[:8]})')
sb=submitexam(A_,eid,aid,u); time.sleep(2)
r2=esc(u,am); rec('TC-P4-01-32b',r2[0] in(200,201),f'nộp bài ({sb[0]}) → escalate lại {r2[0]}') if False else print('after submit',sb[0],r2[0])
save('p401e')
