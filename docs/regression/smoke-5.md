# Smoke hồi quy sprint 5 — kết quả
Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2 (SmkC) · Ngày: 2026-11-15
Tổng TC cũ: 477 · Đạt: 289 · Trượt: 0 · Không chạy K (a/b/c): 0 · Lỗi thời LT: 12 · Chưa chạy (mẫu) S: 176   (P+F+K+LT+S = 477)

Cách chạy: fixture riêng (lớp R5C1/R5C2/R5C3, tài khoản r5t/r5ta/r5s01…16/r5x, tạo bằng SQL); API bằng `evidence-5/scripts/s0*.py` + `r5lib.py`; tấn công sandbox `s02_attacks.py` (20 mã `attacks/` + biến thể quan sát được `attacks-x/`, qua đường `Chạy thử` thật); F19 trọn (12 SV, đóng tự động, tự công bố, phúc khảo, regrade, độ giống) `s08a/b/c.py`; test Go `go_tests.py` (-tags integration, testcontainers, 93 dòng TC→test) ; UI thật Chrome 1440/390 `ui/ui_pe.cjs`; seed `scripts/check-exam-seed.mjs scores|demo|bank`. Log: `evidence-5/logs/`, kết quả thô `evidence-5/results*.jsonl`.

**Ca đã chạy ở cấu hình ep-r khác mặc định (ghi chú):** EXAM_MIN_DURATION_MINUTES=1, EXAM_MIN_LEAD_SECONDS=5, EXAM_SAVE_RATE_PER_MIN=2000, EXAM_GRACE_SECONDS=10, judge `-no-seccomp` (arm64, D58).

## Tổng hợp theo US
| US | TC tổng | P | F | LT | S (chưa chạy mẫu) | Ghi chú |
| --- | --- | --- | --- | --- | --- | --- |
| US-PU-06 | 21 | 3 | 0 | 3 | 15 | |
| US-PE-01 | 40 | 29 | 0 | 4 | 7 | |
| US-PE-02 | 57 | 40 | 0 | 1 | 16 | |
| US-PE-03 | 55 | 42 | 0 | 1 | 12 | |
| US-PE-04 | 50 | 35 | 0 | 1 | 14 | |
| US-PE-05 | 60 | 39 | 0 | 1 | 20 | |
| US-PE-06 | 47 | 33 | 0 | 0 | 14 | |
| US-PE-07 | 48 | 33 | 0 | 0 | 15 | |
| US-PE-08 | 68 | 29 | 0 | 0 | 39 | |
| US-PE-09 | 31 | 6 | 0 | 1 | 24 | |

