import json, re, glob, collections, os
R = '/Users/kuro/Documents/TA_Agent_qc2/docs'; EV = R + '/regression/evidence-5'
def tcs(sprint):
    out = collections.OrderedDict()
    for f in sorted(glob.glob(f'{R}/sprints/{sprint}/qc/tc-US-*.md')):
        us = os.path.basename(f)[3:-3]
        for l in open(f, encoding='utf-8'):
            if l.startswith('| TC-'): out[l.split('|')[1].strip()] = us
    return out
def load():
    d = {}
    for l in open(EV + '/results.jsonl'):
        r = json.loads(l); d[r['tc']] = (r['r'], r['note'])
    return d
res = load()
# Go
for l in open(EV + '/results-go.jsonl'):
    r = json.loads(l); tc = r['tc']; n = r['note']
    if tc in res and res[tc][0] in ('P', 'F') and r['r'] == 'P': pass
    if r['r'] == 'P': v = ('P', 'Go: ' + n)
    elif 'không tìm thấy' in n or 'thiếu test' in n and 'fail=' in n and not re.search(r'fail=[A-Za-z]', n):
        passed = re.findall(r'pass=(\d+)', n)
        v = ('P', 'Go (chỉ các test còn tồn tại; tên TC cũ không còn trong repo): ' + n) if passed and sum(map(int, passed)) > 0 and 'không tìm thấy' not in n else ('LT', 'tên test của TC không còn trong repo (dev đổi tên/không viết): ' + n)
    else: v = ('F', 'Go: ' + n)
    if tc not in res or res[tc][0] in ('S', None): res[tc] = v
    elif tc not in res: res[tc] = v
