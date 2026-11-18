from lib3 import *
G=C7['G']; QG=f'/courses/{G}'; T='qc3t5'; TA='qc3ta'
clearopen(T,G); time.sleep(1)
def titems(u=T,course=G):
    d=A(u,'GET',f'/courses/{course}/today')[1]; return [a for a in d.get('actions',[]) if a['kind']=='TICKET']
n0=titems()
t0=time.time(); ids=[tick('qc3s%02d'%i,G)[0] for i in (1,2,3)]
got=None
for _ in range(100):
    it=titems()
    if it: got=time.time()-t0; break
    time.sleep(0.1)
A(T,'POST',f'{QG}/tickets/{ids[0]}/claim'); A(TA,'POST',f'{QG}/tickets/{ids[1]}/claim'); time.sleep(1.5)
it=titems(); itA=json.dumps(it,ensure_ascii=False)
sv=A('qc3s01','GET','/me/today')[1]; svT=[a for a in sv.get('actions',[]) if a['kind']=='TICKET']
okfmt=bool(it) and re.search(r'1 câu hỏi đang chờ · 1 câu bạn đã nhận · lâu nhất',it[0]['title']+' '+it[0].get('reason','')) and '/inbox?course=' in it[0]['href']
# hết ticket
for t in ids:
    A(T,'POST',f'{QG}/tickets/{t}/claim'); A(T,'POST',f'{QG}/tickets/{t}/dismiss',{'reason':'OUT_OF_SCOPE'})
t1=time.time(); gone=None
for _ in range(80):
    if not titems(): gone=time.time()-t1; break
    time.sleep(1)
rec('TC-P4-03-37',got is not None and got<=5 and okfmt and not svT and gone is not None and gone<=60,f'3 ticket (1 GV nhận, 1 TA nhận): mục TICKET xuất hiện sau {got:.2f}s (≤5); nội dung {itA[:330]}; SV me/today mục TICKET={len(svT)}; xử lý hết → mục biến sau {gone}s (≤60); ticket TA đã nhận không tính vào «đang chờ» của GV (1 đang chờ)')
save('p403c')
