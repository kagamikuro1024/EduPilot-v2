## Lỗi đã được dev sửa và QC chạy lại đạt (bản dựng ≥ dbf48ed0, ep-r dựng lại 2026-11-16)

| Lỗi | Ca | Commit sửa | Kết quả chạy lại |
|---|---|---|---|
| DEF-R6-02 (High) roster PII không vô hiệu khi vào lớp | TC-P3-02-21 (sprint 6) | afbd433e | ĐẠT: vào bằng mã → `course.member_changed` +1, `ep:roster` xoá ≤ 3 s; duyệt 1,0 s; mời ra 0,5 s |
| DEF-R4-01 (TB) mã lớp phân biệt hoa/thường | TC-P208-02 (sprint 4) | 56ad200c (migration 00026) | ĐẠT: SK21D91 → 201, sk21d91 → 409 CONFLICT |
| DEF-R3-04 (Thấp) `DecryptKey(` ngoài llm/llmconfig | TC-P101-24 (sprint 3) | e859c554 | ĐẠT: grep = 0 ngoài `llmconfig/resolver.go` |
| DEF-R2-01 (TB) `redocly lint` openapi.yaml 1 lỗi | P0-02/TC-21 (sprint 1), TC-21 (sprint 2) | 816498c6 | ĐẠT: «valid», 0 lỗi |
| DEF-R9-01 (Thấp) tên test `-G` sprint 9 | TC-P7-01-G06, 02-G10, 03-G03, 03-G07, 06-G11, 08-G08 | P-11-42 (+4a5e49d7 TestSubmitIdempotent) | ĐẠT cả 6 (`evidence-9/logs/p11-42.log`) |
| DEF-R9.5-05 (TB) 9.5 sửa test có sẵn | TC-N10-10 (sprint 9.5) | P-11-36 hồi tố | ĐÓNG |
| DEF-R1.5-01 (TB) đã đăng nhập mở đường dẫn lạ → /login | TC-00-57, TC-00-59 (sprint 1.5) | 36c960f5 | ĐẠT: GV/SV/Admin → màn «Không tìm thấy trang», «Về Hôm nay» → «/» (`evidence-1.5/notfound-r1.log`) |
| DEF-R1.5-02 (Thấp) TA tràn ngang 375/390 px | TC-00-36 (sprint 1.5) | (topbar, qua 1c0700e2/P-11-39) | ĐẠT: 4 vai × 375/390: ox = 0 (`evidence-1.5/overflow-r1.log`) |
| DEF-R5-01 / R10.8-01 axe `label-content-name-mismatch` ở AppShell | TC-PE03-52 … (sprint 5); TC-AS08-25/AS09-36/AS09-39 (10.8) | 1c0700e2, ec1657c2 (P-11-39) | ĐẠT trên '/' (3 vai × 1440/375, cả tag +wcag21a): 0 lỗi (`evidence-10.8/axe-appshell.log`); 3 ca 10.8 do SmkH chạy lại đúng spec |
| DEF-R10.6-01 (Thấp) 409 trước 404 | TC-G03-24, TC-G05-24 (10.6) | 1c0700e2 | ĐẠT: 404 ASSIGNMENT_NOT_FOUND |
| DEF-R10.5-01 (Thấp) «Quyết định nhanh» trên bảng 7 ngày | TC-D07-02 (10.5) | 1c0700e2 | ĐẠT: y(7 ngày)=405 < y(Quyết định nhanh)=903 |
| DEF-R2-02 / R10-01 (contract đỏ) | TC-PG06-10/11 (sprint 2), TC-12-14 (sprint 10) | de20f571 (+ do tải) | ĐẠT lúc tải thấp: `go test -tags testroutes ./internal/contract/...` ok 37 s |
| DEF-R4-02 (Thấp) text.xlsx nhận 200 | TC-P210-03 (sprint 4) | b6d20a67 | ĐẠT: 4 tệp giả/hỏng → 422 UNSUPPORTED_FILE |
| DEF-R9.5-01/16 (Thấp) refuseDemo, check-meta | TC-N01-17, N01-12 (9.5) | 6ec47f79 | ĐẠT: 5 biến thể BASE → 1 dòng rc 1 (kể cả [::1], host.docker.internal); demo_stack «unknown» → rc 1 (`evidence-9.5/rerun2.log`) |
| DEF-R9.5-12 (Cao) real_llm 429/--resume | TC-N08-16/17 (9.5) | 965ca248 | ĐẠT: Retry-After 120 → dừng sau 1 s, «CHƯA HOÀN TẤT — hết hạn mức…», lưu resume.json, rc 3 |
| DEF-R9.5-02 (Thấp) canary-scan dọn nhà cung cấp | TC-N01-18 (9.5) | 6ec47f79 | MỘT PHẦN: 7 dòng = 0, CN1 xoá; CN0 (tạo 201 vì fake nhận mọi khoá) vẫn còn |
| DEF-R9.5-19 (Thấp) SEED_DEFAULT_PASSWORD | TC-N02-09/11 (9.5) | 6ec47f79 | MỘT PHẦN: chuỗi trắng bị từ chối (1 dòng); unset / rỗng vẫn rơi về mật khẩu trong `.env.local`/`.env.example` và chạy tiếp — chờ PM: thiết kế fallback hay lỗi |
| DEF-R8 TC-P6-06-11 (HIGH) PUT điểm chuỗi trắng xoá ô | TC-P6-06-11 (sprint 8) | 78e3d92d | ĐẠT: 11 chuỗi sai (gồm ' ', tab) → 422, DB giữ 9.00 v2 |
| DEF-R8 TC-P6-07-11 điểm âm XLSX | TC-P6-07-11 | cc1703f9 | ĐẠT: SCORE_OUT_OF_RANGE |
| DEF-R8-s8core-01 giải trình «không dự → 0 theo quy chế» | TC-P6-01-20, TC-P6-02-25 | d3ac89fc | ĐẠT: /me/grades text «Kiểm tra hằng tuần: 0 · không dự → 0 theo quy chế» + quote điều 1.4 |
| DEF-R8-s8core-02 float64 trong seedtool | TC-P6-02-15 | 1dea359f | ĐẠT: seedtool hết float64, go test PASS |
| TC-P6-09-49 target_letter ngoài thang | TC-P6-09-49 | 3eb6afe3 (P-11-55) | ĐẠT: 'Z', 'a+' → 422 |
| /me/grades final.locked_at 0001-01-01 | TC-P6-09-04 | 7b62e715 | ĐẠT: locked_at = 2026-11-15T19:01:38Z |
| TC-P6-06-56 (TC sai, P-11-54) | TC-P6-06-56 a/b/c | — | ĐẠT: (a) version chữ 422; (b) thiếu/null trên ô có điểm → 409, DB không đổi; (c) ô mới → 200 |
| TC-P6-09-40 | TC-P6-09-40 | — | ĐẠT: crm-risk-run --now theo buổi neo (P-11-48) → AT_RISK 1, WATCH 3, 30/30 khớp expected_risk.csv; lệch 7/30 ở đồng hồ thật = trôi lịch (đã kiểm) |

