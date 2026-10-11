# Báo cáo cổng P3 (pha 2) — **ĐẠT CÓ ĐIỀU KIỆN**
HEAD `4418da3`. Stack dev `edupilot` (s6) dùng chung, không down (gate k6 tạo lại gateway rồi trả cấu hình thường). `LLM_PROVIDER=fake`. Log: `gate-p3-final-run.log`, `e1/e1-gate-final.{json,md}`.
**Điều kiện đóng:** CI ở HEAD đang đỏ (mục CI). Chưa phát hiện lỗi chức năng P3 nào còn mở.

## Số chính
| Chỉ số | Giá trị | Ngưỡng |
| --- | --- | --- |
| E1 (`eval_pii.py`, 200 mẫu: 96 TP / 4 FN / 0 FP / 100 TN) | recall **0,960**, false_block **0,000**, precision 1,000, F1 0,9796 | ≥ 0,95 / ≤ 0,05 — PASS (bộ mẫu do dev soạn: có thể lạc quan; xem report P3-07) |
| `first_event_ms` p95 (k6, 50 SV, trễ provider 5–15 s) | **85,1 ms**, 0/441 lỗi, 291 mẫu | < 300 ms — PASS |
| TTFT cache p95 / truy xuất p95 (trễ 300 ms) | **44,3 ms** / **342,6 ms**, 0/404 lỗi | < 1500 / < 4000 — PASS |
| `TestNoPayloadLeak` | 146 payload, 0 rò (lượt gate trước: 1.042 placeholder) | 0 — PASS |
| CI ở HEAD (run 38095815031) | **ĐỎ**: Go `TestICSTokenOnlySelf`, `TestCalendarUnionSources` (401 `TOKEN_INVALID` từ `fixture_test.go`; local PASS); Frontend `visual.spec` chat/threads @1440/@390 lệch 15.156 px (0,02) | xanh — **CHƯA ĐẠT**; job Judge xanh |

## Bảng lệnh (gate-P3.md)
| Mục | Kết quả |
| --- | --- |
| A1–A5, B1–B12 | PASS — chạy qua `GATE_K6=1 bash scripts/gate-p3.sh`: 14/14 PASS (vet 5 s, race 25 s, 4 test lẻ, contract 57 s, E1 215 s, Playwright 15 s, antipatterns, k6 106 s + 84 s), `GATE P3: PASS` rc=0 |
| B13, B14 | SKIP (phá cổng / gỡ k6 trên stack dùng chung) — không tính PASS |
| C3 | PASS (`TestChatThreadsContract` nằm trong contract) |
| C4 | PASS — SV gọi `…/posts/{id}/verify` → 403 |
| C5 | PASS — `chat=150 … threads=12/3`, trạng thái 4/3/2/1/2 |
| C1, C2 | C1 SKIP (không `dev:down` stack chung; dev báo seed DB trống 392 s, migration sạch), C2: `go test ./...` toàn cây vòng 1 đỏ `jobs`/`today` do tải, chạy riêng PASS; `pnpm lint` chưa chạy |
| C6 / F1 | quét PII đã làm ở P3-03/05/06; lượt này chỉ `TestNoPayloadLeak` + diff — chưa chạy lại `scan-pii-leak.sh` trên DB mới |
| D1 | PASS — `[^e]fetch(` ngoài `shared/`: 0 |
| D2 | PASS — `OFFSET` trong `store/queries`: 0 |
| D3 | PASS có ghi chú — `os.MkdirAll/WriteFile` chỉ ở `llm/fake/fake.go` (replay của provider giả, không phải đường sản phẩm) |
| D4 | PASS — `llm_audit`: CHAT/INTERACTIVE, CHAT/NEAR_REALTIME, EMBEDDING/INTERACTIVE+BATCH, QUESTION_GEN/BATCH |
| E1 | PASS (antipatterns rc=0) |
| E2–E8, E10 | không chạy lại; kết quả đã nghiệm ở P3-05/P3-06 (375 px, 3G, trạng thái, từ kỹ thuật) trên bản trước, Playwright gate lượt này PASS |
| E9 | PASS — Teacher có `AI_CONFIRM` (lớp 761987 "4 câu trả lời AI chờ xác nhận · 3 câu hỏi AI chưa trả lời được", lớp 761988 "3 …"), TA có `AI_CONFIRM` + `QUESTION_REVIEW`; SV `continue` 3 mục |
| F2 | PASS — TA/TCH/ADM vào chat sinh viên → 403, ADM vào threads → 403 |
| F3 | PASS — SV ngoài lớp: threads 403, chat 403 |
| F4 | PASS — SV B trên phiên của SV A (5 route) → 404 ×5 |
| F5 | PASS — 20 yêu cầu song song cùng `Idempotency-Key`: chat 200 và đúng 1 hàng USER; threads 201 và đúng 1 thread |
| F6 | PASS — 13 thread / 3 trang limit=5, 13 duy nhất, khớp DB; `limit=0/101` → 422 (threads, chat) |
| F7 | không chạy lại (khoá giờ thi đã PASS ở P3-05/06) |
| F8 | PASS một phần — `CANARY-7Q2X` = 0 trong `chat_messages`/`forum_posts`; câu hỏi đáp án không trả citation ANSWER_KEY (kết quả sự kiện `done` không đọc được bằng script, đối chiếu bằng `TestAnswerKeyNeverRetrieved` PASS) |
| F9 | PASS — author `{full_name, role, is_me}`; không email/user_id/MSSV |
| F10 | không chạy lại (3G / OVERLOADED: OVERLOADED không ép được với `fake`, nợ P3-05) |
| G1, G2 | PASS (Playwright private-chat, threads, privacy) |
| G3 | PM/dev cập nhật tick luồng ở `PROGRESS.md` |
| H (tay) | chủ dự án tự làm |

## Lỗi / nợ còn lại
1. **CI đỏ ở HEAD** (Cao với cổng): visual baseline chat/threads lệch (chạy Ubuntu; dev chụp baseline trên macOS? cần cập nhật baseline đúng quy trình) và `internal/calendar` 401 trong CI (local xanh; nghi hạn dùng token/giờ trong fixture) — dev sửa, QC chấm lại.
2. AC5 US-P3-08 (100 → 50 người dùng) cần BA quyết.
3. Nợ cũ: TC-31 (TEACHER 200 `/admin/llm/usage`, chờ BA), mask-timeout/OVERLOADED không ép được (TC-49/50 P3-02, TC-26 P3-05), E1 dataset do dev soạn.
Dọn dữ liệu QC: đã xoá 1 thread + 1 phiên chat QC tạo trong DB dev để số seed còn 12/150.
