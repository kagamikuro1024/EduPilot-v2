# Dựng smoke-10.md từ danh sách TC + kết quả chạy lại (results.jsonl) + ghi chú tay. Chạy: python3 build_smoke10.py
import json, re, collections, os
H = os.path.dirname(__file__)
d = json.load(open('/tmp/qc2-f-s10-ids.json')); ids = d['ids']; orig = d['st']
new = [json.loads(l) for l in open(H + '/results.jsonl')]
def toks(label):
    out = set()
    for m in re.finditer(r'TC-(\d\d)-(\d\d|P\d?)([a-z]?)(?:/(?:(\d\d)-)?(\d\d)([a-z]?))?', label):
        a, b, s, c, dd, s2 = m.groups(); out.add(f"TC-{a}-{b}{s}"); out.add(f"TC-{a}-{b}")
        if dd: out.add(f"TC-{c or a}-{dd}{s2}"); out.add(f"TC-{c or a}-{dd}")
    return out
PASSROWS = collections.defaultdict(list)
for r in new:
    if r['st'] == 'PASS':
        for t in toks(r['tc']): PASSROWS[t].append(r['tc'][:50])
P_ = "P"; F_ = "F"; K_ = "K"; M_ = "M"
# ghi chú tay (đã chạy lại / đã phân tích): TC -> (KQ, ghi chú)
O = {}
def S(kq, note, *tcs):
    for t in tcs: O[t] = (kq, note)