Chờ chạy lại: 9.5 DEF-01/02/12/16/17/19/20 (SmkF dừng, tôi chạy tiếp), R7-01 (CI shard — cần chạy CI, K(b) nếu PM không duyệt đẩy nhánh), R4-02 `text.xlsx` (chưa sửa), R10.8-03/04, UX08-50 (landing, ≥ P-11-?).

## Đợt chạy lại 2026-11-16 (sau merge ≥ 80e90d1b)
- **DEF-R3-02** (d6de360f): ĐÓNG. `node --test scripts/tokens-sync.test.mjs` PASS (mọi khai báo của kit có y nguyên ở tokens.css). TC-PU01-01 vẫn K (lỗi thời, đã ghi).
- **DEF-R9.1-02** (25cd3352, P-11-57): ĐÓNG. Quét `frontend/src/**/*.css`: 0 luật `:hover` ngoài `@media (hover: hover)` → TC-UX04-3 P (smoke-9.1).
- **DEF-R9.1-03 / UX07-8** : ĐÓNG. grep «Kiểm duyệt Threads» trong `frontend/src` = 0 → P.
- **TC-P6-03 (sprint 8)**: chạy lại với stub UTILITY: 01/02/03/07/08/10/13/25/32/38 PASS (03-10 lần đầu F do kịch bản đọc nhầm khoá citations; 03-13 lần đầu F do stub lớp 2 trả unclear_items có khoá `source` → BAD_RESPONSE đúng thiết kế schema; đã sửa stub).
- **DEF-R3-05**: nguyên nhân của số 15/42 s và 12 yêu cầu (kỳ vọng 3) đã tìm ra ở phía QC: `evidence-3/scripts/lib.py` hàm `call()` TỰ THỬ LẠI 502/503 tới 5 lần cách 8 s. llm_audit của lần đo cũ: 4 dòng GRADING cách nhau ~9 s, MỖI dòng `attempts=3`, `queue_wait_ms` 0–1; rồi 2 dòng `circuit_open` (attempts=0, 2–6 ms). Tức mỗi lời gọi = 3 lần thử đúng như TestBatchOutageEndToEnd; nhiễu do kịch bản QC. Lượt sạch (dùng `_call` không thử lại, `p10x3.py`) đang chờ cửa sổ không đo tải.
- **DEF-R6-C-01** (a081590a): mã đã có `THREAD_LOCK_TEXT` «Đăng bài tạm khóa trong lúc bạn làm bài thi.»; chạy UI thật (`evidence-3/scripts/r6c01.mjs`) chưa xong do lệnh dừng Playwright của PM.
- **DEF-R5-02** (d1374b09): chưa chạy lại (cần Playwright).
- **DEF-R3-05** (RERUN 2026-11-16, ep-r ≥ 32b4cb2b): ĐÓNG — TC-P102-14/P103-19/P103-20 PASS bằng llm_audit; nguyên nhân là kịch bản QC (lib.py call() thử lại 502/503 ×5), không phải sản phẩm. Evidence: `evidence-3/logs-p10x3c.log`, `evidence-3/scripts/p10x3.py`.
- **DEF-R6-C-01** (RERUN, a081590a): ĐÓNG — TC-P3-06-42 PASS trên UI thật (r6c01.mjs: có «Đăng bài tạm khóa…», không còn «Chat tạm khóa»).
- **DEF-R5-02** (RERUN, d1374b09): ĐÓNG — TC-PE06-08 PASS (axe 1440 /exams/<id>/take sau Chạy thử = 0).

