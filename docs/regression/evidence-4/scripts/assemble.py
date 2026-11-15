import json,glob,os,re,collections,sys
cat=json.load(open('/tmp/qc2-b/cat.json'))
files=sorted(glob.glob('/tmp/qc2-b/results-*.json'),key=os.path.getmtime)
RES={}
for f in files:
    for k,v in json.load(open(f)).items(): RES[k]=tuple(v)
M={}
for n in (3,4):
    p=f'/tmp/qc2-b/gomap{n}.json'
    if os.path.exists(p):
        for k,v in json.load(open(p)).items():
            if v[0] in('P','F'): M[k]=v
SPEC={3:{'tokens':'P','ui-foundation':'P','shell':'P','dev-ui':'P','settings-llm':'P','a11y':'P','data-layer':'P'},4:{'account':'P','class-join':'P','me':'P','data-layer':'P'}}
OV={ # (kq, ghi chú)
'TC-P104-01':('P','72/78 ca đúng ngay lần đầu; 6 ca còn lại do QC gọi sai đường dẫn ngân sách lớp (`/admin/llm/courses/..`); đường đúng `/courses/{id}/llm-budget` đo lại ở TC-P104-35 (ADMIN ok, GV/TA/SV 403, không JWT 401)'),
'TC-P105-01':('P','/settings/llm thật: 5 phần (thêm «Quyết định nhanh», US-DM-06 AC1), 0 lồng, LEFT page-title 240, 0 nút primary hiện lúc đầu (3 primary nằm trong hộp thoại ẩn)'),
'TC-P105-02':('P','như TC-P105-01'),
'TC-P105-04':('P','8 hàng nhà cung cấp, mỗi hàng 1 nút «Test kết nối» + trạng thái «Đã kết nối/Tạm dừng do lỗi liên tiếp» + «Đang bật»'),
'TC-P105-08':('P','ô khoá: type=password, autocomplete=new-password, spellcheck=false, value rỗng'),
'TC-P105-13':('P','UI: khoá sai (stub 401) → «Khoá API không được nhà cung cấp chấp nhận. Kiểm tra lại khoá.», không thêm hàng, khoá không vào localStorage/sessionStorage'),
'TC-P105-15':('P','UI: khoá đúng → lưu, hàng mới xuất hiện, khoá không lộ ở DOM/storage'),
'TC-P105-39':('P','mở /settings/llm: 5 GET (providers, routes, decision, usage, budget), không ghi'),
'TC-P105-44':('F','chữ «EMBEDDING» thô hiện ở bảng «Mức dùng theo tác vụ» (taskLabel không có nhãn cho EMBEDDING) → DEF-R3-01'),
'TC-P105-54':('P','qc-real-f15-llm: hai tab sửa cùng nhà → tab sau 409 và nạp lại (stack ep-r)'),
'TC-P105-41':('P','qc-real-f15-llm: TEACHER/TA/SV đọc-ghi cấu hình LLM theo ma trận (GV chỉ đọc; ghi 403)'),
'TC-P105-43':('P','qc-real-f15-llm + settings-llm.spec roles: SV/TA chặn UI, 0 request admin/llm'),
'TC-P105-19':('P','settings-llm.spec (gateway giả): đổi mô hình lạc quan ≤100 ms, Hoàn tác'),
'TC-P105-21':('P','settings-llm.spec routing (gateway giả): Lên/Xuống bằng phím'),
'TC-P105-47':('P','settings-llm.spec «mobile + keyboard + axe» (gateway giả) đạt sau khi bổ sung thư mục audit của sprint 1.5 vào bản sao'),
'TC-P105-48':('P','settings-llm.spec axe 0 serious/critical (gateway giả; axe trên màn thật ADMIN/GV: chưa chạy)'),
'TC-PU01-05':('F','pnpm lint rc=0 nhưng 5 cảnh báo (cells.tsx:39, QueueTab.tsx:115/123, Landing.tsx:48, LlmSettings.tsx:13); nền sprint 3 = 0 cảnh báo → DEF-R3-03'),
'TC-GATEPU-01':('F','lint 5 cảnh báo (TC-PU01-05); pbuild/gbuild rc=0, 0 "Failed to load font"'),
'TC-PU01-11':('P','lint-selftest.sh: 7 / 7 luật ESLint + 32 / 32 phép ui-antipatterns bắt được; rc=0 (TC cũ ghi 19 phép, đã tăng)'),
'TC-GATEPU-02':('P','ui-antipatterns 32 ✓ / 0 ✗; --selftest 32/32; lint-selftest 7/7; ui-allow 9 (≤10)'),
'TC-GATEPU-03':('P','playwright desktop: ui-foundation + tokens + shell + dev-ui + settings-llm + a11y + data-layer đạt (ban đầu 5 ca fail vì bản sao thiếu audit.mjs, đã bổ sung)'),
'TC-GATEPU-04':('P','Playwright (bỏ @real) trên bản build gate: tokens/ui-foundation/shell/dev-ui/settings-llm/a11y/data-layer đạt; visual: xem TC-PU05-01'),
'TC-P208-13':('K','chưa chạy (mẫu): lần đầu hỏng do QC dùng GV đã đổi vai; không kết luận'),
'TC-P208-17':('K','chưa chạy (mẫu): như TC-P208-13'),
'TC-P208-10':('K','chưa chạy (mẫu): lần đầu hỏng do QC dùng GV đã đổi vai (2/3 thao tác đã 409)'),
'TC-P208-40':('K','chưa chạy (mẫu): kết quả 404/404/404 đúng nhưng điều kiện đếm enrollments của QC sai'),
'TC-P210-08':('P','import thật loi-2.csv → 200, created_users=28, 2 dòng lỗi không ghi; lớp có 30 enrollments (28 SV + GV + TA)'),
'TC-P209-38':('P','GV lớp mới approve/DELETE uid lớp 2 → 404/404, SV DELETE → 403; (đếm DB lớp 2 không đối chiếu)'),
}

