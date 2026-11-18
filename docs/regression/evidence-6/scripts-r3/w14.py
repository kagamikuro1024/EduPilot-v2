from w7lib import *
import fx
fx.use('QC3S6C','QKTWQ3C'); Q=fx.QA
def clr(): subprocess.run(['docker','exec','ep-r-redis-1','sh','-c',"redis-cli --scan --pattern 'ep:ans:*' | xargs -r redis-cli del"],capture_output=True)
G='Vũ Hoàng Giang'; uidG=psql("select id from users where email='sv.gioi@edupilot.local'")
# ===== 35 API
r=chat('sv.gioi',f'Em là {G}, MSSV 20229001, điểm giữa kỳ của em thế nào?',course=C1)
nm=r['notice'][0] if r['notice'] else {}
mc=psql(f"select masked_count from chat_messages where session_id='{r['sid']}' and role='ASSISTANT'")
rec('TC-P3-05-35-api',nm.get('masked')==2 and mc=='2',f'notice={r["notice"]}; chat_messages.masked_count={mc} (UI chữ "Đã ẩn 2 thông tin…" ở nhóm UI)')
# ===== 36: SSE không placeholder, tin có [[ -f ]] nguyên văn
fx.add_ctx(['[[ -f "$f" ]] làm gì?'],'T')
r=ask('sv.gioi','[[ -f "$f" ]] làm gì?',course=Q,title='T'); allsse=open('/tmp/qc3e/sse_all.txt').read().splitlines()[-len(r['events']):]
ph=[l for l in allsse if PH.search(l.replace('[[ -f','')) ]
hist=psql(f"select content from chat_messages where session_id='{r['sid']}' and role='USER'")
rec('TC-P3-05-36',not ph and '[[ -f "$f" ]]' in r['text'] and hist=='[[ -f "$f" ]] làm gì?',f'khung SSE chứa dạng placeholder: {len(ph)}; câu trả lời giữ nguyên `[[ -f "$f" ]]`: {"[[ -f" in r["text"]}; tin người dùng lưu nguyên văn: {hist==chr(91)*2+" -f"+chr(34)+"$f"+chr(34)+" ]] làm gì?"}; văn bản={r["text"][:100]!r}')
# ===== 37 pii_events
n0=psql(f"select count(*) from pii_events where user_id='{uidG}'")
r=chat('sv.gioi',f'Em là {G}, MSSV 20229001, hỏi về quy chế thi',course=C1)
rows=psql(f"select pii_type||'/'||count||'/'||action||'/'||channel||'/'||(session_id is not null) from pii_events where user_id='{uidG}' order by created_at desc limit 3")
cols=psql("select string_agg(column_name,',') from information_schema.columns where table_name='pii_events'")
n1=psql(f"select count(*) from pii_events where user_id='{uidG}'")
r2=chat('sv.gioi','Hàm băm SHA-256 dùng làm gì',course=C1); n2=psql(f"select count(*) from pii_events where user_id='{uidG}'")
rec('TC-P3-05-37',all('MASKED' in x and 'PRIVATE' in x for x in rows.splitlines()[:2]) and 'text' not in cols and 'content' not in cols and n2==n1 and not r2['notice'],f'tin có tên+MSSV: dòng pii_events {rows.splitlines()[:2]}; cột bảng = {cols} (không cột văn bản); câu thuần học thuật: không notice, pii_events {n1}→{n2} (không ghi MASKED)')
# ===== 38: 8 tin
msgs=[G,'Vu Hoang Giang','Giang Vũ Hoàng','MSSV 20229001','sv.gioi@edupilot.local','0912 345 678','001099012345','Điểm của bạn Bùi Thanh Khải 20229002 bao nhiêu?']
outs=[]
for m in msgs:
    x=chat('sv.gioi',m,course=C1); outs.append((m,(x['notice'][0].get('masked') if x['notice'] else 0),x['row'].split('|')[0])); time.sleep(0.5)
rf=psql(f"select action||'/'||pii_type from pii_events where user_id='{uidG}' order by created_at desc limit 3")
last=chat('sv.gioi',msgs[7],course=C1)
rec('TC-P3-05-38',all(o[1]>=1 for o in outs) and 'Mình chỉ xem được dữ liệu của chính bạn.' in last['text'] and 'BLOCKED/OTHER_PERSON' in psql(f"select string_agg(action||'/'||pii_type,',') from (select * from pii_events where user_id='{uidG}' order by created_at desc limit 4) z"),f'8 tin → notice.masked={[o[1] for o in outs]} (đều ≥1); intent tin 8={outs[7][2]}, câu từ chối={last["text"][:50]!r}; pii_events gần nhất={rf}')
# ===== 40: block upcoming_events
r=chat('sv.gioi','Tuần này em có lịch gì?',course=C1); bl=[json.loads(d) for e,d,*_ in r['events'] if e=='block']
bdb=psql(f"select blocks::text from chat_messages where session_id='{r['sid']}' and role='ASSISTANT'") if False else None
s,d=A('sv.gioi','GET',f"/chat/sessions/{r['sid']}/messages"); persisted=json.dumps(d,ensure_ascii=False)
rec('TC-P3-05-40-api',any(b.get('kind')=='upcoming_events' for b in bl) and 'upcoming_events' in persisted,f'khung block kind={[b.get("kind") for b in bl]}, dữ liệu {json.dumps(bl[0].get("data"),ensure_ascii=False)[:120] if bl else None}; GET messages (tải lại) vẫn có block upcoming_events={"upcoming_events" in persisted} (UI «Lịch sắp tới» ở nhóm UI)')
# ===== 41 citations
fx.add_ctx(['Cấu hình tường lửa mạng cho máy chủ'],'Bài giảng An ninh mạng')
r=ask('sv.gioi','Cấu hình tường lửa mạng cho máy chủ',course=Q,title='Bài giảng An ninh mạng'); dn=r['done'][0] if r['done'] else {}
cit=dn.get('citations',[]); txt=r['text']; used=set(re.findall(r'\[(\d+)\]',txt))
ok=bool(cit) and all(set(c)>={'n','document_id','title','page_no','snippet'} and len(c['snippet'])<=200 for c in cit) and {str(c['n']) for c in cit}<=used|{str(c['n']) for c in cit}
rec('TC-P3-05-41-api',ok and all(str(c['n']) in used for c in cit),f'done.citations={json.dumps(cit,ensure_ascii=False)[:230]}; các [n] xuất hiện trong văn bản={sorted(used)}; mỗi mục đủ n/document_id/title/page_no/snippet(≤200) (UI mở rộng + «Xem tài liệu» ở nhóm UI)')
# ===== 42 [9] dangling
t9='Bài [9] ghi chú'; q9='Phân tích rủi ro hệ thống thông tin doanh nghiệp'; fx.add_ctx([q9],t9)
r=ask('sv.gioi',q9,course=Q,title=t9); 
rec('TC-P3-05-42-a','[9]' not in r['text'] and r['done'] and all(c['n']!=9 for c in r['done'][0].get('citations',[])),f'tiêu đề tài liệu mang "[9]" làm câu trả lời giả lập có [9] nằm ngoài citations: văn bản cuối={r["text"][:90]!r} (không còn [9]); citations n={[c["n"] for c in r["done"][0].get("citations",[])] if r["done"] else None}')
json.dump(R,open('/tmp/qc3e/w14.json','w'),ensure_ascii=False)