## Đợt chạy lại 2026-11-18 (ep-r dựng lại ở 634d59c5, gồm dc67bd93 + 247bf4e4 + 7391aa9e)
- **DEF-R8-A1** (6c75bd54): ĐÓNG — TC-P6-03-36 PASS (EXTRACTION_INVALID, 1 lời gọi; s8r-03e.py).
- **DEF-R8-A2** (dev2): ĐÓNG — TC-P6-04-03 PASS (focus về nút Trang sau Esc), TC-P6-07-47 PASS (focus về «Thêm hành động»).
- **DEF-R8-A3** (TC-P6-03-19): MỞ, đã giao dev (trọng số ngoài [0;100] bị đổi thầm thành 0, mất PARAM_INVALID).
- **DEF-R3-05**: ĐÓNG (lỗi kịch bản QC). **DEF-R3-SMK-01/P103-08**: RÚT LẠI (lỗi kịch bản QC).

## Cập nhật 2026-11-19 (qc2)
- DEF-R8-A4 (TC-P6-05-37): ĐÓNG — sha 22387b06 (có trong 777bb5e2), chạy lại trên ep-r 1409b04d: câu «Cô tính điểm môn này thế nào?» → câu mẫu, 0 LLM; 05-38/39 P (s8r-chat05.py).
- DEF-R8-A5 (TC-P6-05-42): ĐÓNG — sha 2479feb9, chạy lại 2026-11-19 trên ep-r đã rebuild: P («Tài liệu nguồn đã gỡ» hiện đúng).
- DEF-R3-SMK-03 (TC-PU05-40, TC-GATEPU-15): ĐÓNG — sha 07a6b69c, chạy lại 2026-11-19: 7/7 route SV sạch, cả hai TC P.