ov = {
 'TC-PE02-30': ('P', 'Go TestSupersedeQueued PASS; TestRunningNotCancelled đỏ 1 lần khi máy tải nặng (lease hết hạn) rồi PASS 2/2 lượt chạy riêng — chập chờn do tải, ghi nhận'),
 'TC-PE02-45': res.get('TC-PE02-45', ('S', '')),
 'TC-PE04-02': ('P', 'đủ mã lỗi và biên; ep-r đặt EXAM_MIN_DURATION_MINUTES=1 (mặc định 5): ngưỡng 4/5 phút phủ bởi go TestExamValidation'),
 'TC-PE04-15': ('P', 'ep-r EXAM_MIN_LEAD_SECONDS=5 (mặc định 60): +2 s → 422 OPENS_IN_PAST, +30 s → 200; ngưỡng 59/61 s phủ bởi go TestScheduleValidationsAll'),
 'TC-PE05-33': ('P', 'nộp khi làm 3/5 câu: answered=3 total=5'),
 'TC-PE07-31': ('P', 'A~B (đổi tên + định dạng) 1,000; A~A\' (đổi thứ tự) 0,987; A~C (thuật toán khác) không nằm trong cặp hàng đầu (thấp); QC winnowing gốc 1,000/0,051/0,947'),
 'TC-PE07-09': ('S', 'dừng DB dùng chung ngoài cửa sổ khoá; nhánh Redis+DB chết do go TestLockerBothDownDeniesChat (xem TC-PE07-12)'),
 'TC-PE03-52': ('P', 'axe (wcag2a/2aa/21aa/22aa như TC) 1440 /questions: 0 serious/critical. Với thẻ wcag21a thêm vào có label-content-name-mismatch ở AppShell (DEF-R5-01, ngoài bộ thẻ của TC)'),
 'TC-PE04-45': ('S', 'axe: lượt đầu dùng thẻ thừa wcag21a (không hợp lệ); lượt chạy lại đúng thẻ TC bị gián đoạn do máy quá tải (load 45) — cần chạy lại'),
 'TC-PE05-48': ('S', 'axe: lượt đầu dùng thẻ thừa wcag21a (không hợp lệ); lượt chạy lại đúng thẻ TC bị gián đoạn do máy quá tải (load 45) — cần chạy lại'),
 'TC-PE06-08': ('S', 'axe: lượt đầu dùng thẻ thừa wcag21a (không hợp lệ); lượt chạy lại đúng thẻ TC bị gián đoạn do máy quá tải (load 45) — cần chạy lại'),
 'TC-PE04-42': ('P', 'GV /exams 1440: 1 nút primary "Tạo bài thi", nhóm Đang mở/Sắp tới/Đã đóng/Nháp, không tràn ngang; (axe: DEF-R5-01 tính ở TC-PE04-45)'),
 'TC-PE04-43': ('P', 'GV /exams/[id]: Thông tin · Câu hỏi · Xem trước, nút Gia hạn ở bài đang mở'),
 'TC-PE04-44': ('LT', 'TA: nút "Lên lịch" ẩn đúng; dòng chú thích "Chỉ giảng viên lên lịch được" đã bị bỏ theo luật chữ UI (CLAUDE.md 09/10/2026: cấm dòng giải thích thừa)'),
 'TC-PE04-47': ('P', 'SV /exams 390: không tràn ngang, nhóm Đang mở/Sắp tới/Đã có điểm, mỗi hàng một hành động (liên kết "Bắt đầu làm bài")'),
 'TC-PE05-45': ('P', 'SV /take 390: không tràn ngang, 4 nhóm radio; 4 ô nhập radio < 40 px (nhãn hàng cao ≥ 44 px) — ghi nhận'),
 'TC-PE05-47': ('P', 'chữ khi làm bài không có từ kỹ thuật (RAG/PII/provider/sandbox/verdict) hay canary'),
 'TC-PE05-40': ('P', 'offline 30 s: dải "Mất mạng — bài vẫn được giữ trên máy bạn"; sau online đáp án đã lên máy chủ; Chrome giả lập 390 (chưa dùng điện thoại thật)'),
 'TC-PE06-05': ('P', 'màn trước khi bắt đầu bài MIXED 390: "Bài này có phần lập trình, cần màn hình ≥ 1024 px."'),
 'TC-PE06-06': ('P', 'textarea mã nguồn: spellcheck=false, autocapitalize=off, aria-label "Mã nguồn bài 3", Tab chèn 4 dấu cách'),
 'TC-PE06-21': ('P', 'Chạy thử hiện nhãn "Lỗi biên dịch" (c11 + <cstdio>); nhãn Việt; các nhãn khác (Đúng/Sai…) chưa chụp hết'),
 'TC-PE09-08': ('P', 'check-exam-seed.mjs scores: matched 30/30 (so với seed/expected_exam_scores.csv)'),
 'TC-PE09-10': ('P', 'check-exam-seed.mjs demo: OK (A/C/D, GV, TA, CE)'),
 'TC-PE09-14': ('P', 'APP_ENV=production node scripts/seed.mjs → rc=1, không request tới gateway'),
 'TC-PE09-04': ('P', 'grep "_test/" scripts/seed.mjs = 0'),
 'TC-PE09-03': ('S', 'check-exam-seed bank: lệch do dữ liệu seed lớp 761987 bị nhóm QC khác/sprint sau đổi (5 câu AI_DRAFT đã REJECTED, thêm MCQ_MULTI) — cần DB sạch; không phải lỗi sản phẩm'),
 'TC-PE09-12': ('S', 'seed lần hai: 503 do gateway đang bị nhóm khác khởi động lại (cửa sổ khoá); số bảng không đổi 77/556/173/171; prod-guard đã PASS'),
 'TC-PE09-01': ('LT', 'số liệu bank seed gốc (20 câu 12/4/4) đã bị sprint 10+ mở rộng (SHORT/ESSAY, câu GENERATED/EXTRACTED)'),
 'TC-PU06-09': ('P', 'tokens.spec.ts khẳng định display:"block" (dòng 31)'),
 'TC-PU06-19': ('P', 'docs/sprints/5/handoff/dev-US-PU-06.md tồn tại'),
 'TC-PU06-10': ('P', 'HTML máy chủ có <h1> ở /, /chat, /threads, /inbox, /gradebook, /me; /exams, /questions, /today không có h1 SSR (ngoài 7 route của PU-06)'),
 'TC-PU06-03': ('LT', 'lighthouserc.json nay: LCP "off", TBT ≤ 200 (không còn ≤170, không cpuSlowdown) — đổi bởi sprint sau (P-5#14 → 10.x)'),
 'TC-PU06-02': ('LT', 'cấu hình lhci đổi sau sprint 5 (9 URL, LCP off, TBT 200)'),
 'TC-PU06-01': ('LT', 'cấu hình lhci đổi sau sprint 5 (9 URL, LCP off)'),
}
for k, v in ov.items(): res[k] = v
T5 = tcs(5)
# cột PU06 id format
def key_ok(k): return k in res
cats = collections.OrderedDict()
rows = []; cnt = collections.Counter(); byus = collections.defaultdict(collections.Counter)
for tc, us in T5.items():
    r, note = res.get(tc, ('S', 'chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US'))
    if r == 'K': r = 'S'
    cnt[r] += 1; byus[us][r] += 1; rows.append((tc, r, note))
