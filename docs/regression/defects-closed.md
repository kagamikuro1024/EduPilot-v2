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

Chờ chạy lại: 9.5 DEF-01/02/12/16/17/19/20 (SmkF dừng, tôi chạy tiếp), R7-01 (CI shard — cần chạy CI, K(b) nếu PM không duyệt đẩy nhánh), R4-02 `text.xlsx` (chưa sửa), R10.8-03/04, UX08-50 (landing, ≥ P-11-?).