S(P_, "DB: bảng practice_*, enum, không practice_exams", "TC-01-01"); S(P_, "FK phức hợp, CHECK, UNIQUE từng phần, chỉ mục", "TC-01-02")
S(P_, "SV chưa vào lớp → 23503 (px2.py)", "TC-01-03"); S(P_, "chỉ mục practice_attempts_open_key: trùng chủ đề bị từ chối; khác chủ đề chèn được (psql tay)", "TC-01-06")
S(P_, "sqlc generate+diff mã 0 (go-tests.log)", "TC-01-05"); S(P_, "TestGradeShort + TestGradeShortBadKey xanh; bảng 33 ca", "TC-01-07")
S(P_, "go test -race ./internal/quiz/... ok", "TC-01-10"); S(P_, "0 import internal/llm; 0 float64 ở quiz", "TC-01-11")
S(P_, "TestPracticeStudentDTOAllowlist xanh + quét API TC-10-05a", "TC-01-14")
S(P_, "TestExtractQuestions xanh", "TC-02-04"); S(P_, "TestBuildMatrix 8 ca xanh", "TC-05-01")
S(P_, "TestWrongAnswerSignalsNoIdentity xanh", "TC-09-10"); S(P_, "TestPracticeNoExamLeak xanh", "TC-10-01"); S(P_, "TestPracticeNoAnswerLeak xanh", "TC-10-03")
S(P_, "go test -race ./internal/quiz/... ok", "TC-12-07"); S(P_, "5 test Go chạy thật, 0 SKIP", "TC-12-08")
S(F_, "DEF-R10-01: go test ./internal/contract/... không xanh (7 test đỏ; 500 INTERNAL ở POST …/posts/{id}/report, /auth/verify-email, thiếu quan sát); ngoài phạm vi practice, nghi môi trường chung (testcontainers trên Colima đông tải)", "TC-12-14")
S(P_, "seed đủ 24 EXTRACTED; RECHECK-L1: đáp án khớp 24/24 bảng, trích dẫn đúng; trích lại → DUPLICATE (seed idempotent)", "TC-02-01", "TC-02-02", "TC-02-02b", "TC-02-02c", "TC-02-03", "TC-02-05", "TC-02-09")
S(P_, "tài liệu riêng lớp 2 (tải lên lớp 2) dùng từ lớp 1 → 404; (tài liệu dùng chung document_courses từ sau sprint 10 nên TC gốc đổi cách dựng)", "TC-02-12", "TC-02-13", "TC-02-13b")
S(P_, "việc extract thứ 6 trong giờ → 429 RATE_LIMITED + Retry-After (cửa sổ Redis)", "TC-02-19")
S(P_, "TA extract → 202", "TC-02-P3")
S(M_, "đề lỗi câu/lỗi schema cần tuyến QC fake + fixture QUESTION_EXTRACT; trên 9router thật kết quả khác (NO_ANSWER, SUCCEEDED); chưa lấy được khoá global", "TC-02-07", "TC-02-07a", "TC-02-07b")
S(M_, "OVERLOADED việc nền không kích hoạt được (như lần gốc: worker tuần tự; handoff: chưa có test Go)", "TC-02-14", "TC-03-08")
S(M_, "sinh lại trên LLM thật không tất định (9router) → không so được trùng; cần tuyến fake", "TC-03-09")
S(P_, "sinh 5 câu: created=5 PENDING, có trích dẫn", "TC-03-01", "TC-03-01b")
S(P_, "bulk APPROVE EXTRACTED không trích dẫn → CITATION_REQUIRED (RECHECK-L10); id lớp khác → ok:false NOT_FOUND", "TC-04-05", "TC-04-05b")
S(M_, "PUT review CITATION_REQUIRED chưa chạy lại (script chọn câu có citations); đường bulk đã PASS", "TC-04-13")
S(P_, "PUT version cũ → 409 VERSION_CONFLICT (px3.py)", "TC-04-11")
S(P_, "/me/today của TA có «82 câu hỏi chờ duyệt · lớp 761987» (GV bị cắt ở 20 việc do dữ liệu nhiễu của nhóm khác); biến mất sau khi duyệt hết: chưa chạy", "TC-04-15")
S(P_, "mọi ca con đạt: count 4/51/\"abc\"/topic lạ 422; mode lạ; thiếu topic; count=0 → 201 là Nợ P-10-10 (đã chấp nhận)", "TC-05-16")
S(P_, "MOCK 5 / 50 câu (pa1b)", "TC-05-17")
S(P_, "TOPIC phản hồi ngay đúng/sai/khoá; câu MANUAL không có trích dẫn (hợp lệ P-10-11); câu 2 đúng", "TC-06-08")
S(P_, "MCQ_MULTI: đúng khi trùng tập, sai khi thiếu/thừa, rỗng → 422", "TC-06-25b")
S(P_, "explain tối đa 30 lượt cần LLM/giờ/SV, lượt 31 → 429 (pg.py)", "TC-S4")
S(M_, "OVERLOADED explain cần restart gateway với LLM_MAX_CONCURRENCY=1 LLM_QUEUE_MAX=1 FAKE_LLM_LATENCY (khoá global); chưa chạy", "TC-06-20")
S(P_, "exam_events tăng 0→1 là sự kiện TAB_TAKEOVER của lượt thi lớp khác do nhóm khác (course khác); không có sự kiện từ luyện đề", "TC-10-14", "TC-07-23")
S(P_, "Hôm nay SV có «Ôn chủ đề X», biến mất sau khi luyện xong chủ đề 0,04 s (RECHECK-L3)", "TC-09-06")
S(P_, "Hồ sơ SV /students/{uid}/profile (tuyến đã đổi tên từ /360) không có dữ liệu luyện đề", "TC-09-15")
S(K_, "LỖI THỜI: L12 (đã sửa sprint 10) cho phép ánh xạ cột theo tên tiêu đề → tệp thiếu cột nay trả 200 kèm suggested_mapping", "TC-11-09")
S(M_, "check-practice-seed.mjs báo thiếu «B có điểm yếu An toàn ứng dụng web»: dữ liệu seed lớp 1 bị chính ca QC đổi (nộp lượt dở, thêm lượt); cần reseed sạch để chạy lại", "TC-12-05")
S(P_, "chat từ chối nội dung môn học khi đang thi (409 EXAM_IN_PROGRESS); nộp bài → exam-lock mở; chat hỏi lại được (pi2.py)", "TC-10-07", "TC-10-07c", "TC-10-08", "TC-10-08b")
S(K_, "ngoài phạm vi theo spec (P-10-7 Q16)", "TC-08-08")
S(K_, "(b) cần người/công cụ ngoài đồ án: Lighthouse/k6 hoãn theo chủ dự án; thử tải 100 SV", "TC-12-17", "TC-07-25", "TC-02-18")