## Đợt chạy lại DefR (qc2, 2026-11-21; HEAD qc2 9f05e1b2, ep-r dựng ở 6201b665 + fix). Bảng: evidence-defrerun/results-DefR.tsv
- **DEF-R2-04** (562f5a45): ĐÓNG — `go test -count=1 ./internal/contract` KHÔNG tag ok 56,3 s; có `-tags testroutes` ok (TC-PG06-10/11 và TC sprint 1/2 tương ứng).
- **DEF-R9.5-16** (6ec47f79): ĐÓNG — TC-N01-12 P (demo_stack=unknown → check-meta rc=1).
- **DEF-R9.5-22** (239df7fc): ĐÓNG — TC-N01-03 P (lệnh README chạy nguyên văn rc=0 «đã tồn tại»).
- **DEF-R10.8-02** (b8c43df8): ĐÓNG — TC-AS13-05, TC-AS14-11 P (34/34 tên test SRS §11 có trong mã; 7 test PASS).
- **DEF-R9.5-20** (98d4373d): ĐÓNG — TC-N06-04 P (Redis tắt trên ep-r: 503 OVERLOADED 3,7 s; cũ 39,5 s; TestSendRedisDownFailsFast xanh).
- **DEF-R5-03** (46bf5626), **DEF-R5-04** (4cd9c07e): ĐÓNG — TC-PE03-49, TC-PE03-51 P (ui lượt smkP_q.cjs 08:56).
- DEF-R4-04 · TC-P206-28 · 3121bda8 · P: EXPLAIN ListUsersAdmin trên DB tạm 60.000 user (schema ep-r có migration 00029): trang đầu / lọc role / lọc status / con trỏ đều Index Scan users_created_idx, 0,04–0,07 ms (SmkT4N, evidence-4/logs/s4n-p20628-explain.txt)
- DEF-R4-08 · TC-GATEP2-09 (phần --if-empty) · 69b06337+3b82e1cd · P: node scripts/seed.mjs --if-empty trên ep-r 475 lớp in «Đã có dữ liệu, không seed.» sau ~1 s, số dòng users/courses/enrollments/documents không đổi (SmkT4N, evidence-4/logs/s4n-seedifempty.txt); thời gian seed ≤ 360 s (P-11-76) do qc2 đo
- DEF-R4-03 · TC-P202-46 · P-11-72 (sửa TC, bỏ COOKIE_DOMAIN) · P: 4 biến TTL sai rc=1 nêu tên biến (SmkT4N, evidence-4/logs/s4n-p20246.txt)
- DEF-R4-06 · TC-P202-47 · P-11-73 (sửa TC, phạm vi AC15) · P: grep JWT_EXPIRATION phạm vi sản phẩm = 0, go test ./internal/contract ok
- DEF-R4-07 · TC-P201-19 · P-11-74 (sửa TC, dung sai +0,5 s +0,1 s) · P: 1,49–1,50 / 5,5 / 30,5 s nằm trong dung sai
- **DEF-R9.1-06** (6d127b88): ĐÓNG — TC-UX12-2 P (ep-r: Muộn / Vắng phép / Vắng giữ đỏ rgb(206,22,40) khi rê chuột, trước sửa xám; 375 cảm ứng luật rê không áp; evidence-9.1/logs/qc-uiux91-ru1-real.rerun7z2.log).

## Đợt chạy lại S4X (qc2 pair 1, 2026-11-21; ep-r @c12be038). Bảng: evidence-defrerun/results-S4X.tsv
- **DEF-R4-10** (d1f278f4): ĐÓNG — TC-P208-34 P (GV đã mở sẵn trang, bấm thông báo phân công trong 60 s → bộ chọn lớp đúng lớp mới; smkt3-bell 3/3).
- **DEF-R4-05** (f5f8a8be): ĐÓNG — TC-P201-11, TC-GATEP2-44, TC-GATEP2-53 P (log Caddy 0 dòng chứa token mời / token đặt lại / mã tham gia, 0 Next-Url/Next-Router-State-Tree).
- **DEF-R5-10** (dd160463) phần API: TC-PE03-33 P (tests_stale=[True], GV 200 tests_version 7→8, TA 403, QUESTION_IN_USE 409); phần giao diện chờ chạy lại sau rebuild.
- **DEF-R5-11** (3f748ec2): ĐÓNG — TC-PE08-04 P (nộp MCQ-only 200 trong 0,02 s, GRADED; chủ lượt nhận SSE `exam.attempt` status GRADED).
- **DEF-R7-02** (e4b82a05): ĐÓNG — TC-P4-02-14 P (số về 0 → ticket mới: dòng gộp chưa đọc + 1 khung SSE; evidence-defrerun/R7-02/p402-14.log).
- **DEF-R5-07** (d7fecdc2): ĐÓNG — TC-PE07-18 P (dán rồi đóng ngay: rời trang và đóng tab thật PASTE 0→1, không lộ nội dung; browserContext.close() không phát pagehide là hạn chế Playwright).
- **DEF-R5-08** (473472ec): ĐÓNG — TC-PE08-31 P. **DEF-R5-09** (247bf4e4): ĐÓNG — TC-PE08-38 P (bài trống dùng bài OPEN mới; bài cố định cũ đã đóng nên hiện «Vắng», đúng US). **DEF-R5-10** (dd160463): ĐÓNG — TC-PE03-33 P cả API (tests_stale) lẫn Drawer «Chấm theo bộ test cũ».
- **DEF-R3-01** (ffb46656): ĐÓNG — TC-P105-44 P (/settings/llm Admin 1440/390, GV 1440: 0 từ RAG/PII/trace/fallback/embedding/prompt).
- **DEF-R5-06** (0e52f1e5): ĐÓNG — TC-PE06-27 P (nhãn «Đã nhận, đang xếp hàng» 0,2 s → KẾT QUẢ 7,7 s khi 30 SV khác xếp hàng; r5s34, ep-r @aba738ef).
- DEF-R9.5-17 (TC-N02-02/03, 6ec47f79): ĐÓNG — RERUN 22/11 @859a90db trên qc2s: seed lần 2 rc=0, đếm bảng nghiệp vụ không đổi, không trùng; sửa tay 6 điểm rồi seed 2 lượt rc=0, không 422 IDEMPOTENCY_KEY_REUSED (evidence-8/logs/winD-*; evidence-defrerun/results-S4X.tsv). DEF-R9.5-26 (seed trên ep-r >100 lớp) vẫn mở, riêng.

