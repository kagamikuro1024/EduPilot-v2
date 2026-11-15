import sys,json,re,glob,collections; sys.path.insert(0,'/tmp/qc2-A')
from gen_lib import *
B='/Users/kuro/Documents/TA_Agent_qc2/docs/regression/evidence-2/'
def parse(path):
    res={};cur=None;fl={}
    try: lines=open(path,errors='ignore').read().splitlines()
    except: return res,fl
    for l in lines:
        m=re.match(r'^== (TC-\S+)',l)
        if m: cur=m.group(1)
        m=re.match(r'^-- (TC-\S+) (PASS|FAIL|MANUAL|K)\b(.*)',l)
        if m: res[m.group(1)]=(m.group(2),m.group(3).strip())
        if cur and l.startswith('    FAIL') and cur not in fl: fl[cur]=l.strip()[5:].strip()[:150]
    return res,fl
ids={'01':'US-PG-01','02':'US-PG-02','03':'US-PG-03','04':'US-PG-04','05':'US-PG-05','06':'US-PG-06','07':'US-PG-07','gate':'GATE-PG'}
# phân loại tay: (mã -> (trạng thái, ghi chú)); ghi đè kết quả script
OV={}
def D(codes,note):
    for c in codes.split(): OV['TC-PG'+c if not c.startswith('GATE') else c]=('D',note)
def X(codes,note):
    for c in codes.split(): OV['TC-PG'+c]=('X',note)
D('01-11','biến JWT_EXPIRATION không còn (thay bằng ACCESS_TOKEN_TTL, sprint 3) — giá trị abc không còn bị báo')
D('02-01 02-02 02-19 02-21 02-70','schema 00001 → 00025: nay 60+ bảng/enum/index (sprint 3…11) — TC đếm cố định theo 00001')
D('02-42','goose version mới nhất 25 (TC đòi 1)')
D('02-48','float64 trong mã sqlc sinh ra từ cột double precision (sprint sau)')
D('02-49','chuỗi «OFFSET» chỉ trong dòng chú thích «không OFFSET» (observability.sql:2); không có OFFSET trong câu SQL — grep quá chặt')
X('02-39 02-40 02-41','`migrate down` xoá schema ep-r dùng chung — cấm chạy trên stack chung')
X('02-44','`pnpm dev:down` + xoá volume + `pnpm dev` project edupilot — cấm (stack thứ hai / xoá dữ liệu chung)')
OV['TC-PG06-01']=('F','DEF-R2-01: redocly lint openapi.yaml 1 lỗi (analytics/exams/compare: parameters null)')
OV['TC-PG06-10']=('F','DEF-R2-02: `go test ./internal/contract` rc 1 (TestStatusParity trượt 2/2 lần)')
OV['TC-PG06-11']=('F','DEF-R2-02: gồm TestStatusParity')
D('06-28','-run chỉ 2 test: TestSpec_EveryDocumentedStatusExercised cần chạy cả bộ kịch bản — TC lỗi thời theo cách chọn test')
D('01-16','APP_ENV=production thoát ngay (<100 ms): sprint sau thêm kiểm tra bắt buộc cho production (khoá/CORS) — không còn "đợi phụ thuộc"')
D('03-98','APP_ENV=production: log lỗi nêu biến production khác, không nêu APP_ENV (kiểm tra production sprint sau)')
D('04-41','BCRYPT_COST=4 — sau khi thêm APP_ENCRYPTION_KEY: PASS ở lần 2 (xem rerun2)')
CONF='01-14 01-15 01-19 01-75 01-76 01-83 04-41 02-60 02-64 02-69 03-66 03-67 05-03 05-35'.split()
for c in CONF: OV['TC-PG'+c]=('X','lần 1 trượt khi 6 luồng + nhóm khác cùng tải stack; chưa chạy lại riêng để xác nhận')
SC='05-22 05-44 05-45 05-46 07-04 07-15 07-16 07-34'.split()
for c in SC: OV['TC-PG'+c]=('X','cần gateway=2 (ep-r chạy 1 gateway) — chờ cửa sổ khoá global')
for c in '04 05 06 07'.split(): D('06-'+c,'openapi.yaml 5 đường dẫn (sprint 2) → 298 đường dẫn / 344 thao tác (sprint 3…11); 2 chữ «_test» trong mô tả')
D('06-24','thông báo lỗi hợp đồng nêu GET /api/v1/jobs/{id} — spec nay 298 route, thông báo nêu route khác')
D('04-44','TC đòi 0 đường dẫn /auth/ /me/ /admin/ trong openapi — nay có (US-PU-01…)')
D('04-45','GET /api/v1/auth/login → 405 (route auth có thật từ sprint 3), /admin/users → 200 — TC đòi 404')
D('03-51','OFFSET nằm trong câu chú thích/SQL truy vấn sau sprint 2 (xem `grep`): TC đòi 0')
D('07-01 07-02 07-66','compose nay 13 service (thêm judge, docling, mock-*, migrate…) — TC đòi 6–8 service, không python/docling')
D('07-56','nay thêm mock-graph/mock-imap công bố cổng (sprint 7/10.8)')
D('07-58','.env.example có giá trị bí mật không chứa «-dev» (khoá demo sprint sau)')
D('07-09 07-14 07-22','cổng/chứng chỉ chuẩn 80/443/5433 — ep-r chạy cổng offset 780/1143/6133 (khác biệt cấu hình stack, không phải mã)')
D('07-35 07-37','mã sản xuất nay có ghi đĩa (regress, fake llm) / thay đổi FS container worker')
D('07-38 07-39 07-40 07-41 07-43 07-44','bằng chứng là run GitHub Actions của nhánh ci/sqlc-drift (nhánh đã xoá) và ci.yml tái cấu trúc (US-CI-02)')
D('01-34','khoá Redis ngoài quy ước: Redis ep-r dùng chung với 7 nhóm khác (ep:auth:*, ep:llm:* … thêm sau sprint 2)')
D('01-68','test Ryuk: 2 lần RYUK_DISABLED=true (Makefile sprint sau)')
D('01-85','golangci-lint gọi 3 lần thay vì 2 (Makefile/CI sprint sau)')
# reruns/lock override từ log
import os
def collect(st):
    r,f=parse(B+'run-logs/%s.log'%st)
    for ex in sorted(glob.glob(B+'run-logs/%s-*.log'%st)+glob.glob(B+'run-logs/%s.rerun*.log'%st)):
        r2,f2=parse(ex); r.update(r2); f.update(f2)
    if st=='01':
        r0,f0=parse(B+'run-logs-r2stack/01-partial.log')
        for k,v in r0.items():
            if k not in r: r[k]=(v[0],'(chạy ở stack tạm ep-r2, đã dỡ)'); f[k]=f0.get(k,'')
    return r,f