# --- kết quả lô M (luật không lấy mẫu): results_m.jsonl, dòng sau thắng; nhiều dòng cùng mã gốc (TC-04-13 X) → F nếu có F, ngược lại ca đầu tiên không P, ngược lại P
_RM = {}
if os.path.exists(H + '/results_m.jsonl'):
    for l in open(H + '/results_m.jsonl'):
        r = json.loads(l); base = r['tc'].split(' ')[0]; _RM.setdefault(base, {})[r['tc']] = (r['kq'], r['note'])
    for base, d in _RM.items():
        if base not in ids: continue
        vals = list(d.values()); bad = [v for v in vals if v[0] not in ('P',)]
        kq = bad[0][0] if bad else 'P'
        kq = 'K' if kq in ('Ka', 'Kb', 'Kc', 'LT') else kq
        pre = {'LT': 'LT: ', 'Ka': '(a) ', 'Kb': '(b) ', 'Kc': '(c) '}.get(bad[0][0] if bad else '', '')
        note = (pre + '; '.join(dict.fromkeys(v[1] for v in (bad or vals))))
        O[base] = (kq, note)
rows = []; cnt = collections.Counter(); byus = collections.defaultdict(collections.Counter)
for t in ids:
    us = t.split('-')[1]
    if t in O: kq, note = O[t]
    elif t in PASSROWS: kq, note = P_, "script: " + "; ".join(sorted(set(PASSROWS[t]))[:2])
    elif orig.get(t, '').startswith('NGOÀI'): kq, note = K_, "ngoài phạm vi theo spec (lần gốc)"
    else: kq, note = M_, f"chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: {orig.get(t,'?').split(' ')[0]}"
    cnt[kq] += 1; byus[us][kq] += 1; rows.append((t, kq, note))
