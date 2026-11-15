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