rr={}
for st in ['01','02','03','04','05']:
    for ex in glob.glob(B+'run-logs/%s.rerun2.log'%st): rr.update(parse(ex)[0])
lk={};lkf={}
for ex in glob.glob(B+'run-logs/*-lock.log'):
    a,b=parse(ex); lk.update(a); lkf.update(b)
lk.pop('TC-PG02-69',None)
D('02-60','đo xmin outbox↔jobs không còn hợp lệ khi worker chạy (worker cập nhật dòng outbox khi dispatch ⇒ xmin đổi) [INFERENCE — chưa dừng worker để đo lại]')
items=[];ctr=collections.Counter()
import subprocess
DEFS={}
for st in ['01','02','03','04','05','06','07','gate']:
    r,f=collect(st); us=ids[st]
    for tc in [x.strip() for x in open('/tmp/qc2-A-ids-2-tc-%s.txt'%us)]:
        s,n=r.get(tc,(None,''))
        if tc in lk and lk[tc][0] in ('PASS','FAIL','MANUAL'):
            ls=lk[tc][0]
            if ls=='PASS': items.append((tc,'P','cửa sổ khoá global (gateway=2/chaos)',us)); continue
            if ls=='FAIL': items.append((tc,'F','CẦN TRIAGE: '+lkf.get(tc,''),us)); continue
        if tc in rr and tc in ['TC-PG'+c for c in CONF]:
            rs=rr[tc][0]
            if rs=='PASS': items.append((tc,'P','chạy lại lần 2 (đã thêm APP_ENCRYPTION_KEY vào env TC — bắt buộc từ sprint 3, proposals#4)' if tc.startswith(('TC-PG01','TC-PG04')) else 'chạy lại lần 2',us)); continue
            if rs=='FAIL': items.append((tc,'F','trượt 2 lần: '+parse(B+'run-logs/%s.rerun2.log'%st)[1].get(tc,''),us)); continue
        if tc=='TC-PG02-60' and s!='PASS': items.append((tc,'D',OV[tc][1],us)); continue
        if tc in OV and s!='PASS': items.append((tc,OV[tc][0],OV[tc][1],us)); continue
        if s=='PASS': items.append((tc,'P',n.replace('(chạy ở stack tạm ep-r2, đã dỡ)','(stack tạm ep-r2)'),us))
        elif s=='FAIL': items.append((tc,'F','CHƯA PHÂN LOẠI: '+f.get(tc,''),us))
        elif s=='MANUAL': items.append((tc,'X','bước tay (chờ 150 s/đo thủ công) — chưa chạy'+(' '+f.get(tc,'')),us))
        elif s=='K': items.append((tc,'X','cần lệnh phá huỷ/scale trên ep-r (dừng postgres/redis, gateway=2, chế độ default) — chưa chạy trong cửa sổ khoá global',us))
        else: items.append((tc,'X','chưa chạy (gate: go test toàn bộ/xoá volume — chưa tới lượt)',us))