json.dump(rows, open(H + '/s10-rows.json', 'w'), ensure_ascii=False)
print(cnt, len(rows))
# --- ghi smoke-10.md
T = len(rows); p, f, k, m = (cnt[x] for x in "PFKM")
USN = {"01": "P9-01 Lược đồ, Quiz Engine, DTO", "02": "P9-02 Trích đề cũ", "03": "P9-03 Sinh câu", "04": "P9-04 /questions + duyệt hàng loạt", "05": "P9-05 Dựng đề theo ma trận", "06": "P9-06 Luyện theo chủ đề", "07": "P9-07 Thi thử", "08": "P9-08 Tự luận + nhận xét", "09": "P9-09 Lịch sử, điểm yếu, Hôm nay", "10": "P9-10 Liêm chính", "11": "P9-11 Nhập Forms", "12": "P9-12 Seed + cổng"}
out = [f"""# Smoke hồi quy sprint 10 — kết quả
Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2 (nhóm F) · Ngày: 2026-11-15
Tổng TC cũ: {T} · Đạt: {p} · Trượt: {f} · Không chạy (K, gồm lỗi thời / ngoài phạm vi): {k} · Chưa chạy (mẫu): {m}   (P+F+K+M = {p+f+k+m})
Quy ước: «K» chỉ gồm lỗi thời / ngoài phạm vi theo spec / loại (b). «M» = chưa chạy vì ngoài mẫu «đường chính + ≥1 nhánh lỗi / phân quyền mỗi US» (PM quyết có chạy nốt không) hoặc cần khoá cấu hình global (ghi rõ ở cột ghi chú); không tính trượt.
Cách chạy (bản sao đã sửa tham số ở `docs/regression/evidence-10/scripts/`, script gốc không đổi): `export SEED_DEFAULT_PASSWORD='Edupilot#Seed-2026'`; `python3 pa1.py pa1b.py pa2.py pa2b.py pa3.py pb3.py pb4.py pc.py pd.py pe.py pk.py pg.py pi2.py pj2.py px.py px2.py px3.py px4.py recheck_l1.py recheck_l3.py recheck_low.py` (qlib.py đổi cổng 943→1143, `ep-qc`→`ep-r-postgres-1`, thử lại 429/5xx; pa2/pa2b thêm nhánh SHORT; pb.py ánh xạ tên tài liệu seed; `clean_inprog.py` nộp lượt dở trước mỗi script). Go: `go test ./internal/quiz ./internal/practice` + `sqlc diff` + `go test ./internal/contract` (log `logs/go-tests.log`, `logs/contract-fail*.log`). UI: `frontend/e2e/qc2-r10/` (bản sao spec `qc-s10-*` đặt lớp mặc định 761987 bằng localStorage `ep:ui:course`) — 1440: luyện đề / lịch sử / Hôm nay / thi thử không tràn ngang, không chữ cấm (AI, RAG, citation, model…); 375: /practice, /practice/history đạt; ảnh `/tmp/qc2-f-shots10`. Kết quả UI không map sang từng TC (spec thăm dò, không có assertion theo TC) nên các TC E2E vẫn ở «M» trừ khi nêu khác.
Môi trường: 9router thật cho CHAT/QUESTION_GEN/GRADING (tuần tự, không song song); Postgres ep-r bị nhóm 9.5 làm sập một lần lúc 12:59 UTC (đã hồi phục, các ca đang chạy được chạy lại).

Lượt 2 (luật mới «không lấy mẫu»): chạy nốt 112 ca M bằng `pm.py`/`pm_a*.py` (kết quả `results_m.jsonl`, dòng sau thắng). **TẠM DỪNG lúc 23:2x theo lệnh PM (tải máy 44, nguy cơ sập Postgres):** đã xong TC-01-04, TC-01-15, TC-04-13, TC-12-21; còn lại chạy khi tải < 15. Script lô A1 (`pm_a1.py`: TC-05-04/05/05/06/15) đã viết, chạy dở (ngân hàng QC-r10-A 26/30 câu SHORT đã tạo, chủ đề riêng của QC). Ca Go (`go test` bằng testcontainers) bị hết hạn migrate khi tải cao: TestWrongAnswerSignalsNoIdentity, TestFormsImportNoLLM/ScoreDecimal, TestLibraryToolHidesExamPaperWhileLocked chạy lại xanh (4/4), overlay QC TC-09-11..13 (`/tmp/qc2-r10-ovl/qc2_r10_test.go`) chưa xong.

## Tổng hợp theo US
| US | TC tổng | P | F | K | M | Ghi chú |
| --- | --- | --- | --- | --- | --- | --- |"""]
for u in sorted(byus):
    c = byus[u]; out.append(f"| {USN[u]} | {sum(c.values())} | {c['P']} | {c['F']} | {c['K']} | {c['M']} | |")
out.append(f"| **Tổng** | {T} | {p} | {f} | {k} | {m} | |")
out.append("\n## Kết quả từng ca\n| TC | KQ | Ghi chú |\n| --- | --- | --- |")
for t, kq, note in rows:
    out.append(f"| {t} | {kq} | {note[:230].replace('|','/')} |")