for _k in ['TC-P103-03','TC-P103-14','TC-P103-15','TC-P103-25','TC-P103-37','TC-P103-38','TC-P103-41','TC-P104-34','TC-P104-34b','TC-P102-14','TC-P102-42']:
    OV[_k]=('K','chưa chạy (mẫu): lần chạy đầu hỏng do kịch bản QC (cầu dao nhà stub mở sớm / PUT ngân sách sai định dạng / đọc audit lệch thứ tự) — không kết luận; script /tmp/qc2-b/p10x.py chạy lại cần khoá global')
OV['TC-P204-17']=('P','POST /me/password 204 (khớp openapi và báo cáo gốc; TC ghi 200); phiên hiện tại giữ (refresh 200), phiên khác bị thu hồi (401)')
OV['TC-P204-35']=('P','preview token hợp lệ 200, sai kind 422; giới hạn 20/phút/IP: 21 lần → 429 đo ở gateway phụ giới hạn mặc định (TC-P204-12)')
OV['TC-P208-23']=('K','chưa chạy (mẫu): không có thông báo của GV2 để thử (kịch bản QC lỗi); id lạ → 404')
OV['TC-P205-18']=('P','gateway phụ giới hạn mặc định: login 10×401 rồi 429; register 5/giờ rồi 429; forgot-password 5/giờ rồi 429 (ep-r nới 1000/phút cho gateway chính)')
OV['TC-P203-18']=('P','gateway phụ: 20 token đoán → 410 rồi 429 ở lần 21')

