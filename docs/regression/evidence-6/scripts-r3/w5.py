from w3lib import *
import fx,threading,hashlib
# ===== 02-37: precheck cần nhúng + thread mới (worker trả lời) không tạo khoá ep:mask:*
fx.use('QC3S6C','QKTWQ3C'); Q=fx.QA
title='Hỏi về hàm băm'; body='Giải thích giúp mình về hàm băm mật mã và tính chống va chạm'
fx.add_ctx([title+'\n'+body]); 
def keys(): return set(rds('--scan','--pattern','ep:mask:*').split())
n0=keys()
s,d=A('sv.gioi','POST',f'/courses/{Q}/threads/precheck',{'body':body,'title':title}); 
s2,th=A('sv.gioi','POST',f'/courses/{Q}/threads',{'title':title,'body':body}); tid=th['thread']['id']
st=None
for i in range(30):
    time.sleep(2); st=psql(f"select ai_state from forum_threads where id='{tid}'")
    if st in('SKIPPED','ANSWERED'): break
n1=keys()
rec('TC-P3-02-37',s==200 and s2 in(200,201) and st=='ANSWERED' and not (n1-n0),f'precheck {s}, tạo thread {s2}, worker xử lý xong (ai_state={st}); khoá ep:mask:* trước={len(n0)} sau={len(n1)} khoá mới={len(n1-n0)}')
# ===== 03-16: bài AI của thread (đường llm.Chat ở worker)
body16=psql(f"select body from forum_posts where thread_id='{tid}' and kind='AI' order by created_at desc limit 1")
print('AI body:',body16[:200])
rec('TC-P3-03-16',bool(body16) and PH.search(body16) is None,f'thread {tid[:8]} (ai_state={st}): bài AI body={body16[:100]!r}; placeholder trong body: {PH.search(body16) is not None}')
# ===== 02-43: giữ tối đa 32 rune, token không có '[' không bị giữ
tk=login('admin')
def stream(prompt,sess,delay=None):
    if delay is not None: A('admin','POST','/_test/llm/fake',{'stream_delay_ms':delay})
    st,ev=sse('POST','/_test/llm/chat',tk,{'task':'CHAT','prompt':prompt,'course_id':C1,'session_id':sess,'stream':True},maxsec=60)
    if delay is not None: A('admin','POST','/_test/llm/fake',{'stream_delay_ms':20})
    toks=[(json.loads(d)['text'],t) for e,d,t in ev if e=='token']; return st,toks,ev
st,toks,ev=stream('xin chào các bạn ơi hôm nay học bài mới nhé','qc3-43a',delay=60)
gaps=[round((toks[i+1][1]-toks[i][1])*1000) for i in range(len(toks)-1)]
okA=st==200 and len(toks)>=8 and max(gaps)<250
p43=' '*40  # '[[' + 40 rune (khoảng trắng khớp tiền tố) không đóng
st,tb,evb=stream('[['+p43+'x','qc3-43b',delay=60)
frames=[t for t,_ in tb]; full=''.join(frames)
fi=next((i for i,f in enumerate(frames) if '[[' in f),None)
heldrunes=len(''.join(frames[fi:]).split('[[',1)[1]) if fi is not None else 0
pre=''.join(frames[:fi]) if fi is not None else ''
okB=fi is not None and '[['+p43+'x' in full and fi>0 and 28<=len(frames[fi].split('[[',1)[1])<=34 and '[' not in pre
rec('TC-P3-02-43',okA and okB,f'(a) câu không có "[": {len(toks)} khung, khoảng cách tối đa {max(gaps)} ms (không bị giữ); (b) "[[" + 40 rune không đóng: khung đầu chứa "[[" là khung #{fi}/{len(frames)} và mang {len(frames[fi].split("[[",1)[1]) if fi is not None else 0} rune sau "[[" (giữ ≤32 rồi phát), văn bản cuối đủ 100%={("[["+p43+"x") in full}')
# ===== 02-45: xoá ánh xạ giữa luồng
sess='qc3-45-'+str(int(time.time()))
res={}
def run(): res['r']=stream('Chào Bùi Thanh Khải và Vũ Minh Long cùng 20229002 và 20229001 nhé các bạn','%s'%sess,delay=500)
t=threading.Thread(target=run); t.start(); time.sleep(1.2); rds('del','ep:mask:'+sess); t.join()
st,toks,ev=res['r']; full=''.join(x for x,_ in toks)
rec('TC-P3-02-45',st==200 and 'Khải' not in full and '[[' not in full and any(e=='done' for e,*_ in ev),f'DEL ep:mask:{sess[-8:]} sau 1,2 s giữa luồng (500 ms/từ): HTTP {st}, văn bản cuối={full!r}, còn "[[": {"[[" in full}, có done: {any(e=="done" for e,*_ in ev)}')
json.dump(R,open('/tmp/qc3e/w5.json','w'),ensure_ascii=False)
