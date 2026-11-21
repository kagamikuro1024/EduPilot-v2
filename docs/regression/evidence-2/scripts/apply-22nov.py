import re,glob
B='evidence-2/run-logs-22nov/'
why={
'PG01-11':'biến JWT_EXPIRATION không còn (đổi thành ACCESS_TOKEN_TTL, sprint 3)',
'PG01-16':'APP_ENV=production nay kiểm khoá/CORS bắt buộc nên thoát <100 ms (sprint sau)',
'PG01-34':'Redis ep-r dùng chung 7 nhóm khoá (ep:auth:*, ep:llm:* … thêm sau sprint 2)',
'PG01-68':'Makefile test nay có 2 lần RYUK_DISABLED=true',
'PG01-85':'Makefile lint nay gọi golangci-lint 3 lần (không tag, testroutes, +1)',
'PG02-01':'schema 00001→00029: nay 60+ bảng (sprint 3…11); TC đếm theo 00001',
'PG02-02':'enum nay nhiều hơn (sprint 3…11)',
'PG02-19':'index 308 (TC đòi 17 theo 00001)',
'PG02-21':'66 cột id uuidv7() (TC đòi 5)',
'PG02-42':'goose version mới nhất 29 (TC đòi 1)',
'PG02-48':'float64 trong mã sqlc sinh từ cột double precision (sprint sau; spec v1.5 miễn _test.go)',
'PG02-49':'1 dòng «OFFSET» nằm trong chú thích «không OFFSET» (observability.sql:2), không có OFFSET trong câu SQL — grep quá chặt',
'PG02-70':'cùng lý do PG02-01/19/21 (schema tăng)',
'PG03-51':'11 dòng OFFSET trong mã/chú thích ngoài _test.go (sprint sau thêm truy vấn OFFSET ở bảng nhỏ hoặc chú thích)',
'PG03-98':'APP_ENV=production: log lỗi nêu biến production khác (kiểm tra production sprint sau)',
'PG04-44':'openapi nay có /auth/ /me/ /admin/ (sprint 3…11), TC đòi 0',
'PG04-45':'route auth/me/admin nay tồn tại (405/415/200), TC đòi 404 theo sprint 2',
'PG04-46':'route auth/me/admin nay tồn tại (401/405/415), TC đòi 404 theo sprint 2',
'PG06-04':'openapi.yaml 299 đường dẫn (TC đòi 5)','PG06-05':'2 chữ «_test» nằm trong mô tả (sprint sau)',
'PG06-06':'openapi.test.yaml 19 đường dẫn thử (TC đòi 13)','PG06-07':'345 thao tác (TC đòi 5), 21 thao tác thử (TC đòi 15)',
'PG07-01':'compose nay 13 service (thêm judge, mock-graph, mock-imap…)',
'PG07-02':'compose nay có docling (11 chuỗi)',
'PG07-35':'4 chỗ ghi đĩa: internal/llm/fake/fake.go (ghi bản ghi replay của provider giả, chỉ dùng test/demo) và cmd/regress/main.go (CLI offline) — không nằm trên đường request',
'PG07-37':'mỗi container có 2 thay đổi FS: A /seed, A /seed/demo (thư mục mount seed của ep-r)',
'PG07-38':'gh không có run CI cho sha 06f164c7 (commit cũ của sprint 2 đã bị thay bằng rebase/merge)',
'PG07-39':'không tìm được run CI của sprint 2','PG07-40':'không có run CI sprint 2 để đếm bước','PG07-41':'không có run CI sprint 2 để đếm job',
'PG07-43':'nhánh ci/sqlc-drift đã xoá (TC ghi: xoá sớm = FAIL)','PG07-44':'không có run ci/sqlc-drift để đối chiếu',
'PG07-56':'compose nay công bố thêm cổng mock-graph, mock-imap; trên ep-r caddy ánh xạ 780/1143, frontend không công bố 3000 (ep-r dùng cổng riêng)',
'PG07-58':'.env.example có SEED_DEFAULT_PASSWORD không chứa «-dev» (mật khẩu seed theo thiết kế)',
'PG07-66':'compose có override nay 13 service (thêm judge, mock-graph, mock-imap)',
}
manual={
'PG06-38':('P','TC tay: exempt.go (đếm operation/status/reason = 0/0/0, reason rỗng 0) khớp handoff dev-US-PG-06.md «exempt.go miễn trừ — rỗng»: hai danh sách cùng 0 mục; TC-PG06-28 PASS cùng lượt'),
'PG07-09':('P','RERUN 22/11 @aba738ef trên ep-r (cổng đổi 80→780): curl -sI http://localhost:780/ → HTTP/1.1 308 Permanent Redirect, Location: https://localhost/'),
'PG07-14':('P','RERUN 22/11 @aba738ef trên ep-r (cổng 443→1143): openssl s_client -connect localhost:1143 → issuer CN=Caddy Local Authority - ECC Intermediate; SAN DNS:localhost; notAfter=Nov 22 01:07:54 2026 GMT'),
}
res={}
for f in sorted(glob.glob(B+'0?-n.log')):
    n=f.split('/')[-1][:2]; cur=None; fails={}
    for l in open(f,errors='ignore'):
        m=re.match(r'^-- (TC-\S+) (PASS|FAIL|MANUAL)',l)
        if m:
            code=m.group(1)[3:]; res[code]=(m.group(2),fails.get(m.group(1),''),f.split('/')[-1])
        m=re.match(r'^== (TC-\S+)',l)
        if m: cur=m.group(1)
        if cur and l.startswith('    FAIL') and cur not in fails: fails[cur]=l.strip()[5:].strip()[:140]
for k,(v,n) in manual.items(): res[k]=(v,n,'manual')
lines=open('smoke-2.md',encoding='utf-8').read().split('\n')
out=[];done=0
for l in lines:
    m=re.match(r'^\| (TC-\S+?)(?:…TC-(\S+))? \| N \|',l)
    if m:
        a=m.group(1)[3:]; b=m.group(2)
        codes=[a]
        if b:
            n0=int(a.split('-')[1]); n1=int(b.split('-')[1]); p=a.split('-')[0]
            codes=[f'{p}-{x:02d}' for x in range(n0,n1+1)]
        rs=[res.get(c) for c in codes]
        if all(rs):
            vs=[r[0] for r in rs]
            verdict='P' if all(v=='P' for v in vs) else 'F'
            if verdict=='P':
                note='; '.join(r[1] if r[2]=='manual' else f'RERUN 22/11 @aba738ef: script PASS ({r[2]})' for r in rs)
            else:
                note='; '.join(f'{c}: '+(why.get(c) or (rr[1] if rr[0]!="P" else "PASS"))+(f' [{rr[1]}]' if rr[0]!='P' and c in why and rr[1] and rr[2]!='manual' else '') for c,rr in zip(codes,rs))
                note='RERUN 22/11 @aba738ef FAIL — TC lệch spec/sản phẩm sau sprint 2, chờ PM quyết sửa TC/LT (proposal): '+note+f' (log evidence-2/run-logs-22nov/{rs[0][2]})'
            tcs=m.group(1)+('…TC-'+b if b else '')
            out.append(f'| {tcs} | {verdict} | {note} |'); done+=1; continue
    out.append(l)
open('smoke-2.md','w',encoding='utf-8').write('\n'.join(out))
print('đã cập nhật',done)