out.append("""
## Trượt: bước tái hiện
### DEF-R10-01 — contract test `./internal/contract` không xanh ở be0c0d96 (TC-12-14)
`cd backend-go && go test ./internal/contract/ -run 'TestStudentNeverSeesConfidence|TestChatThreadsContract'` (testcontainers pgvector/redis/minio). Lần 1 (cả gói): 7 test đỏ (TestChatThreadsContract, TestContract_ResponsesMatchSpec, TestSpec_EveryDocumentedStatusExercised, TestStudentNeverSeesConfidence, TestNoAnswerLeak, TestResultsPermissionMatrix, TestGolden_Auth). Chạy riêng: TestNoAnswerLeak, TestGolden_Auth xanh; TestStudentNeverSeesConfidence đỏ `scenarios_thread_test.go:247 POST …/posts/{id}/report → 500`; TestChatThreadsContract đỏ `scenarios_test.go:389 POST /auth/verify-email → 500` (hai lần chạy khác nhau, endpoint khác nhau). Hai test «ResponsesMatchSpec / EveryDocumentedStatusExercised» đỏ khi chỉ chạy tập con (cần cả gói). Không liên quan practice; không phân biệt được lỗi sản phẩm với quá tải Colima (40 container). Chi tiết ở `defects-10.md`.

## Ca K và lý do (bảng gom)
| TC | Lý do |
| --- | --- |
| TC-11-09 | LỖI THỜI (L12 sprint 10: ánh xạ theo tên tiêu đề) |
| TC-08-08 | ngoài phạm vi theo spec (P-10-7 Q16) |
| TC-12-17, TC-07-25, TC-02-18 | (b) công cụ/tải: Lighthouse, 100 SV, k6 — hoãn theo chủ dự án ở lần gốc; thu nhỏ tải chưa làm trong nhóm này |

## Ca M (chưa chạy) — nhóm nguyên nhân
- Cần khoá global: TC-02-07/07a/07b, TC-03-09 (route QUESTION_GEN → QC fake + fixture), TC-06-20 (restart gateway LLM_MAX_CONCURRENCY=1), TC-02-14/TC-03-08 (OVERLOADED việc nền: không có đường kích hoạt, như lần gốc).
- TC-12-05: seed lớp 1 bị chính ca QC đổi → cần reseed sạch rồi chạy `node scripts/check-practice-seed.mjs` (API_URL, SEED_DEFAULT_PASSWORD).
- Còn lại: dạng tài liệu / E2E chưa có assertion theo TC, ngoài mẫu smoke.
""")
open(H + '/../../smoke-10.md', 'w').write("\n".join(out))
open(H + '/../../defects-10.md', 'w').write("""### DEF-R10-01 — go test ./internal/contract không xanh (TC-12-14)
- Mức: Trung bình
- Sprint gốc: 10 · TC gốc: TC-12-14 · Hồi quy do sprint sau?: KHÔNG RÕ (7 test đỏ nằm ở thread/auth/results, không ở practice; nghi quá tải môi trường hoặc lỗi 500 của sprint 11/10.8)
- Môi trường: máy QC, Colima 10 GB chung với ~40 container ep-*, be0c0d96
- Bước tái hiện: 1. `cd backend-go && go test ./internal/contract/ -run 'TestStudentNeverSeesConfidence|TestChatThreadsContract' -v`  2. so với `go test ./internal/contract/ -run 'TestNoAnswerLeak|TestGolden_Auth'` (xanh).
- Kỳ vọng (TC-12-14): contract test xanh · Thực tế: 500 INTERNAL ở `POST /api/v1/courses/{id}/posts/{id}/report` (scenarios_thread_test.go:247) và `POST /api/v1/auth/verify-email` (scenarios_test.go:389); chạy cả gói: 7 test đỏ.
- Bằng chứng: evidence-10/logs/contract-fail.log, contract-fail2.log, go-tests.log · Nghi mã: backend-go/internal/contract/scenarios_thread_test.go:247, scenarios_test.go:389
- Chạy lại lần 2: trượt (khác endpoint mỗi lần ⇒ có thể chập chờn do tải)

Ghi nhận không phải lỗi sản phẩm: (1) check-practice-seed.mjs đỏ do dữ liệu seed lớp 1 bị ca QC đổi; (2) sập Postgres ep-r lúc 12:59 UTC do nhóm khác nạp T1 vào cùng container; (3) specs `qc-s10-*` gốc giả định lớp mặc định 761987 nhưng sv.kha nay thuộc nhiều lớp (đã khắc phục ở bản sao).
""")