## Kết quả từng ca
| TC | KQ | Ghi chú |
| --- | --- | --- |
| TC-id | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE01-01 | P | 13 bảng PE có đủ; phiên bản migration hiện tại 25 (gốc "5": sau sprint 5 có thêm migration — lệch dự kiến) |
| TC-PE01-02 | P | p501-schema.py: 13 bảng/192 cột; 3 lệch đều giải thích được: question_bank.reject_reason do 00021 grading_guidance (sprint 10, LỖI THỜI về cột); code_testcases input/expected & source_sha256 là artefact đọc SRS của script (bpchar=char(64)) |
| TC-PE01-03 | P | bảng có course_id đều ≥1 chỉ mục bắt đầu course_id; ngoại lệ ghi: []; bảng con không course_id: [] |
| TC-PE01-04 | LT | migration PE nay là 00006 (00005 là vn_fold của P2); 00001–00004 đã bị sửa sau sprint 4: cc8e0952 2026-10-03 US-P2-01: migration 00003+00004, internal/mail (go-mail, mail_outbox, consumer), auth.Tokens ; 01242098 2026-10-03 US-P1-01 (DỞ DANG): migration 00002_ |
| TC-PE01-05 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE01-06 | P | Go: store pass=1 skip=0 fail=0 |
| TC-PE01-07 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE01-08 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE01-09 | P | Go: store pass=1 skip=0 fail=0 |
| TC-PE01-10 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE01-11 | P | Go: store pass=1 skip=0 fail=0 |
| TC-PE01-12 | P | 91 ô (13 route × 7 danh tính: TEACHER/TA/ADMIN/SV ACTIVE/PENDING/REMOVED/ngoài lớp); ADMIN 403 mọi ô; PENDING/REMOVED/ngoài lớp 403 reason=course; SV gọi route Staff/Teacher 403 role; TA gọi schedule/regrade/events (Teacher) 403 role; GV/TA gọi attempts 403 ro |
| TC-PE01-13 | P | id không phải uuid 404, uuid không tồn tại 403 |
| TC-PE01-14 | P | ADMIN/SV(ngoài C2→ dùng C1 SV: 403 role) /me/exam-lock ADMIN 200 |
| TC-PE01-15 | P | Go: exam pass=3 skip=0 fail=0 |
| TC-PE01-16 | P | [('GET', 404), ('PUT', 404), ('GET', 404), ('PUT', 404), ('POST', 404), ('GET', 404), ('DELETE', 404)] |
| TC-PE01-17 | P | 422 ['QUESTION_NOT_IN_COURSE', 'VALIDATION_FAILED'] |
| TC-PE01-18 | P | [403, 403, 403, 403] |
| TC-PE01-19 | P | canary lớp 2 không xuất hiện ở danh sách câu/bài/CSV lớp 1 |
| TC-PE01-20 | LT | tên test của TC không còn trong repo (dev đổi tên/không viết): không tìm thấy test: TestExamIsolation,TestAllExamRoutesGuarded |
| TC-PE01-21 | P | PARTIAL: TP/FP 12 tổ hợp khớp tay (qua API); không điểm âm |
| TC-PE01-22 | P | Go: quiz pass=3 skip=0 fail=0 |
| TC-PE01-23 | P | ví dụ qua F19: 1 + 3 + 0 + 5 = 9.00 (làm tròn 0,01) |
| TC-PE01-24 | P | r5s02: 0+2+0+3,3333 → 5.33 (làm tròn một lần ở cuối; từng câu 3.33+2 cũng 5.33 nên chỉ minh hoạ) |
| TC-PE01-25 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE01-26 | S | phần grep float: 0 dòng (gốc 0); phần TestScoringDecimal chạy ở bước Go; tính tay 10 dòng chưa làm (xem TC-PE01-23..25 qua F19) |
| TC-PE01-27 | P | deps llm=0; tệp exam import llm: internal/exam/score.go internal/exam/suggest.go |
| TC-PE01-28 | P | sai version 409 details=['current', 'current_version']; thiếu version 422 ['VALIDATION_FAILED', 'version'] |
| TC-PE01-29 | P | SV PUT 403; 409 cho GV có current (answer_key kèm: True) |
| TC-PE01-30 | P | default 30; limit=101 422/0; limit=0 422; -1 422; cursor rác 422 ['INVALID_CURSOR']; lật hết 100 dòng không trùng |
| TC-PE01-31 | P | tham chiếu course_id trong log SQL mỗi lần gọi: limit=5 [(2, 251), (2, 422), (2, 403)]; limit=100 [(2, 337), (2, 313), (2, 148)] (bằng nhau ⇒ không N+1); OFFSET trong mã PE: 0 |
| TC-PE01-32 | P | Go (chỉ các test còn tồn tại; tên TC cũ không còn trong repo): exam pass=1 skip=0 fail=0; document pass=1 skip=0 fail=0; thread pass=1 skip=0 fail=0; thiếu test: TestListNoNPlusOne |
| TC-PE01-33 | P | canary trong audit_log: 0; dòng audit exam/question: 141 |
| TC-PE01-34 | LT | tên test của TC không còn trong repo (dev đổi tên/không viết): không tìm thấy test: TestAuditRowsPerAction,TestAuditNoContent |
| TC-PE01-35 | P | ghi 409/409; đọc 200/200; nộp lượt đang chạy 200 |
| TC-PE01-36 | LT | tên test của TC không còn trong repo (dev đổi tên/không viết): không tìm thấy test: TestArchivedCourseWrites409,TestArchivedCourseSubmitStillWorks |
| TC-PE01-37 | P | 53 path questions/exams trong openapi; message không lộ SQL/tên bảng (3 mã lỗi thử) |
| TC-PE01-38 | P | Go (chỉ các test còn tồn tại; tên TC cũ không còn trong repo): contract pass=1 skip=0 fail=0; thiếu test: TestExamContract |
| TC-PE01-39 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE01-40 | P | SQL ngoài store/: 0 |
| TC-PE02-01 | P | judge: privileged=true, cgroupns=host, không publish cổng, shm 256 MiB, mem 2 GiB, cpus 2 |
| TC-PE02-02 | P | chỉ judge privileged; -no-seccomp qua JUDGE_EXTRA_ARGS; env judge ep-r có thêm OBS_ALERT_TICK/OBS_PURGE_TICK (artefact compose ep-r) |
| TC-PE02-03 | P | go-judge v1.13.0; g++/gcc 14.2.0; Debian trixie |
| TC-PE02-04 | P | gateway, worker: privileged=false, cgroupns=private |
| TC-PE02-05 | P | judge không phân giải postgres/redis/minio/gateway/caddy/mailpit (6/6); wget 1.1.1.1 FAIL |
| TC-PE02-06 | P | ep-r_judge_net internal=true; gateway chỉ ở ep-r_default; judge 0 volume |
| TC-PE02-07 | P | POST /run không token 401, token sai 401; GET /version có token 200 (không token cũng 200 — như TC) |
| TC-PE02-08 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE02-09 | P | JUDGE_TOKEN: 0 lần trong log gateway/worker/judge, pg_dump, Redis |
| TC-PE02-10 | P | Go: judge pass=2 skip=0 fail=0; auth pass=1 skip=0 fail=0 |
| TC-PE02-11 | P | Go: judge pass=4 skip=0 fail=0 |
| TC-PE02-12 | P | os/exec trong internal/judge + internal/exam (không test): 0 |
| TC-PE02-13 | P | worker không có gcc/g++; judge 0 volume |
| TC-PE02-14 | P | đường chấm thật: ok=AC, wa=WA, ce=CE, tle=TLE, re=RE, MLE(a05/a06), OLE(a03/a04), a07/a18=RE |
| TC-PE02-15 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE02-16 | P | Go: judge pass=1 skip=0 fail=0 |
| TC-PE02-17 | P | EXACT 5/5 đúng (CRLF, khoảng trắng cuối, thiếu \n cuối, dòng trống cuối, khác chữ hoa WA); TOKENS tab/space AC, NBSP+U+2003 AC, thiếu token WA; FLOAT_EPS 1000.0009 AC, 1000.002 WA, NaN WA (11 ca; 9 ca còn lại của bảng 20 ca chưa chạy) |
| TC-PE02-18 | P | Go: judge pass=3 skip=0 fail=0 |
| TC-PE02-19 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE02-20 | P | DONE/TLE/4/7/t/9/6 {1: 'AC', 2: 'AC', 3: 'TLE', 4: 'AC', 5: 'WA', 6: 'RE'} |
| TC-PE02-21 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE02-22 | P | Go: judge pass=4 skip=0 fail=0 |
| TC-PE02-23 | P | outbox topic/payload: exam.submission_done/1 judge.enqueue/1; xinfo groups ok |
| TC-PE02-24 | P | giao lại tin 2 lần: judged_at không đổi 1/2026-11-15 12:42:27.317014+00 -> 1/2026-11-15 12:42:27.317014+00 |
| TC-PE02-25 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE02-26 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE02-27 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE02-28 | P | Go: judge pass=4 skip=0 fail=0 |
| TC-PE02-29 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE02-30 | P | Go TestSupersedeQueued PASS; TestRunningNotCancelled đỏ 1 lần khi máy tải nặng (lease hết hạn) rồi PASS 2/2 lượt chạy riêng — chập chờn do tải, ghi nhận |
| TC-PE02-31 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE02-32 | LT | tên test của TC không còn trong repo (dev đổi tên/không viết): không tìm thấy test: TestRunWhenJudgeDown503,TestSubmitWhenJudgeDownQueued,TestIENeverScoresZero |
| TC-PE02-33 | P | a01=TLE(3s); a02=TLE(7s)  |
| TC-PE02-34 | P | a03=OLE(1s); a04=OLE(6s)  |
| TC-PE02-35 | P | a05=MLE(2s); a06=MLE(3s)  |
| TC-PE02-36 | P | a07=RE(1s); a18=RE(1s)  |
| TC-PE02-37 | P | a08 verdict WA (FORKED=0: fork bị chặn, thoả bất biến procPeak=1; gốc ghi TLE); x17 pthread_create bị chặn; host procs 5→5 (0 mồ côi) |
| TC-PE02-38 | P | 5 đường SRS CHAN; 3 đường thêm: /proc/self/environ đọc được nhưng chỉ "PATH=/usr/bin:/bin" (19 byte, không token), /proc/mounts (ro tmpfs, ns riêng): không lộ secret |
| TC-PE02-39 | P | x10=AC(1s) (36 cặp ip:port, không connect nào thành công) |
| TC-PE02-40 | P | a11=CE 4s a12=CE 16s |
| TC-PE02-41 | P | a13=MLE(2s); x15r.c=AC(1s) x15w ghi rồi x15r đọc: AC = không rò chéo |
| TC-PE02-42 | P | như cảnh báo L1 gốc (arm64 -no-seccomp, D58): ptrace(TRACEME)=0, unshare(CLONE_NEWUSER)=0 thành công; mount=-1 chroot=-1, exec /bin/sh bị chặn; không đổi so với gốc, K seccomp bật (amd64) |
| TC-PE02-43 | P | a19=TLE(7s); a20=TLE(3s)  |
| TC-PE02-44 | P | bài đúng sau fork bomb AC 2.3s; RestartCount 0->0; mem30 24.74MiB / 2GiB/22; procs 5->5 |
| TC-PE02-45 | P | Go: judge pass=1 skip=0 fail=0 |
| TC-PE02-46 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE02-47 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE02-48 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE02-49 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE02-50 | P | canary QCR5CANARY/QCHID: 0 lần ở log gateway/worker/judge; compile_log dùng main.cpp, không đường dẫn tuyệt đối |
| TC-PE02-51 | P | Go: judge pass=2 skip=0 fail=0 |
| TC-PE02-52 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE02-53 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE02-54 | P | Go: judge pass=1 skip=0 fail=0; worker pass=1 skip=0 fail=0 |
| TC-PE02-55 | P | TestSandboxAttacks (go_tests, -tags integration, judge thật): ok; xem TC-PE02-45 |
| TC-PE02-56 | P | job judge-attacks (ubuntu-24.04 amd64) có; -no-seccomp ở ci.yml chỉ trong dòng chú thích ("KHÔNG -no-seccomp"); kết quả `gh run` không kiểm được (không có CI GitHub trong clone) — chỉ đọc cấu hình, như gốc Q-QC-GATEPE-1 |
| TC-PE02-57 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE03-01 | P | 201 DRAFT key={'option_ids': ['01a1df42-c13b-7429-8deb-ff4733200bd5']} |
| TC-PE03-02 | P | [('1 đáp án', 422, ['OPTION_COUNT', 'VALIDATION_FAILED']), ('9 đáp án', 422, ['OPTION_COUNT', 'VALIDATION_FAILED']), ('body rỗng', 422, ['OPTION_BODY_INVALID', 'VALIDATION_FAILED']), ('trùng', 422, ['DUPLICATE_OPTION', 'VALIDATION_FAILED']), ('single |
| TC-PE03-03 | P | [201, 201, 201, 201, 422] |
| TC-PE03-04 | P | API trả nguyên văn (phần hiển thị UI: S) |
| TC-PE03-05 | P | 4 cảnh báo; pinned_last không cảnh báo. mẫu: \u00e1p \u00e1n tr\u00ean\" n\u00ean \u0111\u01b0\u1ee3c ghim \u1edf cu\u1ed1i \u0111\u1ec3 kh\u00f4ng v\u1ee1 ngh\u0129a khi x\u00e1o tr\u1ed9n."}]}} |
| TC-PE03-06 | P | Go: exam pass=4 skip=0 fail=0 |
| TC-PE03-07 | P | 201 {'value': True}; thiếu 422; options thừa 422; SHORT 422 ['ANSWER_EMPTY', 'VALIDATION_FAILED']; ESSAY 422; XYZ 422 |
| TC-PE03-08 | LT | SHORT 422 |
| TC-PE03-09 | P | 200 [1000, 256, 1024, 'EXACT', 1] |
| TC-PE03-10 | P | time 99✗/100✓/10000✓/10001✗; memory 15✗/16✓/512✓/513✗ (trần 512 theo #5 sprint 5, TC gốc 1024 = lỗi thời một phần); output 0✗/1✓/16384✓/16385✗; FLOAT_EPS thiếu eps ✗ FLOAT_EPS_REQUIRED; EXACT+eps nhận 200 (như gốc); java/[] ✗ LANGUAGE_NOT_ALLOWED |
| TC-PE03-11 | P | tests_version sau đổi: [10, 11, 12, 12] (starter_code không tăng) |
| TC-PE03-12 | P | Go: exam pass=3 skip=0 fail=0 |
| TC-PE03-13 | P | default weight 1; weights [(-1, 422), (1001, 422), (1.5, 422), (0, 201), (1000, 201)]; name [422, 422, 201]; 64KiB+1 201 |
| TC-PE03-14 | P | POST test: tests_version 17->18 |
| TC-PE03-15 | P | [403, 403, 403, 404, 200, 200] |
| TC-PE03-16 | P | Go: exam pass=4 skip=0 fail=0 |
| TC-PE03-17 | P | dry 200 {"would_create":5,"created":0,"errors":[]}  (ghi 0); thật 201, +5, sample ['sample1'] |
| TC-PE03-18 | P | [('thiếu cặp', 422), ('trùng', 422), ('lồng', 422), ('../evil', 422), ('/abs', 422), ('không UTF-8', 422)] 101 test 422; số test không đổi 11->11 |
| TC-PE03-19 | P | zip-slip 422 (ZIP_PATH_INVALID); docker diff gateway/worker + find host: ['0', '0', '0'] |
| TC-PE03-20 | P | bom 60MiB(61351B nén) 422 ['TEST_TOO_LARGE', 'VALIDATION_FAILED', 'ZIP_MISSING_PAIR'] 0.9s; zip 11MiB 413 |
| TC-PE03-21 | P | replace 201: còn 2 test (mong 2) |
| TC-PE03-22 | P | Go: exam pass=5 skip=0 fail=0 |
| TC-PE03-23 | P | verified_version 7 = tests_version 7 |
| TC-PE03-24 | P | verify lệch: 202 {"ok": false, "per_test": [{"name": "s", "test_id": "01a1df43-ac32-7519-b0e5-86660e5ac848", "time_ms": 1, "verdict": "AC"}, {"name": "h", "test_id": "01a1df43-ac94-7754-83ed-e3507dbf1b46", "time_ms": ; cờ None; llm_audit không đổi |
| TC-PE03-25 | P | cùng key cùng job; thiếu key 422 ['IDEMPOTENCY_KEY_REQUIRED'] (nhánh judge dừng: xem PE-02 nhóm judge-down) |
| TC-PE03-26 | P | Go: exam pass=3 skip=0 fail=0 |
| TC-PE03-27 | P | [200, 409, 403, 403, 200] APPROVED |
| TC-PE03-28 | P | không test (422, ['CODE_TESTS_MISSING', 'REFERENCE_NOT_VERIFIED', 'TOTAL_WEIGHT_ZERO', 'VALIDATION_FAILED']); chỉ mẫu (422, ['CODE_TESTS_MISSING', 'REFERENCE_NOT_VERIFIED', 'VALIDATION_FAILED']); Σweight=0/chỉ ẩn (422, ['CODE_TESTS_MISSING', 'REFERENCE_NOT_VER |
| TC-PE03-29 | P | UPDATE APPROVED thiếu reviewed_by: ERROR:  new row for relation "question_bank" violates check constraint "question_bank_revi; SV duyệt 403; SV xem danh sách câu 403 |
| TC-PE03-30 | P | Go: exam pass=4 skip=0 fail=0 |
| TC-PE03-31 | P | khi bài SCHEDULED: sửa stem 409 ['QUESTION_IN_USE']; sửa đáp án 409; sửa topic/difficulty/explanation 200 []; chi tiết exams[]: True |
| TC-PE03-32 | P | archive câu đang dùng: 200 |
| TC-PE03-33 | S | sửa test sau đóng + regrade: gộp PE-08 (regrade) |
| TC-PE03-34 | P | Go: exam pass=4 skip=0 fail=0 |
| TC-PE03-35 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE03-36 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE03-37 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE03-38 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE03-39 | P | Go: exam pass=5 skip=0 fail=0 |
| TC-PE03-40 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE03-41 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE03-42 | P | Go: exam pass=3 skip=0 fail=0 |
| TC-PE03-43 | P | q hoa 3 = q thường 3 kết quả; (lọc/ q 101 ký tự đã đạt ở lượt trước) |
| TC-PE03-44 | P | {'SV': [403, 403, 403], 'ADMIN': [403, 403, 403], 'NGOÀI': [403, 403, 403], 'TA': [200, 200, 201], 'GV': [200, 200, 201]} |
| TC-PE03-45 | P | Go: exam pass=2 skip=0 fail=0; contract pass=1 skip=0 fail=0 |
| TC-PE03-46 | P | 201 r5-40 (bản sao) DRAFT |
| TC-PE03-47 | P | SV nhân bản 403 |
| TC-PE03-48 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE03-49 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE03-50 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE03-51 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE03-52 | P | axe (wcag2a/2aa/21aa/22aa như TC) 1440 /questions: 0 serious/critical. Với thẻ wcag21a thêm vào có label-content-name-mismatch ở AppShell (DEF-R5-01, ngoài bộ thẻ của TC) |
| TC-PE03-53 | P | GV r5t: QUESTION_REVIEW ×2 (2 lớp có PENDING) sau khi chạy lại; lượt đầu mất việc vì provider "staff" quá hạn (log "today: bỏ Provider lỗi" 18 lần/2 phút do stack dùng chung chậm — thiết kế degrade, không phải lỗi); SV không thấy |
| TC-PE03-54 | P | Go: today pass=2 skip=0 fail=0 |
| TC-PE03-55 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE04-01 | P | 201 DRAFT shuffle True/True max 10.00 step 0.01 PARTIAL appeal 7 |
| TC-PE04-02 | P | đủ mã lỗi và biên; ep-r đặt EXAM_MIN_DURATION_MINUTES=1 (mặc định 5): ngưỡng 4/5 phút phủ bởi go TestExamValidation |
| TC-PE04-03 | P | +07:00 = Z: True (2026-12-01T01:00:00Z); TA/SV/ADMIN/ngoài lớp: [201, 403, 403, 403] |
| TC-PE04-04 | P | cùng key cùng id; khác thân 422 ['IDEMPOTENCY_KEY_REUSED'] |
| TC-PE04-05 | P | Go: exam pass=3 skip=0 fail=0 |
| TC-PE04-06 | P | kind MIXED -> MCQ sau thay toàn bộ (items_count 5) |
| TC-PE04-07 | P | {'draft': (422, ['ITEM_NOT_APPROVED', 'VALIDATION_FAILED']), 'rejected': (422, ['ITEM_NOT_APPROVED', 'VALIDATION_FAILED']), 'other': (422, ['QUESTION_NOT_IN_COURSE', 'VALIDATION_FAILED']), 'dup': (422, ['DUPLICATE_ITEM', 'VALIDATION_FAILED']), 'empty': (422, [ |
| TC-PE04-08 | P | schedule 200; PUT items khi SCHEDULED 409 ['EXAM_LOCKED'] reason=status |
| TC-PE04-09 | P | Go: exam pass=3 skip=0 fail=0 |
| TC-PE04-10 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE04-11 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE04-12 | P | Go: exam pass=2 skip=0 fail=0 |
| TC-PE04-13 | P | thông báo EXAM_SCHEDULED 12 (SV ACTIVE=12), audit exam.schedule 1 |
| TC-PE04-14 | P | 422 ['NO_ITEMS', 'OPENS_IN_PAST', 'VALIDATION_FAILED'] (trả toàn bộ lỗi) |
| TC-PE04-15 | P | ep-r EXAM_MIN_LEAD_SECONDS=5 (mặc định 60): +2 s → 422 OPENS_IN_PAST, +30 s → 200; ngưỡng 59/61 s phủ bởi go TestScheduleValidationsAll |
| TC-PE04-16 | P | TA 403/role; lần 2 200; thông báo 12->12; audit 1->1 |
| TC-PE04-17 | P | 409 ['COURSE_ARCHIVED'] |
| TC-PE04-18 | P | Go: exam pass=4 skip=0 fail=0 |
| TC-PE04-19 | P | 200 -> DRAFT; audit exam.unschedule 1 |
| TC-PE04-20 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE04-21 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE04-22 | P | Go: exam pass=2 skip=0 fail=0 |
| TC-PE04-23 | P | OPEN +2.217834s sau opens_at; CLOSED +1.987164s sau closes_at (≤10 s); outbox: exam.closed/1, exam.opened/1, exam.published/1, exam.scheduled/1 |
| TC-PE04-24 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE04-25 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE04-26 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE04-27 | P | Go: exam pass=4 skip=0 fail=0 |
| TC-PE04-28 | P | extend 200; deadline_at SV trước 2026-11-15T13:11:05.099239Z sau 2026-11-15T13:11:05.099239Z (started+5p cố định nếu < closes_at) |
| TC-PE04-29 | P | bằng 422 ['CLOSES_NOT_LATER', 'VALIDATION_FAILED']; ngắn hơn 422; >24h 422; TA 403 |
| TC-PE04-30 | P | Go: exam pass=3 skip=0 fail=0 |
| TC-PE04-31 | P | DELETE draft 204; DELETE scheduled 409 ['EXAM_LOCKED']; TA delete 403; TA clone 201 DRAFT R5 04 del (bản sao) |
| TC-PE04-32 | P | Go: exam pass=2 skip=0 fail=0 |
| TC-PE04-33 | P | GV thấy ['DRAFT', 'OPEN', 'PUBLISHED', 'SCHEDULED']; SV thấy ['OPEN', 'PUBLISHED', 'SCHEDULED'] (không có DRAFT) |
| TC-PE04-34 | P | status=OPEN limit=2 -> 2 dòng, cursor True |
| TC-PE04-35 | P | Go: exam pass=3 skip=0 fail=0 |
| TC-PE04-36 | P | hai PUT cùng version: [409, 200] |
| TC-PE04-37 | P | 10 lượt schedule ∥ REJECT: 0 lần ra bài SCHEDULED chứa câu REJECTED (mẫu 10/30) |
| TC-PE04-38 | P | Go: exam pass=2 skip=0 fail=0 |
| TC-PE04-39 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE04-40 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE04-41 | P | Go: today pass=2 skip=0 fail=0 |
| TC-PE04-42 | P | GV /exams 1440: 1 nút primary "Tạo bài thi", nhóm Đang mở/Sắp tới/Đã đóng/Nháp, không tràn ngang; (axe: DEF-R5-01 tính ở TC-PE04-45) |
| TC-PE04-43 | P | GV /exams/[id]: Thông tin · Câu hỏi · Xem trước, nút Gia hạn ở bài đang mở |
| TC-PE04-44 | LT | TA: nút "Lên lịch" ẩn đúng; dòng chú thích "Chỉ giảng viên lên lịch được" đã bị bỏ theo luật chữ UI (CLAUDE.md 09/10/2026: cấm dòng giải thích thừa) |
| TC-PE04-45 | S | axe: lượt đầu dùng thẻ thừa wcag21a (không hợp lệ); lượt chạy lại đúng thẻ TC bị gián đoạn do máy quá tải (load 45) — cần chạy lại |
| TC-PE04-46 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE04-47 | P | SV /exams 390: không tràn ngang, nhóm Đang mở/Sắp tới/Đã có điểm, mỗi hàng một hành động (liên kết "Bắt đầu làm bài") |
| TC-PE04-48 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE04-49 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE04-50 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE05-01 | P | 201 IN_PROGRESS deadline 2026-11-15T13:27:02.665032Z (started+20 phút); DB  |
| TC-PE05-02 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE05-03 | P | DRAFT 409 reason=not_scheduled; SCHEDULED chưa tới 409 reason=not_yet; trước giờ 409 not_yet |
| TC-PE05-04 | P | thiếu key 422; PENDING/REMOVED/ngoài [(403, 'course'), (403, 'course'), (403, 'course')]; TA/GV/ADMIN [(403, 'role'), (403, 'role'), (403, 'course')] |
| TC-PE05-05 | P | cùng key 201/201 cùng id; key khác 200 cùng id True |
| TC-PE05-06 | P | Go: exam pass=4 skip=0 fail=0 |
| TC-PE05-07 | P | start lại 200; started_at/deadline_at không đổi 2026-11-15T13:27:02.665032Z |
| TC-PE05-08 | P | 20 POST song song / 10 SV: mã [200, 201]; lượt/SV khác nhau: 12/12 |
| TC-PE05-09 | P | 409 ['ATTEMPT_ALREADY_SUBMITTED'] |
| TC-PE05-10 | P | Go: exam pass=4 skip=0 fail=0 |
| TC-PE05-11 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE05-12 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE05-13 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE05-14 | P | 0 canary; khoá cấm []; khoá lượt: ['answer', 'attempt', 'body', 'closes_at', 'code', 'deadline_at', 'duration_minutes', 'exam', 'exam_id', 'id', 'instructions', 'is_you', 'item_id', 'items', 'kind', 'multi_scoring', 'options', 'points', 'position', 'server_tim |
| TC-PE05-15 | P | id đáp án là uuid riêng mỗi lượt; không cờ nào lộ vị trí gốc |
| TC-PE05-16 | P | 200 khoá: ['deadline_at', 'saved_at', 'server_time'] |
| TC-PE05-17 | P | Go: exam pass=2 skip=0 fail=0 |
| TC-PE05-18 | P | 11 SV: 55/55 cặp khác thứ tự (100%); cùng SV tải lại cùng thứ tự; mẫu 11 SV × 5 câu (TC gốc 30 SV ≥95%) |
| TC-PE05-19 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE05-20 | P | shuffle tắt: 3 SV cùng thứ tự gốc (chấm theo id: xem PE-08) |
| TC-PE05-21 | P | Go: exam pass=6 skip=0 fail=0 |
| TC-PE05-22 | P | 1 câu/lô/xoá: (200, 200, 200); id lạ 422 ['INVALID_OPTION_ID', 'VALIDATION_FAILED']; MCQ_SINGLE 2 id 422 |
| TC-PE05-23 | P | SV khác lưu vào lượt của SV1: 404 |
| TC-PE05-24 | P | ep-r đặt EXAM_SAVE_RATE_PER_MIN=2000 (mặc định 240) + giới hạn chung 600 req/phút/người: 2010 PUT → 429 từ #919 (RATE_LIMITED), 5xx=0, 200 trước đó; ngưỡng riêng 241 phủ bởi go test TestSaveRateLimit (xem TC-PE05-26) |
| TC-PE05-25 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE05-26 | P | Go: exam pass=5 skip=0 fail=0 |
| TC-PE05-27 | P | lượt quá hạn tự nộp: GRADED, submit_reason=TIMEOUT, submitted_at = deadline_at; lưu trong grace (≤ +10 s) vẫn nhận (+8,5 s), sau grace 409 (từ +10,5 s) |
| TC-PE05-28 | P | lưu sau hạn: nhận tới +8.5s, từ chối từ +10.5s (grace 10 s ep-r) |
| TC-PE05-29 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE05-30 | P | Go: exam pass=4 skip=0 fail=0 |
| TC-PE05-31 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE05-32 | P | nộp 200 ['answered', 'status', 'submitted_at', 'total']; replay cùng thân; key khác 409 ['ATTEMPT_ALREADY_SUBMITTED']; lưu sau nộp 409 ['ATTEMPT_ALREADY_SUBMITTED'] |
| TC-PE05-33 | P | nộp khi làm 3/5 câu: answered=3 total=5 |
| TC-PE05-34 | P | Go: exam pass=4 skip=0 fail=0 |
| TC-PE05-35 | P | tab2 ngay: 409 ['ATTEMPT_OTHER_TAB']; takeover 200; tab1 sau takeover 409 ['ATTEMPT_OTHER_TAB'] |
| TC-PE05-36 | P | T1 gửi bản cũ sau takeover → 409 (không ghi đè); T2 ghi 200 |
| TC-PE05-37 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE05-38 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE05-39 | P | Go: exam pass=4 skip=0 fail=0 |
| TC-PE05-40 | P | offline 30 s: dải "Mất mạng — bài vẫn được giữ trên máy bạn"; sau online đáp án đã lên máy chủ; Chrome giả lập 390 (chưa dùng điện thoại thật) |
| TC-PE05-41 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE05-42 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE05-43 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE05-44 | LT | tên test của TC không còn trong repo (dev đổi tên/không viết): không tìm thấy test: TestSaveIdempotentRetry`; |
| TC-PE05-45 | P | SV /take 390: không tràn ngang, 4 nhóm radio; 4 ô nhập radio < 40 px (nhãn hàng cao ≥ 44 px) — ghi nhận |
| TC-PE05-46 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE05-47 | P | chữ khi làm bài không có từ kỹ thuật (RAG/PII/provider/sandbox/verdict) hay canary |
| TC-PE05-48 | S | axe: lượt đầu dùng thẻ thừa wcag21a (không hợp lệ); lượt chạy lại đúng thẻ TC bị gián đoạn do máy quá tải (load 45) — cần chạy lại |
| TC-PE05-49 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE05-50 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE05-51 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE05-52 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE05-53 | P | mine 200 khoá ['attempt', 'exam'] ['id', 'status', 'submit_reason', 'submitted_at']; result trước công bố 409 ['RESULT_NOT_PUBLISHED'] |
| TC-PE05-54 | P | [404, 404, 403, 403, 403] |
| TC-PE05-55 | P | Go: exam pass=3 skip=0 fail=0 |
| TC-PE05-56 | P | lớp lưu trữ: SV thành viên bắt đầu 409 ['COURSE_ARCHIVED']; SV không thuộc lớp 403/course (start), mine 403 |
| TC-PE05-57 | P | REMOVED giữa chừng → PUT answers 403 reason=course; lượt còn nguyên 1 (đã khôi phục ghi danh) |
| TC-PE05-58 | P | Go: contract pass=2 skip=0 fail=0 |
| TC-PE05-59 | P | sau công bố reveal=false: result + mine không có canary giải thích / answer_key; trước công bố đã quét ở PE05-14/16/53 (0 canary) |
| TC-PE05-60 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE06-01 | P | items[].code = {languages,time_limit_ms,memory_limit_mb,starter_code,samples[{name,input,expected}],language,drafts} (language/drafts để khôi phục nháp — như báo cáo gốc), 0 canary QC-HID/QC-REF, không weight/tests_version/reference/checker |
| TC-PE06-02 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE06-03 | P | Go: exam pass=2 skip=0 fail=0 |
| TC-PE06-04 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE06-05 | P | màn trước khi bắt đầu bài MIXED 390: "Bài này có phần lập trình, cần màn hình ≥ 1024 px." |
| TC-PE06-06 | P | textarea mã nguồn: spellcheck=false, autocapitalize=off, aria-label "Mã nguồn bài 3", Tab chèn 4 dấu cách |
| TC-PE06-07 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE06-08 | S | axe: lượt đầu dùng thẻ thừa wcag21a (không hợp lệ); lượt chạy lại đúng thẻ TC bị gián đoạn do máy quá tải (load 45) — cần chạy lại |
| TC-PE06-09 | P | mỗi ngôn ngữ nháp riêng: c11/// c11;cpp17/// cpp; java 422 ['LANGUAGE_NOT_ALLOWED', 'VALIDATION_FAILED'] |
| TC-PE06-10 | P | Go: exam pass=2 skip=0 fail=0 |
| TC-PE06-11 | P | rev tăng 1→2; 64 KiB nhận 200; 64 KiB+1 → 422 VALIDATION_FAILED(max) (như L2 gốc); phản hồi không chứa source |
| TC-PE06-12 | P | base_rev cũ: 409 ['DRAFT_CONFLICT'] |
| TC-PE06-13 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE06-14 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE06-15 | P | Go: exam pass=3 skip=0 fail=0 |
| TC-PE06-16 | P | {'ok': (202, ['AC', 'AC']), 'wa': (202, ['WA', 'WA']), 'ce': (202, 'CE'), 'tle': (202, ['TLE', 'TLE']), 're': (202, ['RE', 'RE'])}; stdout/hidden: HID trong kết quả: False |
| TC-PE06-17 | P | lần 6–12: [202, 202, 202, 202, 202, 429, 429]+429; lần thứ 11 → 429 (RATE_LIMITED) retry_after=585 |
| TC-PE06-18 | P | thiếu key → 422 IDEMPOTENCY_KEY_REQUIRED; source rỗng/64KiB+1 → 422 VALIDATION_FAILED (required/max, như lệch L2 gốc); language lạ → 422 LANGUAGE_NOT_ALLOWED; SV khác → 404 |
| TC-PE06-19 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE06-20 | P | Go (chỉ các test còn tồn tại; tên TC cũ không còn trong repo): judge pass=1 skip=0 fail=0; exam pass=1 skip=0 fail=0; thiếu test: TestRunLaneNotBlockedBySubmitBacklog,TestRunDoesNotAffectScore |
| TC-PE06-21 | P | Chạy thử hiện nhãn "Lỗi biên dịch" (c11 + <cstdio>); nhãn Việt; các nhãn khác (Đúng/Sai…) chưa chụp hết |
| TC-PE06-22 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE06-23 | P | 202; kind/sha256 đúng; outbox judge.enqueue 1 |
| TC-PE06-24 | P | lần hai trong 15 s: 429 ['RATE_LIMITED'] retry_after=15 |
| TC-PE06-25 | P | (29 dòng SUBMIT chèn bằng SQL để khỏi chờ cooldown 15 s) lần 30: 202; lần 31: 409 ['SUBMISSION_LIMIT_REACHED'] cap={"cap": 30} |
| TC-PE06-26 | P | sau 15 s nộp ok 202; lần 1 DONE (lần 1 nộp wa trước khi lần 2 vào hàng: DONE); lần 2 DONE is_final=True |
| TC-PE06-27 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE06-28 | P | Go (chỉ các test còn tồn tại; tên TC cũ không còn trong repo): exam pass=4 skip=0 fail=0; thiếu test: TestSubmitSupersedes |
| TC-PE06-29 | P | khoá: ['compile_ok', 'created_at', 'id', 'is_final', 'language', 'samples', 'source', 'status']; 0 canary test ẩn; ẩn chỉ đếm? hidden-khoá: [] |
| TC-PE06-30 | P | CE: compile_log "main.cpp: In function 'int main()':\nmain.cpp:1:13: error: 'undefined_fn' was not declared " |
| TC-PE06-31 | P | Go: exam pass=3 skip=0 fail=0 |
| TC-PE06-32 | P | limit=2 → 2, mới nhất trước, cursor có; SV khác đọc lịch sử: 404/403 |
| TC-PE06-33 | P | Go: exam pass=2 skip=0 fail=0 |
| TC-PE06-34 | P | nộp bài thi → SUBMIT cuối (wa): verdict WA/SUBMIT/f |
| TC-PE06-35 | P | draft 200 rev=1; nộp bài thi → code_submissions: SUBMIT/t/AC |
| TC-PE06-36 | P | có SUBMIT + nháp khác: số auto/tổng = 0/1 (không tạo SUBMIT auto) |
| TC-PE06-37 | P | không nháp, không nộp: 0 code_submissions (earned=0 khi chấm: xem PE-08) |
| TC-PE06-38 | P | Go (chỉ các test còn tồn tại; tên TC cũ không còn trong repo): exam pass=3 skip=0 fail=0; thiếu test: TestFinalPicksLastSubmit |
| TC-PE06-39 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE06-40 | P | nháp nhận tới +9.5s, từ chối từ +10.5s (grace 10 s ep-r); GRADED/TIMEOUT; SUBMIT auto từ bản nháp cuối: SUBMIT/t/103 |
| TC-PE06-41 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE06-42 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE06-43 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE06-44 | P | 0 canary test ẩn/lời giải/giải thích ở mine/run/submissions (ở trạng thái đang làm) |
| TC-PE06-45 | P | [404, 404, 403, 403, 403] |
| TC-PE06-46 | P | Go: exam pass=1 skip=0 fail=0 |
| TC-PE06-47 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE07-01 | P | GET b8332d13aed3=attempt_id; TTL 2409 s ≈ còn lại 2409 s (±2 s) = deadline − now + grace |
| TC-PE07-02 | P | r5s02: nộp tay 200; khoá Redis gỡ sau 0.1s (≤10); chat-gate allowed=True |
| TC-PE07-03 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE07-04 | P | extend +2h → 200; TTL khoá lượt mới = 2410 s (deadline cố định started+40p; closes mới 2h): theo deadline_at |
| TC-PE07-05 | P | Go: exam pass=5 skip=0 fail=0 |
| TC-PE07-06 | P | đang làm {'allowed': False}; SV không làm bài {'allowed': True} |
| TC-PE07-07 | P | xoá khoá Redis → chat-gate vẫn allowed=False (tra DB); khoá nạp lại: True |
| TC-PE07-08 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE07-09 | S | dừng DB dùng chung ngoài cửa sổ khoá; nhánh Redis+DB chết do go TestLockerBothDownDeniesChat (xem TC-PE07-12) |
| TC-PE07-10 | S | đo lại khi máy rảnh: p95 phía gateway 302 ms / 777 ms khi load máy 45 (Go -race + Playwright + nhóm QC khác song song) — không đại diện; chưa có số hợp lệ |
| TC-PE07-11 | P | SV {'locked': True, 'until': '2026-11-15T14:00:41.791154367Z'}; ADMIN {'locked': False}; GV {'locked': False}; ẩn danh 401 |
| TC-PE07-12 | P | Go: exam pass=4 skip=0 fail=0 |
| TC-PE07-13 | P | 204; DB: TAB_TAKEOVER/{"reload": false} ; PASTE/{"chars": 812, "item_id": "01a1df64-05b8-7875-aab5-43113e9833fa"} ; TAB_HIDDEN/{} ; TAB_VISIBLE/{"duration_ms": 5000} |
| TC-PE07-14 | P | 51 sự kiện/yêu cầu → 204 []; meta quá lớn → 204 [] |
| TC-PE07-15 | P | sự kiện vào lượt của người khác: 404 |
| TC-PE07-16 | P | canary clipboard/ip trong exam_events: 0; log gateway+worker 0 (pg_dump có 1 dòng = mã nguồn tấn công x15 của chính QC, không phải sự kiện) |
| TC-PE07-17 | P | Go: exam pass=5 skip=0 fail=0 |
| TC-PE07-18 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE07-19 | P | [200, 403, 403, 403, 403] |
| TC-PE07-20 | P | GV thấy tóm tắt liêm chính: False; TA: False; khoá GV: ['adjusted', 'attempt_id', 'auto_score', 'flags', 'score', 'status', 'student', 'submit_reason', 'submitted_at'] |
| TC-PE07-21 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE07-22 | P | Go: exam pass=2 skip=0 fail=0 |
| TC-PE07-23 | P | cột "điểm trừ/gian lận/cờ" trong exam_*: không có |
| TC-PE07-24 | P | score*.go / quiz: tham chiếu exam_events/similarity_reports (không test): 0 |
| TC-PE07-25 | P | chữ buộc tội trong mã giao diện/Go (không test): không có |
| TC-PE07-26 | P | Go: exam pass=3 skip=0 fail=0 |
| TC-PE07-27 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE07-28 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE07-29 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE07-30 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE07-31 | P | A~B (đổi tên + định dạng) 1,000; A~A' (đổi thứ tự) 0,987; A~C (thuật toán khác) không nằm trong cặp hàng đầu (thấp); QC winnowing gốc 1,000/0,051/0,947 |
| TC-PE07-32 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE07-33 | P | Go: similarity pass=7 skip=0 fail=0 |
| TC-PE07-34 | P | GV chạy lại: 202 job SUCCEEDED; số cặp 18; run_id mới True |
| TC-PE07-35 | P | cặp gắn cờ: [('R5 SV 1', 'R5 SV 6'), ('R5 SV 3', 'R5 SV 8'), ('R5 SV 2', 'R5 SV 7'), ('R5 SV 9', 'R5 SV 10'), ('R5 SV 10', 'R5 SV 12'), ('R5 SV 9', 'R5 SV 12')] |
| TC-PE07-36 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE07-37 | P | go list -deps internal/exam/similarity: net/http = 0 |
| TC-PE07-38 | P | Go: exam pass=3 skip=0 fail=0; similarity pass=2 skip=0 fail=0 |
| TC-PE07-39 | P | chi tiết cặp 200 khoá ['a', 'b', 'pair'] |
| TC-PE07-40 | P | TA xem similarity 403; TA review 403 |
| TC-PE07-41 | P | Go: exam pass=2 skip=0 fail=0; today pass=1 skip=0 fail=0 |
| TC-PE07-42 | P | exam_events: id,course_id,exam_id,attempt_id,student_id,type,occurred_at,client_at,meta |
| TC-PE07-43 | P | Go: exam pass=2 skip=0 fail=0 |
| TC-PE07-44 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE07-45 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE07-46 | P | Go: exam pass=2 skip=0 fail=0 |
| TC-PE07-47 | P | chat thật (P-7#17: mã EXAM_IN_PROGRESS): tạo phiên 201; gửi tin trong giờ thi → 409 ['EXAM_IN_PROGRESS'] |
| TC-PE07-48 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-01 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-02 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-03 | P | Go: exam pass=3 skip=0 fail=0 |
| TC-PE08-04 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-05 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-06 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-07 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-08 | P | Go: exam pass=4 skip=0 fail=0 |
| TC-PE08-09 | P | 12 lượt: điểm hệ thống = điểm tính tay (Decimal; MCQ_SINGLE/MULTI PARTIAL, code theo trọng số 6/4/0, TF bỏ trống=0): lệch [] |
| TC-PE08-10 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-11 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-12 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-13 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-14 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-15 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-16 | P | Go: exam pass=2 skip=0 fail=0 |
| TC-PE08-17 | P | công bố PUBLISHED/13:21:53.759 (đóng 13:21:47; chấm xong tới 13:21:53.699); published_at đặt 1 lần |
| TC-PE08-18 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-19 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-20 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-21 | P | thông báo công bố: EXAM_PUBLISHED/12, EXAM_SCHEDULED/12 (SV có lượt = 12; SV vắng 2 không nhận) |
| TC-PE08-22 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-23 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-24 | P | Go: exam pass=4 skip=0 fail=0 |
| TC-PE08-25 | P | result SV: 200; 0 canary test ẩn/lời giải; khoá câu code: ['answer', 'compile_log', 'correct', 'earned', 'explanation', 'final_submission', 'hidden', 'item_id', 'max', 'mine', 'options', 'overridden', 'position', 'samples', 'stem', 'type'] |
| TC-PE08-26 | P | reveal_answers=true: khoá hiển thị câu MCQ ['answer', 'compile_log', 'correct', 'earned', 'explanation', 'final_submission', 'hidden', 'item_id', 'max', 'mine', 'options', 'overridden', 'position', 'samples', 'stem', 'type'] |
| TC-PE08-27 | P | test ẩn chỉ đếm: {"passed": 2, "total": 2} |
| TC-PE08-28 | P | reveal_answers=false: result 200; canary giải thích 0; khoá đáp án/đúng-sai: [('correct', 'true')]; điểm hiện 3.33 |
| TC-PE08-29 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-30 | P | Go: exam pass=4 skip=0 fail=0 |
| TC-PE08-31 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-32 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-33 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-34 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-35 | P | Go: exam pass=2 skip=0 fail=0 |
| TC-PE08-36 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-37 | P | Go: exam pass=4 skip=0 fail=0 |
| TC-PE08-38 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-39 | P | stats: mean 2.72 median 2.00 hardest r5-2 |
| TC-PE08-40 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-41 | P | Go: exam pass=3 skip=0 fail=0 |
| TC-PE08-42 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-43 | P | CSV 15 dòng (12 làm + 2 vắng + tiêu đề); đầu: '\ufeffmssv;ho_ten;trang_thai;diem_tu_dong;diem_chinh_thuc;nop_luc;ly_do_nop;cau_1;cau_2;cau_3;cau_4'; vắng: 'R5S0013;R5 SV 13;ABSENT;;;;;;;;' |
| TC-PE08-44 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-45 | P | Go: exam pass=4 skip=0 fail=0 |
| TC-PE08-46 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-47 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-48 | P | TA sửa điểm: 403 |
| TC-PE08-49 | P | thiếu lý do 422; có lý do 200; auto_score giữ 0.00 |
| TC-PE08-50 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-51 | P | override void: 200; r5s06 6.00 → 7.00 (+1 cho người bỏ/sai); r5s05 1.00 → 1.00 |
| TC-PE08-52 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-53 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-54 | P | regrade all: 202; sau chấm lại: [('01', '9.00'), ('06', '7.00'), ('02', '6.33'), ('07', '4.33')] (không đổi điểm đã chỉnh tay) |
| TC-PE08-55 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-56 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-57 | P | phúc khảo 201; lần hai 409 ['APPEAL_EXISTS'] |
| TC-PE08-58 | P | GV trả lời ADJUSTED 200; SV thấy 3.50 adjusted=True appeal=ADJUSTED; audit: exam.score_adjust/1 |
| TC-PE08-59 | P | thông báo phúc khảo gửi SV: 3 |
| TC-PE08-60 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-61 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-62 | P | Go: exam pass=1 skip=0 fail=0; contract pass=1 skip=0 fail=0 |
| TC-PE08-63 | P | results: TA/ADMIN/SV/ngoài lớp = [200, 403, 403, 403]; CSV GV/ADMIN [200, 403] |
| TC-PE08-64 | P | Go: exam pass=1 skip=0 fail=0; contract pass=1 skip=0 fail=0 |
| TC-PE08-65 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-66 | P | Go: exam pass=2 skip=0 fail=0 |
| TC-PE08-67 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE08-68 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE09-01 | LT | số liệu bank seed gốc (20 câu 12/4/4) đã bị sprint 10+ mở rộng (SHORT/ESSAY, câu GENERATED/EXTRACTED) |
| TC-PE09-02 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE09-03 | S | check-exam-seed bank: lệch do dữ liệu seed lớp 761987 bị nhóm QC khác/sprint sau đổi (5 câu AI_DRAFT đã REJECTED, thêm MCQ_MULTI) — cần DB sạch; không phải lỗi sản phẩm |
| TC-PE09-04 | P | grep "_test/" scripts/seed.mjs = 0 |
| TC-PE09-05 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE09-06 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE09-07 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE09-08 | P | check-exam-seed.mjs scores: matched 30/30 (so với seed/expected_exam_scores.csv) |
| TC-PE09-09 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE09-10 | P | check-exam-seed.mjs demo: OK (A/C/D, GV, TA, CE) |
| TC-PE09-11 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE09-12 | S | seed lần hai: 503 do gateway đang bị nhóm khác khởi động lại (cửa sổ khoá); số bảng không đổi 77/556/173/171; prod-guard đã PASS |
| TC-PE09-13 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE09-14 | P | APP_ENV=production node scripts/seed.mjs → rc=1, không request tới gateway |
| TC-PE09-15 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE09-16 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE09-17 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE09-18 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE09-19 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE09-20 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE09-21 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE09-22 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE09-23 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE09-24 | P | GATE PE trong handoff: 4 |
| TC-PE09-25 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE09-26 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE09-27 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE09-28 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE09-29 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PE09-30 | P | weekly_exam trong PROGRESS.md 2 dòng; report.md có; đủ 6 nợ (tách sandbox, precompiled header, checker tuỳ chỉnh, nối sổ điểm P6, chat khoá P3, retention RUN/exam_events) |
| TC-PE09-31 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PU06-01 | LT | cấu hình lhci đổi sau sprint 5 (9 URL, LCP off) |
| TC-PU06-02 | LT | cấu hình lhci đổi sau sprint 5 (9 URL, LCP off, TBT 200) |
| TC-PU06-03 | LT | lighthouserc.json nay: LCP "off", TBT ≤ 200 (không còn ≤170, không cpuSlowdown) — đổi bởi sprint sau (P-5#14 → 10.x) |
| TC-PU06-04 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PU06-05 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PU06-06 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PU06-07 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PU06-08 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PU06-09 | P | tokens.spec.ts khẳng định display:"block" (dòng 31) |
| TC-PU06-10 | P | HTML máy chủ có <h1> ở /, /chat, /threads, /inbox, /gradebook, /me; /exams, /questions, /today không có h1 SSR (ngoài 7 route của PU-06) |
| TC-PU06-11 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PU06-12 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PU06-13 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PU06-14 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PU06-15 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PU06-16 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PU06-17 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PU06-18 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |
| TC-PU06-19 | P | docs/sprints/5/handoff/dev-US-PU-06.md tồn tại |
| TC-PU06-20 | S | chưa chạy (mẫu): dạng tài liệu / UI chi tiết ngoài mẫu smoke 1 chính + 1 lỗi mỗi US |

## Trượt: bước tái hiện
### DEF-R5-01 — Nút đầu trang AppShell vi phạm axe `label-content-name-mismatch` (serious) ở mọi màn đăng nhập (TC-PE03-52, PE04-45, PE05-48, PE06-08)
Xem `defects-5.md`.
### DEF-R5-02 — Vùng cuộn ở màn làm bài code không focus được bằng bàn phím (TC-PE06-08)
Xem `defects-5.md`.

## Ca không đạt/khác ghi nhận không tính lỗi
- TC-PE02-37: fork bomb A8 nay ra `WA` (FORKED=0, fork bị chặn, 0 tiến trình mồ côi) thay vì `TLE` như gốc — bất biến an toàn thoả.
- TC-PE02-42: ptrace(TRACEME) và unshare(CLONE_NEWUSER) thành công (arm64, -no-seccomp) — như cảnh báo L1 gốc; chưa kiểm được seccomp bật (cần amd64/CI).
- TC-PE03-10: trần bộ nhớ nay 512 MiB (#5 sprint 5), TC gốc ghi 1024.
- go `TestRunningNotCancelled` đỏ 1 lần khi máy tải nặng, PASS 2/2 khi chạy riêng.
- Today: provider `staff`/`assignment` bị bỏ do quá hạn khi stack dùng chung chậm (log "today: bỏ Provider lỗi") — thiết kế degrade, ghi nhận.

## Ca K và lý do
K (a/b/c) = 0. Ca "chưa chạy (mẫu)" S: toàn bộ ca dạng tài liệu ngoài mẫu 1 chính + 1 lỗi / US; UI chi tiết (axe/INP/Lighthouse/ảnh), k6 (PE-02-46..49, PE-05-25, PE-09-15..19), `go test -race ./...` toàn bộ (…-39/50/55/60/47/48/68), TC-PE08 phần phát hành đôi (kill -9 hai bộ lập lịch), TC cần cửa sổ khoá global nếu chưa chạy (xem cuối tệp).

## Còn treo / chưa chạy hết (lý do môi trường, không phải a/b/c)
- Máy quá tải (load average 45) khi chạy song song Go `-race`, Playwright mock 5.5 và 7 nhóm QC khác → ca đo thời gian/UI không đại diện (TC-PE07-10 p95 gateway 302–777 ms; axe UI lượt 2 gián đoạn).
- `evidence-5/scripts/s02d_globals.py` (dừng judge / kill -9 worker / dừng Redis: TC-PE02-15/25/26/31/52/53, PE05-31, PE07-08/45) chờ khoá `/tmp/qc2-lock-global` 25 phút không được (nhóm khác giữ liên tục) → CHƯA chạy; chạy lại khi có khoá.
- TC-PE05-02/29 (`s11_b1.py`), PE07-03/44 (`s11_api.py` b4/b5): lỗi 503/422 do gateway bị nhóm khác khởi động lại — cần chạy lại; PE09-03/12/13 cần DB sạch (stack tạm) — chưa dựng.
- Go `-race exam/judge/quiz` + contract + TestSandboxAttacks (`/tmp/qc2-c5/gorace.log`, `gocontract.log`, `gosandbox.log`) chạy nền lúc dừng; `go vet ./...` rc=0.
- Sau khi tiến trình nền xong: `python3 evidence-5/scripts/make_report.py` (ghi đè tệp này từ results*.jsonl).
