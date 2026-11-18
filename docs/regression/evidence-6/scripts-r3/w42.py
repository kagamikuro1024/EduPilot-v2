from p8lib import *
import fx,re
from h import pay,reset
fx.use('QC3P8A','QKTWP8A'); C=json.load(open('/tmp/qc3e/p8c.json')); A1=C['A']; QA=f'/courses/{A1}'
Q='có tài liệu nào về an ninh mạng không? '+str(int(time.time()))
subprocess.run(['docker','exec','ep-r-redis-1','sh','-c',"redis-cli --scan --pattern 'ep:ans:*' | xargs -r redis-cli del"],capture_output=True)
reset(); s,th=A('sv.gioi','POST',f'{QA}/threads',{'title':Q,'body':Q+' Nhờ thầy cô chỉ giúp.','tags':[],'week_no':1}); tid=th['thread']['id']; st=''
for i in range(30):
    time.sleep(3); st=psql(f"select ai_state||'/'||coalesce(ai_skip_reason,'') from forum_threads where id='{tid}'")
    if not st.startswith('PENDING') and not st.startswith('PROCESSING'): break
items=pay(); pt=json.dumps(items,ensure_ascii=False); ids=set(re.findall(r'/library/([0-9a-f-]{36})',pt))
body=psql(f"select left(body,200) from forum_posts where thread_id='{tid}' and kind='AI' limit 1")
print(st,len(ids),[(i['task']) for i in items],body[:120])
rec('TC-P8-02-33',len(ids)>0 and len(ids)<=5 and not re.search(r'user_id|student_id|mssv',pt),f'Threads (kênh công khai): bài hỏi «có tài liệu nào về an ninh mạng không?» → ai_state={st}; lời nhắc gửi provider chứa kết quả search_library: {len(ids)} mục (≤5, href /library/{{id}}), không có tham số danh tính; chat riêng: intent LIBRARY_SEARCH, 5 mục (≤5) cùng định dạng (lượt trước); bài AI: {body[:60]!r}')