for _k,_n in {'TC-P203-37':'qc-real-f1: đăng ký → Mailpit → xác minh → đăng nhập → / (ô nhập mã lớp)','TC-P202-51':'qc-real-f1: đăng nhập + tải lại qua Caddy cùng origin: giữ phiên, đúng 1 refresh','TC-P202-34':'qc-real-f1: không token ở storage/cookie đọc được sau đăng nhập+tải lại','TC-P204-14':'qc-real-f1: reset mật khẩu máy A → token máy B chết ngay, mở nguội về /login','TC-P206-41':'qc-real-f1: Admin mời GV → Mailpit → đặt mật khẩu → khoá → mất phiên → mở khoá','TC-P209-30':'qc-real-f1: đăng ký bằng MSSV của SV trong lớp, email khác → PENDING EMAIL_MISMATCH + cảnh báo, không thấy dữ liệu SV thật','TC-P205-01':'qc-real-f1 «dò mật khẩu»: 5 lần sai → chờ tăng dần (429/423)','TC-P205-02':'qc-real-f1: trong lúc chờ, đúng mật khẩu cũng bị từ chối','TC-P205-04':'qc-real-f1: lần sai thứ 10 → khoá 15 phút','TC-P205-05':'qc-real-f1: đang khoá, đúng mật khẩu bị từ chối','TC-P205-06':'qc-real-f1: đúng MỘT thư cảnh báo khoá'}.items(): OV[_k]=('P',_n+' (spec @real chạy trên ep-r, 6/6 đạt)')
VIS={'TC-PU01-34','TC-PU05-01','TC-PU05-04','TC-PU05-05'}
def reason(k,v):
    s=v['steps']; us=v['us']
    if k in VIS: return ('K','NỀN TẢNG: ảnh mốc visual do CI Linux; máy chạy macOS lệch kết xuất chữ (diff ~1 % do khử răng cưa, không phải lệch bố cục); cần chạy trong container Linux — chưa chạy')
    if s.startswith('**T**') or s.startswith('**T/A**'): return ('K','(b) cần người thật chấm / kịch bản tay có ảnh')
    if re.search(r'gh run|gh api|ci/ui-drift|\*\*G\*\* `gh',s) or 'GATEP' in k and re.search(r'CI|gh run',s): return ('K','(c) CI GitHub / nhánh ci/ui-drift của PR sprint 3 — ngoài phạm vi đồ án')
    if k in('TC-GATEP1-18','TC-P102-41'): return ('K','BLOCKED: ghi âm nhà thật cần khoá thật (LLM thật bị cấm; replay-only SKIP)')
    if '**L**' in s or 'k6' in s or 'lhci' in s: return ('K','chưa chạy (mẫu): tải/đo hiệu năng nặng (k6, Lighthouse) — chưa thu nhỏ để chạy trên ep-r')
    return ('K','chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US»')
OUT={}
for k,v in cat.items():
    n=v['sprint']
    if k in OV: OUT[k]=OV[k]; continue
    if k in RES and RES[k][0] in('P','F','K'): OUT[k]=RES[k]; continue
    if k in M: OUT[k]=(M[k][0],'go test: '+M[k][1]); continue
    sp=re.findall(r'\$PW (?:--grep\S* \S+ )?([a-z0-9-]+)\.spec',v['steps'])
    if sp and sp[0] in SPEC[n] and sp[0]!='visual': OUT[k]=(SPEC[n][sp[0]],f'Playwright {sp[0]}.spec.ts (bản build gate, gateway giả): 0 ca đỏ'); continue
    OUT[k]=reason(k,v)
