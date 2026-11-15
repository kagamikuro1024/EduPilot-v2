from lib import *
import hashlib,urllib.request,time
exec(open('s6_p8.py').read().split('# --- presign nhánh lỗi')[0])
D=json.load(open('p8doc.json'))['doc']
b=mkdocx(['x'])
# 07 PUT sai kích thước (Content-Length khớp thân thật nhưng khác chữ ký)
s,d=presign('teacher',Q,'a.docx',DOCX,b); big=b+b'junk'; h=dict(d['headers']); h['Content-Length']=str(len(big))
req=urllib.request.Request(d['url'],data=big,method='PUT',headers=h)
try: st=urllib.request.urlopen(req,timeout=30).status
except urllib.error.HTTPError as e: st=e.code
st_ok=put(d,b)
rec('TC-P8-01-07',st>=400 and st_ok==200,f'PUT sai kích thước → {st}, đúng → {st_ok} (mốc hết hạn 10 phút: không chờ)')
# 58
res={}
for u in ('sv.gioi','ta','teacher','admin'):
    r=[]
    for m,p in (('POST',f'{Q}/uploads/presign'),('POST',f'{Q}/uploads/complete'),('POST',f'{Q}/documents/{D}/retry'),('POST',f'{Q}/documents/{D}/reindex')):
        s,_=A(u,m,p,{}); r.append(s)
    res[u]=r
print(res)
ok=all(x==403 for x in res['sv.gioi']) and all(x==403 for x in res['admin']) and all(x!=403 for x in res['ta']) and all(x!=403 for x in res['teacher'])
rec('TC-P8-01-58',ok,str(res))
s1,_=A('ta','POST',f'{Q}/documents/reindex',{}); s2,_=A('teacher','POST',f'{Q}/documents/reindex',{})
rec('TC-P8-01-59',s1==403 and s2==202,f'reindex cả lớp: TA {s1}, TEACHER {s2}')
# 60 ngoài lớp
r=[A('r5t','POST',f'{Q}/uploads/presign',{})[0],A('sv.moi','POST',f'{Q}/uploads/presign',{})[0],A('r5t','GET',f'{Q}/documents')[0]]
jobs=psql(f"select id from jobs order by created_at desc limit 1"); sj=A('sv.gioi','GET',f'/jobs/{jobs}')[0]
rec('TC-P8-01-60',all(x==403 for x in r) and sj==404,f'staff lớp khác/SV ngoài → {r}; xem job người khác → {sj}')
# 02-xx
s,d=A('teacher','GET',f'{Q}/documents?limit=1'); pg=[len(d['items'])]; cur=d.get('next_cursor')
s2,_=A('teacher','GET',f'{Q}/documents?limit=0'); s3,_=A('teacher','GET',f'{Q}/documents?limit=101')
rec('TC-P8-02-03',s==200 and s2==422 and s3==422,f'limit=1 → {len(d["items"])} mục; limit 0/101 → {s2}/{s3}')
# 12 khoá lạc quan
s,d=A('teacher','GET',f'{Q}/documents/{D}'); v=d['version']
s1,_=A('teacher','PATCH',f'{Q}/documents/{D}',{'title':'QC e6 Hash A','version':v}); s2,d2=A('teacher','PATCH',f'{Q}/documents/{D}',{'title':'QC e6 Hash B','version':v})
rec('TC-P8-02-12',s1==200 and s2 in(409,412),f'PATCH lần 1 → {s1}; version cũ → {s2} {d2.get("code") if isinstance(d2,dict) else ""}')
# 13 SV thấy / không thấy theo cờ
q='Hash'
s,l1=A('sv.gioi','GET',f'{Q}/library?q={q}'); s,d=A('teacher','GET',f'{Q}/documents/{D}'); v=d['version']
A('teacher','PATCH',f'{Q}/documents/{D}',{'visible_to_students':False,'version':v}); s,l2=A('sv.gioi','GET',f'{Q}/library?q={q}')
A('teacher','PATCH',f'{Q}/documents/{D}',{'visible_to_students':True,'version':v+1})
n1=len(l1['items']); n2=len(l2['items'])
rec('TC-P8-02-13',n1>=1 and n2==0,f'thư viện SV: hiện {n1} → ẩn {n2} (ngay yêu cầu kế tiếp)')
# 14 audit
a=psql(f"select action from audit_log where entity_id='{D}' order by created_at desc limit 3"); rec('TC-P8-02-14',a!='' ,a.replace('\n',','))
# 16/17 chunks
s,d=A('teacher','GET',f'{Q}/documents/{D}/chunks?limit=2'); ch=d['items'][0]['id'] if s==200 and d['items'] else None
s1,_=A('teacher','PATCH',f'{Q}/documents/{D}/chunks/{ch}',{'text':'Hash functions: SHA-256 digest (QC e6 đã sửa)'}) 
s2,_=A('teacher','PATCH',f'{Q}/documents/{D}/chunks/{ch}',{'text':''}); s3,_=A('teacher','PATCH',f'{Q}/documents/{D}/chunks/{ch}',{'text':'a'*4001})
time.sleep(2); tx=psql(f"select text from content_chunks where id='{ch}'")
rec('TC-P8-02-16',s==200 and s1==200 and 'đã sửa' in tx,f'đọc chunks {s}, sửa {s1}, DB đã đổi={"đã sửa" in tx}'); rec('TC-P8-02-17',s2==422 and s3==422,f'text rỗng/4001 → {s2}/{s3}')
# 18 stats
s,d=A('teacher','GET',f'{Q}/documents/stats'); rec('TC-P8-02-18',s==200,str(d)[:100])
# 23-26 library
r=[A('sv.gioi','GET',f'{Q}/library?q={x}')[0] for x in ('hash','HASH','a')]
rec('TC-P8-02-23',r[0]==200 and r[1]==200,f'library q → {r}')
s,d=A('sv.gioi','GET',f'{Q}/library?limit=1'); s2,_=A('sv.gioi','GET',f'{Q}/library?limit=101'); rec('TC-P8-02-26',s==200 and s2==422,f'limit=1 ok, 101 → {s2}')
# 27 download
s,d=A('sv.gioi','GET',f'{Q}/library/{D}'); s2,_=A('sv.gioi','GET',f'{Q}/library/{D}/download')
rec('TC-P8-02-27',s==200 and s2 in(200,302),f'chi tiết → {s}, download → {s2}; keys={list(d)[:8] if s==200 else d}')
# 45-47 phân quyền
rr={u:[A(u,'GET',f'{Q}{p}')[0] for p in ('/documents','/documents/stats',f'/documents/{D}',f'/documents/{D}/chunks')] for u in ('sv.gioi','ta','teacher','admin')}
rec('TC-P8-02-45',all(x==403 for x in rr['sv.gioi']) and all(x==403 for x in rr['admin']) and all(x==200 for x in rr['ta']+rr['teacher']),str(rr))
s1,_=A('ta','DELETE',f'{Q}/documents/{D}'); rec('TC-P8-02-21',s1==403,f'TA xoá tài liệu → {s1}')
lib={u:A(u,'GET',f'{Q}/library')[0] for u in ('ta','teacher','admin','sv.gioi')}
rec('TC-P8-02-47',lib['sv.gioi']==200 and lib['admin']==403,str(lib))
s,_=A('sv.moi','GET',f'{Q}/library'); s2,_=A('sv.moi','GET',f'{Q}/library/{D}/download'); rec('TC-P8-02-48',s==403 and s2 in(403,404),f'SV ngoài lớp: library {s}, download {s2}')