items.sort(key=lambda x:(list(ids.values()).index(x[3]),num(x[0])))
json.dump(items,open('/tmp/qc2-A/items2.json','w'),ensure_ascii=False)
c=collections.Counter(x[1] for x in items); print(c,len(items))
for x in items:
    if x[1]=='F': print(x[0],x[2][:150])

sys.path.insert(0,'/tmp/qc2-A')
from gen_lib import smoke
how=['- `docs/regression/evidence-2/scripts/{pg01..pg07,gate-pg}.sh` (bản sao của `docs/sprints/2/qc/scripts`, chỉ chỉnh cổng/project sang ep-r: 6133/7080/9700/1143, `-p ep-r`, thêm APP_ENCRYPTION_KEY vào env nhị phân trần — bắt buộc từ sprint 3, proposals#4; docker bọc chặn lệnh phá huỷ). Log: `evidence-2/run-logs/<story>.log`.',
'- Lượt 1: 7 story chạy tuần tự/song song nhẹ trên ep-r (1 gateway, bản test). Ca cần dừng/scale/recreate (gateway=2, kill, stop postgres/redis/minio, chế độ default) chạy ở các cửa sổ khoá global (`scripts-lock/`, ≤ 400 s mỗi cửa sổ, khôi phục baseline sau mỗi cửa sổ): `run-logs/<story>-lock.log`.',
'- Ca trượt lượt 1 chạy lại lần 2 riêng (`*.rerun2.log`, `*.rerun3.log`).',
'- Go contract: `go test -race -tags testroutes ./internal/contract` (qua `gt`), thêm chạy riêng TestStatusParity.',
'- X (chờ chạy): ca còn trong chuỗi cửa sổ khoá hoặc cần xoá schema/volume ep-r dùng chung (02-39/40/41/44, gate-20) — xin PM quyết (chạy được khi không còn nhóm khác dùng ep-r).']
usn={'US-PG-01':'khung dịch vụ','US-PG-02':'schema/blob/outbox','US-PG-03':'idempotency/REST','US-PG-04':'JWT/auth','US-PG-05':'SSE','US-PG-06':'OpenAPI/contract','US-PG-07':'compose/CI','GATE-PG':'cổng PG'}
defs=['### DEF-R2-01 — openapi.yaml không qua `redocly lint` (TC-PG06-01; cũng P0-02/TC-21) — chi tiết `defects-2.md`','### DEF-R2-02 — `go test ./internal/contract` đỏ (TC-PG06-10, -11) — chi tiết `defects-2.md`']
items2=[tuple(x) for x in items]
c=smoke('2','',items2,how,usn,defs)
print(c)
json.dump(items2,open('/tmp/qc2-A/items2.json','w'),ensure_ascii=False)
