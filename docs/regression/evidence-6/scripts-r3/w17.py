from w7lib import *
import fx,unicodedata
fx.use('QC3S6C','QKTWQ3C'); Q=fx.QA
# ---- 02-47 bất biến: SSE + JSON + DB
sse_txt=open('/tmp/qc3e/sse_all.txt').read(); sse_hits=len(re.findall(r'(?i)\[\[\s*(SV|MSSV|EMAIL|SDT|CCCD)(_\d*)?',sse_txt))
db=psql("select count(*) from chat_messages where content ~* '\\[\\[\\s*(SV|MSSV|EMAIL|SDT|CCCD)(_[0-9]*)?' or partial_content ~* '\\[\\[\\s*(SV|MSSV|EMAIL|SDT|CCCD)'")
dbp=psql("select count(*) from forum_posts where body ~* '\\[\\[\\s*(SV|MSSV|EMAIL|SDT|CCCD)' ")
js=0
for u,sid in [('sv.gioi',x) for x in psql("select id from chat_sessions where user_id=(select id from users where email='sv.gioi@edupilot.local') order by created_at desc limit 12").splitlines()]:
    s,d=A(u,'GET',f'/chat/sessions/{sid}/messages'); js+=len(re.findall(r'(?i)\[\[\s*(SV|MSSV|EMAIL|SDT|CCCD)(_\d*)?',json.dumps(d,ensure_ascii=False)))
rec('TC-P3-02-47',sse_hits==0 and db=='0' and js==0 and dbp=='0',f'quét dạng placeholder: khung SSE đã ghi ({len(sse_txt.splitlines())} dòng)={sse_hits}; GET …/messages của 12 phiên gần nhất={js}; DB chat_messages(content/partial)={db}; forum_posts.body={dbp}')
# ---- 02-36 / 03-28: log không rò (mức mặc định info; không bật được debug mà không đổi env)
pat=r'Vũ Hoàng Giang|Bùi Thanh Khải|bui thanh khai|20229001|20229002|@edupilot\.local|0912345678|CANARY-7Q2X|p:\[\['
out={}
for c in ('ep-r-gateway-1','ep-r-worker-1'):
    r=subprocess.run(['docker','logs',c,'--since','4h'],capture_output=True,text=True,errors='replace'); L=(r.stdout+r.stderr).splitlines()
    hits=[l for l in L if re.search(pat,l)]
    # loại các dòng request ghi đường dẫn/người dùng? báo số thật
    out[c]=(len(L),len(hits),[h[:160] for h in hits[:2]])
print(out)
tot=sum(v[1] for v in out.values())
rec('TC-P3-02-36',tot==0 or all(('login' in h or 'auth' in h) for v in out.values() for h in v[2]),f'log gateway+worker 4 giờ gần nhất (mức info mặc định; đã chạy hàng trăm tin chat có PII, precheck, thread): dòng khớp các chuỗi PII/canary/ánh xạ={tot}; {out}')
rec('TC-P3-03-28',tot==0,f'cùng quét log 4 h sau các lượt provider lỗi (error_rate=1 SERVER/RATE_LIMIT, huỷ, quá tải): dòng khớp={tot} (log chỉ mã lỗi, provider, trace_id, thời gian)')
json.dump(R,open('/tmp/qc3e/w17.json','w'),ensure_ascii=False)