# DEF mapping
DEF={'TC-P105-44':'DEF-R3-01','TC-PU01-05':'DEF-R3-03','TC-GATEPU-01':'DEF-R3-03','TC-P101-24':'DEF-R3-04','TC-PU01-01':'DEF-R3-02','TC-P208-02':'DEF-R4-01'}
for n in (3,4):
    ids=[k for k,v in cat.items() if v['sprint']==n]; T=len(ids)
    P=sum(1 for k in ids if OUT[k][0]=='P'); F=sum(1 for k in ids if OUT[k][0]=='F'); K=T-P-F
    kc=collections.Counter()
    for k in ids:
        if OUT[k][0]=='K':
            r=OUT[k][1]; kc['(a)']+=0
            t='(b)' if r.startswith('(b)') else '(c)' if r.startswith('(c)') else 'lỗi thời' if 'LỖI THỜI' in r else 'nền tảng' if r.startswith('NỀN TẢNG') else 'BLOCKED(khoá thật)' if r.startswith('BLOCKED') else 'chưa chạy (mẫu)'
            kc[t]+=1
    L=[f'# Smoke hồi quy sprint {n} — kết quả','Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2/SmkB · Ngày: 2026-11-15',
       f'Tổng TC cũ: {T} · Đạt: {P} · Trượt: {F} · Không chạy (K, gồm lỗi thời): {K}   (P+F+K = {P+F+K})',
       'K chia theo loại: '+'; '.join(f'{a}: {b}' for a,b in kc.items() if b)+'. Loại «chưa chạy (mẫu)» / «nền tảng» KHÔNG phải K (a)/(b)/(c) — chờ PM quyết có chạy nốt.','']
    if n==3: L+=['Cách chạy: (1) Go: `go test -race -json ./internal/llm/... ./internal/llmconfig/... ./internal/platform/crypto/...` + `-tags integration` + `-tags testroutes ./internal/store ./internal/contract` → map `-run` của TC sang kết quả từng test; (2) API trên ep-r bằng script Python (RBAC 78 ca, Test kết nối, idempotency, version, SSRF, canary, ngân sách, tuyến); stub OpenAI cục bộ `evidence-3/scripts/qstub.py` (:9741/:9742); (3) DB ep-r: ràng buộc/chỉ mục/AES-GCM giải mã chéo bằng Python; (4) gateway phụ `docker run edupilot-gateway-test` cho ca biến môi trường sai; (5) Playwright trên bản build gate sao chép sang /tmp (next standalone :3721 + api giả :3722): tokens, ui-foundation, shell, dev-ui, settings-llm, a11y, data-layer, visual; (6) UI thật ep-r `/settings/llm` 1440+390 (Playwright, ảnh ở `evidence-3/shots/`); (7) `ui-antipatterns.sh`, `--selftest`, `lint-selftest.sh`, `pnpm lint` trong bản sao.',
        'Lưu ý môi trường: Chromium 1255 chưa có lúc đầu → dùng headless-shell 1247; CORS_ORIGINS ep-r thiếu https://localhost:1143 (mọi F5 văng /login) — đã nhờ PM-QC sửa, tôi dùng route đổi Origin của /auth/refresh; khoá global đang bị nhóm khác giữ nên cụm ca đổi tuyến/ngân sách/cầu dao (TC-P102-14…19/42, TC-P103-03/14…41, TC-P104-27/34, TC-P105-20/22) CHƯA chạy ("chưa chạy (mẫu)" — script `/tmp/qc2-b/p10x.py` sẵn sàng).','']
    else: L+=['Cách chạy: (1) API ep-r bằng script Python trên tài khoản TỰ TẠO (đăng ký→Mailpit→xác minh; Admin mời GV/TA; lớp/GV/TA riêng hậu tố SK…): đăng nhập/refresh/CSRF/token giả, đăng ký/xác minh/gửi lại, quên/đặt lại mật khẩu, phiên, mời GV, khoá/mở khoá, vai, lớp (mã tham gia, duyệt, mời ra), roster import, Admin mở lớp/gán GV; (2) gateway phụ mặc định giới hạn (`docker run edupilot-gateway-test`, Redis db 9) cho ca giới hạn tốc độ IP (ep-r nới 1000/phút); (3) Playwright account/class-join/me trên bản build gate (gateway giả) + `qc-real-f1-account` / `qc-real-f2-class` (ep-r); (4) go test auth/mail/course/user/today.',
        'Lưu ý môi trường: Postgres ep-r tự phục hồi sau crash lúc ~12:59:54Z (500 INTERNAL ~1 phút); ngưỡng đăng nhập sai theo IP 30/15 phút bị chạm khi nhiều nhóm cùng IP 172.22.0.1 (xoá `ep:auth:ipfail/fail`); APP_PUBLIC_URL của thư là https://localhost (thiếu :1143).','']
    L+=['## Tổng hợp theo US','| US | TC tổng | P | F | K | Ghi chú |','| --- | --- | --- | --- | --- | --- |']
    byus=collections.OrderedDict()
    for k in ids: byus.setdefault(cat[k]['us'],[]).append(k)
    for us,ks in byus.items():
        p=sum(1 for k in ks if OUT[k][0]=='P'); f=sum(1 for k in ks if OUT[k][0]=='F'); L.append(f'| {us} | {len(ks)} | {p} | {f} | {len(ks)-p-f} | |')
    L+=['','## Kết quả từng ca','| TC | KQ | Ghi chú |','| --- | --- | --- |']
    ids_sorted=ids
    i=0
    while i<len(ids_sorted):
        k=ids_sorted[i]; r=OUT[k]
        if r[0]=='K':
            j=i
            while j+1<len(ids_sorted) and OUT[ids_sorted[j+1]]==r and cat[ids_sorted[j+1]]['us']==cat[k]['us']: j+=1
            L.append(f'| {k}'+(f'…{ids_sorted[j]}' if j>i else '')+f' | K | {r[1]} |'); i=j+1
        else:
            d=DEF.get(k); L.append(f'| {k} | {r[0]} | {r[1].replace("|","/")[:300]}'+(f' ({d})' if d and r[0]=="F" else '')+' |'); i+=1
    L+=['','## Trượt: bước tái hiện','Xem `defects-%d.md`.'%n]
    open(f'/Users/kuro/Documents/TA_Agent_qc2/docs/regression/smoke-{n}.md','w').write('\n'.join(L)+'\n')
    print(n,T,P,F,K,dict(kc))
    json.dump({k:OUT[k] for k in ids},open(f'/tmp/qc2-b/final-{n}.json','w'),ensure_ascii=False)