## Đợt chạy lại DefR — S10X (qc2, 2026-11-21; ep-r aba738ef)
- **DEF-R10.7-01..04**: ĐÓNG — audit 10.7 pha `rr-defrerun` chạy lại đủ (data×4 vai, rỗng ×4 gồm teacher @1440/@375 với lớp QCEMPTYr107, đang tải, lỗi): 310 lượt chụp, lỗi chụp 0; H3=0, E7=0, P5=0 (01), H7=0, X3=0 (02, 03), T2=0 (04). Lệch còn lại 4 (H6×2, E5×2, chỉ student/practice/history) KHÔNG thuộc bốn DEF này: mục mới, chưa có DEF. Lỗi lượt trước (empty-teacher 2 failed) do dữ liệu: danh sách lớp của teacher (196 lớp) chỉ nạp ~100 lớp mới nhất nên không thấy QCEMPTYr107; đã đặt enrollment lớp này lên đầu, không phải lỗi sản phẩm. Bằng chứng: evidence-10.7/audit-rr-defrerun.md, evidence-defrerun/R10.7/all.log, results-S10X.tsv.

## Đợt 22/11 (qc2)
- DEF-R4-09 (9754c523, gochecknoglobals extract.go): ĐÓNG — golangci-lint run rc=0, go vet rc=0, ./internal/today -race ok (lượt 2), ./internal/integration ok (TC-P211-44 P; evidence-4/logs/lint-gate.log).
- DEF-Q2-S8-03 (674f9779, seed khi gateway tắt): ĐÓNG — TC-P6-SEED-35 P: lỗi «Gateway <API_URL> chưa sẵn sàng (ECONNREFUSED)…» rc=1 sau 1 s; bật lại rc=0 (evidence-8/logs/seed35b-*.log).
- DEF-R105S-02 (af07726d, run.py đường «thật»): ĐÓNG — TC-D08-14 P: LLM_DECISION_ALLOW_REAL=true + --real, trần 300; test_real.py 3 ca OK (chưa chạy lượt thật: chưa có phép PM).

- DEF-Q2-S4-01 (seed > 360 s): ĐÓNG 22/11 bởi f473c2ff; đo lại 281 s @6b3678c4 (evidence-2/fresh/seedt2/).
- DEF-Q2-DEV-01 (dev.mjs không truyền cổng mock với EP_PORT_OFFSET): ĐÓNG 22/11 bởi d9f244a5; bước 19/20 qua dev.mjs không lỗi (seedt2/dev.log).

- 22/11 (rerun F @118e1a40): ĐÓNG DEF-R2-04 (contract không tag ok 76,7 s: P0-02/TC-20, P0-03/TC-15, GATE-03/06, PG07-63 = P), DEF-R4-09 (lint 0 issues: P212-27 P), DEF-R9.5-22 (N01-03 P). TC-P209-51 P (UI + API 429 ở mã sai thứ 6). DEF-R9.5-14 còn 1 số lệch: nfr-real-llm.md «633,9 ms» ≠ load-b-ttft.json p(95)=633,99312 (phải 634,0) — gửi dev. evidence-f16/.
- DEF-R9.5-14 (4bfe444c): ĐÓNG 22/11 — TC-N09-04 P (139 số, 0 lệch; evidence-9.5/rerun/n09-04-4bfe444c.log).
- DEF-R9.5-06 (84e29eec): ĐÓNG 22/11 — TC-N10-06 P. DEF-R9.5-25: TC-N07-02 P theo P-11-101 (c17e5c28).
- DEF-R1-01 (752ce1f9): ĐÓNG 22/11 — P0-01/TC-07 P. DEF-R105S-01 (ca0aa7df + P-11-102): ĐÓNG — D08-06 P. DEF-R105S-06 (f15746a4): ĐÓNG — D06-22 P (ep-r FE @8f87223b).