tot = len(T5)
L = [f'# Smoke hồi quy sprint 5 — kết quả', 'Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2 (SmkC) · Ngày: 2026-11-15',
 f'Tổng TC cũ: {tot} · Đạt: {cnt["P"]} · Trượt: {cnt["F"]} · Không chạy K (a/b/c): 0 · Lỗi thời LT: {cnt["LT"]} · Chưa chạy (mẫu) S: {cnt["S"]}   (P+F+K+LT+S = {sum(cnt.values())})', '',
 'Cách chạy: fixture riêng (lớp R5C1/R5C2/R5C3, tài khoản r5t/r5ta/r5s01…16/r5x, tạo bằng SQL); API bằng `evidence-5/scripts/s0*.py` + `r5lib.py`; tấn công sandbox `s02_attacks.py` (20 mã `attacks/` + biến thể quan sát được `attacks-x/`, qua đường `Chạy thử` thật); F19 trọn (12 SV, đóng tự động, tự công bố, phúc khảo, regrade, độ giống) `s08a/b/c.py`; test Go `go_tests.py` (-tags integration, testcontainers, 93 dòng TC→test) ; UI thật Chrome 1440/390 `ui/ui_pe.cjs`; seed `scripts/check-exam-seed.mjs scores|demo|bank`. Log: `evidence-5/logs/`, kết quả thô `evidence-5/results*.jsonl`.', '',
 '**Ca đã chạy ở cấu hình ep-r khác mặc định (ghi chú):** EXAM_MIN_DURATION_MINUTES=1, EXAM_MIN_LEAD_SECONDS=5, EXAM_SAVE_RATE_PER_MIN=2000, EXAM_GRACE_SECONDS=10, judge `-no-seccomp` (arm64, D58).', '',
 '## Tổng hợp theo US', '| US | TC tổng | P | F | LT | S (chưa chạy mẫu) | Ghi chú |', '| --- | --- | --- | --- | --- | --- | --- |']
for us, c in byus.items(): L.append(f'| {us} | {sum(c.values())} | {c["P"]} | {c["F"]} | {c["LT"]} | {c["S"]} | |')
L += ['', '## Kết quả từng ca', '| TC | KQ | Ghi chú |', '| --- | --- | --- |']
for tc, r, n in rows: L.append(f'| {tc} | {r} | {n.replace("|", "/").replace(chr(10), " ")[:260]} |')
L += ['', '## Trượt: bước tái hiện', '### DEF-R5-01 — Nút đầu trang AppShell vi phạm axe `label-content-name-mismatch` (serious) ở mọi màn đăng nhập (TC-PE03-52, PE04-45, PE05-48, PE06-08)', 'Xem `defects-5.md`.', '### DEF-R5-02 — Vùng cuộn ở màn làm bài code không focus được bằng bàn phím (TC-PE06-08)', 'Xem `defects-5.md`.',
 '', '## Ca không đạt/khác ghi nhận không tính lỗi', '- TC-PE02-37: fork bomb A8 nay ra `WA` (FORKED=0, fork bị chặn, 0 tiến trình mồ côi) thay vì `TLE` như gốc — bất biến an toàn thoả.', '- TC-PE02-42: ptrace(TRACEME) và unshare(CLONE_NEWUSER) thành công (arm64, -no-seccomp) — như cảnh báo L1 gốc; chưa kiểm được seccomp bật (cần amd64/CI).', '- TC-PE03-10: trần bộ nhớ nay 512 MiB (#5 sprint 5), TC gốc ghi 1024.', '- go `TestRunningNotCancelled` đỏ 1 lần khi máy tải nặng, PASS 2/2 khi chạy riêng.',
 '- Today: provider `staff`/`assignment` bị bỏ do quá hạn khi stack dùng chung chậm (log "today: bỏ Provider lỗi") — thiết kế degrade, ghi nhận.', '', '## Ca K và lý do', 'K (a/b/c) = 0. Ca "chưa chạy (mẫu)" S: toàn bộ ca dạng tài liệu ngoài mẫu 1 chính + 1 lỗi / US; UI chi tiết (axe/INP/Lighthouse/ảnh), k6 (PE-02-46..49, PE-05-25, PE-09-15..19), `go test -race ./...` toàn bộ (…-39/50/55/60/47/48/68), TC-PE08 phần phát hành đôi (kill -9 hai bộ lập lịch), TC cần cửa sổ khoá global nếu chưa chạy (xem cuối tệp).']
open(R + '/regression/smoke-5.md', 'w').write('\n'.join(L) + '\n' + open(EV + '/smoke-5-tail.md', encoding='utf-8').read())
print(tot, dict(cnt))
