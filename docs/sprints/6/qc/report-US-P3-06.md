# QC report — US-P3-06 (Threads `/threads` + tường lửa PII)  · Kết luận: PASS (vòng 3, 2026-10-11; BUG-1b đã sửa)

Handoff: `docs/sprints/6/handoff/dev-US-P3-06.md`. Bộ TC: `tc-US-P3-06.md` (49 TC).
**Môi trường:** như P3-05 (gateway + worker build từ HEAD `af83a54`, DB riêng `qc_p801`, Redis db 9, docling thật, frontend `next build` + `next start :3410` trỏ gateway QC; không đụng stack s55). Nhà cung cấp `fake` (xem hạn chế embedding ở report P3-05): bài AI chỉ có khi nội dung thread trùng một đoạn tài liệu. Bài thi dựng bằng API (`EXAM_MIN_LEAD_SECONDS=5`).

## Cổng đã chạy
| Lệnh | Kết quả |
| --- | --- |
| `go test -count=1 -race ./internal/thread ./internal/chat` | PASS — 94 test |
| `go test -race -tags testroutes ./internal/contract` | PASS |
| `TestAllWritePathsUseFirewall` | PASS |
| `go test -count=1 -race ./cmd/worker -run TestRosterInvalidateNotBlockedByLongJob` | **FAIL** — panic `nil pointer dereference` trong `pgxpool.(*Pool).Acquire` ← `store.IngestClaim` ← `ingest.(*Processor).Ingest` ← `ingest.(*Queue).handle` (Processor dựng với `Pool=nil` trong test). Handoff ghi test này "nay có" và đóng nợ AC3 của P3-02 nhưng nó không chạy được |
| `bash scripts/ui-antipatterns.sh` | PASS (rc=0) |

## TC
| TC | Kết quả | Chứng cứ |
| --- | --- | --- |
| 01, 49 | PASS | `{items,next_cursor}`, mỗi hàng có `title`, `preview`, `tags`, `week_no`, `author`, `answer_state`, `reply_count`, `last_activity_at`; 19 thread qua 4 trang `limit=5`: 19 hàng, 19 khác nhau, bằng DB; `limit=500` → `422` |
| 02 | PASS (một phần) | `week=3`, `tag=quy-che`, `q=canh bao hoc vu` (không dấu) đều trả đúng 1; `state=pending` 0 (không có bài chờ) |
| 03 | KHÔNG KIỂM ĐƯỢC | chưa dựng thread bị ẩn |
| 04 | PASS | `precheck` sạch → `allowed:true`; số hàng `forum_threads/forum_posts/pii_events` không đổi |
| 05 | PASS | `reasons` EMAIL, MSSV, PHONE; `redacted_text/title` thay `[đã ẩn]` |
| 06 | PASS | rào 60/phút: 429 ở yêu cầu thứ 61 trong phút (tính cả 18 lần trước đó) |
| 07, 08 | PASS | UI 375 px: sau khi dừng gõ hiện một dòng "Phát hiện…", không mở hộp thoại |
| 09 | PASS | đăng bài có MSSV + email → `422 PII_DETECTED` và hộp thoại "Bài này có thông tin cá nhân": "…Chúng tôi tìm thấy: 1 email, 1 mssv." đúng hai lối `Chuyển sang chat riêng`, `Ẩn thông tin rồi đăng` (cộng nút `Đóng`); nút cao 44 px. Chữ `mssv` viết thường trong câu (Thấp) |
| 10 | PASS | 0 hàng thread mới; `pii_events` chỉ có `BLOCKED`/`SWITCHED` + loại, 0 cột chữ |
| 11 | PASS | `Chuyển sang chat riêng` → `/chat?session=…`, nháp chuyển nguyên văn `MSSV 20229001 và email … được mấy điểm?` (nội dung; tiêu đề không vào nháp chat), phiên mới 0 tin |
| 12 | KHÔNG KIỂM ĐƯỢC | cần lượt thi + chuyển nháp cùng lúc |
| 13, 14 | PASS | `redact:true` → `201`, lưu bản đã ẩn; DB không còn chuỗi `sv.kha@…`/`20229001`; `pii_events` `EMAIL/MSSV REDACTED` |
| 15, 16 | PASS | gọi thẳng API, thiếu `redact` → `422`; PII ở tiêu đề → `422`; email ở nội dung → `422` |
| 17, 18 | PASS | `"Em được mấy điểm giữa kỳ?"` và không dấu `"em duoc may diem chuyen can?"` → `422`, `personal_question:true`, lý do `PERSONAL_QUESTION`; `redact:true` vẫn `422` (không có lối ẩn) |
| 19 | PASS | thread khớp tài liệu → `ai_state=ANSWERED`, đúng 1 bài `AI` `PENDING`, 1 trích dẫn, 1 lời gọi `CHAT` làn `NEAR_REALTIME`; SV không thấy `confidence` (không có trường) |
| 20 | KHÔNG KIỂM ĐƯỢC | chưa giao lại outbox |
| 21, 22 | PASS (một phần) | ngoài ngữ cảnh / điểm thấp → `SKIPPED LOW_SCORE`, 0 bài AI; chưa kiểm `LLM_UNAVAILABLE`, `me/today` |
| 23 | PASS | SV: 0 trường `confidence`/`ai_body`; Staff thấy `confidence` (1,000: do fake) |
| 24 | PASS | `verify` `200` → `VERIFIED`, audit `thread.post.verify`; thông báo `THREAD_ANSWERED`, `THREAD_VERIFIED` tạo (có `dedupe_key` riêng) |
| 25 | PASS | `correct` với `version` → `CORRECTED`, `ai_body` còn, SV thấy bản sửa, `ai_body` không lộ; `version` cũ → `409 VERSION_CONFLICT` |
| 26 | PASS | `reject` → SV: thread vẫn đọc được, bài AI **không** hiện; `answer_state` null; tìm kiếm không ra; "thread tương tự" không có |
| 27 | PASS | `verify` lần hai `200` |
| 28 | PASS | `verify` bài `REJECTED` → `409 POST_STATE_CONFLICT`; `version` cũ → `409 VERSION_CONFLICT` |
| 29 | PASS (một phần) | `THREAD_ANSWERED` 1, `THREAD_VERIFIED` 2 (verify, correct); thông báo `THREAD_REPLY` chưa kiểm |
| 30 | KHÔNG KIỂM ĐƯỢC | embedding giả không cho thread tương tự |
| 31 | PASS | SV và ADMIN gọi `verify/correct/reject` → `403`; ADMIN list → `403` |
| 32 | PASS | người ngoài lớp (sv31) list / precheck → `403` |
| 33 | PASS | `author.full_name = "Vũ Hoàng Giang"` hiện với `sv.kha` (Q3 công khai) |
| 34 | KHÔNG KIỂM ĐƯỢC | chưa tắt mạng |
| 35 | PASS | 20 yêu cầu song song cùng khoá: tất cả `201`, cùng một `id`, 1 hàng |
| 36, 37 | KHÔNG KIỂM ĐƯỢC | 1440/1024 px, trạng thái rỗng/lỗi chưa chạy |
| 38 | PASS (một phần) | `/threads` 375 px: không cuộn ngang; mọi điều khiển ≥ 44 px trừ link ẩn "Bỏ qua điều hướng"; `Đặt câu hỏi`, `Bộ lọc` hiện; `/threads` mở composer tại chỗ (không modal) |
| 39 | PASS | quét AST |
| 40 | PASS | trong lượt thi: `POST /threads` → `409 EXAM_IN_PROGRESS` + `until`, 0 hàng mới, 1 `CHAT_BLOCKED`; lớp khác (người này không thuộc) `403` |
| 41 | PASS | khi khoá: list `200`, đọc thread `200`, precheck `200`, bình luận `201` |
| 42 | PASS (API) | nộp bài → đăng thread được sau **9 ms**; giao diện khoá chưa chụp |
| 43 | KHÔNG KIỂM ĐƯỢC | không dừng Redis |
| 44 | PASS | 12 mẫu tấn công (tên có dấu / không dấu / đảo / HOA, `mssv …`, `20229001`, email, 2 dạng SĐT, 2 dạng CCCD, tên người khác): **0 lọt** qua `precheck` |
| 45 | PASS | 6 mẫu âm (AES, "điểm chuyên cần được tính thế nào?", `8080`/CVE, `Hoa Kỳ`, `minh chứng`, `An toàn thông tin`): **0 chặn nhầm** |
| 46 | PASS | thread tiêm lời nhắc đòi `get_my_grade_summary`/`get_my_attendance`: `SKIPPED`, 0 bài AI; payload provider không có tên tool cá nhân |
| 47 | PASS (một phần) | payload bài AI không chứa PII; `TestNoPayloadLeak` (P3-03) |
| 48 | PASS | thread có đúng văn bản đoạn `ANSWER_KEY` (canary) → 0 bài AI, canary không có trong payload |

## AC
AC1–AC17, AC19, AC20 PASS ở phần kiểm được; AC18 PASS một phần (bố cục, lọc ở dưới 1100 px chưa chụp); AC3 của P3-02 (`TestRosterInvalidateNotBlockedByLongJob`) **FAIL**.

## Lỗi
- **BUG-1 (Trung bình)** — `thread.created` không có handler: mỗi thread tạo ra một outbox `thread.created` nhưng worker báo `topic "thread.created" chưa đăng ký handler` và đẩy vào dead-letter sau 4 lần (`outbox dead-letter … attempts 4`). Tái hiện: `POST /courses/{id}/threads` rồi xem `/tmp/qc801/wk.log`. Nghi: chưa gọi `RegisterKind`/handler cho `thread.created` ở `cmd/worker/registry.go`. AI trả lời vẫn chạy (đi qua `jobs`+`ep:ingest`) nên hệ quả hiện thời là nhiễu dead-letter, nhưng mọi tiêu thụ sau này (P4) sẽ mất sự kiện.
- **BUG-2 (Trung bình, test)** — `TestRosterInvalidateNotBlockedByLongJob` panic nil pointer (xem cổng), nên AC3 của P3-02 vẫn chưa được chứng minh bằng test chạy được.
- **BUG-3 (Thấp)** — hộp thoại PII ghi "1 email, 1 mssv" (loại viết thường, thiếu viết hoa `MSSV`).
- Ghi chú: `GET /me/today` (TC-22) và thread tương tự (TC-30) cần embedding thật để đo.

## Đề nghị
FAIL vì BUG-1/2 (dev sửa trong vòng sửa 1). Phần PII (44, 45, 15–18, 13–14, 40–41) PASS đầy đủ. QC chạy lại sau fix: TC-20, 22, 29 (REPLY), 34, 36, 37, 42 (giao diện), 43.


## Chấm lại sau fix `23c1587` (2026-10-11)
Stack QC dựng lại (DB mới, seed, docling, tài liệu `Quy chế đào tạo` + `ANSWER_KEY` canary, frontend build lại từ HEAD). Nhắc: nhà cung cấp `fake` trong **gateway** và trong **worker** là hai tiến trình độc lập, `POST /_test/llm/fake` chỉ chỉnh gateway.

| Lỗi | Kết quả | Chứng cứ |
| --- | --- | --- |
| BUG-1 (`thread.created` chưa có handler) | **PASS cho `thread.created`** | 5 thread tạo → 5 hàng `outbox thread.created`, 0 `dead_at`, `attempts=0`, không còn dòng `chưa đăng ký handler` cho topic này |
| **BUG-1b (mới, Trung bình)** | **FAIL** | topic `thread.post_decided` (phát khi Staff `verify` / `correct` / `reject`) cũng chưa có handler: worker log `topic "thread.post_decided" chưa đăng ký handler` mỗi 1–2 s, hàng `outbox` `attempts=3` đang tới ngưỡng dead-letter. Tái hiện: tạo thread có bài AI → `POST /posts/{pid}/verify` → xem log worker. Thông báo `THREAD_VERIFIED` vẫn tạo (không qua topic) nên người dùng không thấy lỗi, nhưng sự kiện sẽ mất khi P4 cần |
| BUG-2 (`TestRosterInvalidateNotBlockedByLongJob` panic) | **PASS** | `go test -count=1 -race ./cmd/worker -run 'TestRosterInvalidateNotBlockedByLongJob|TestRosterInvalidateRegistered'` xanh (0,45 s / 0,70 s); `internal/thread`, `internal/chat`, `cmd/worker`, `internal/privacy`: 158 test xanh; contract 19 xanh |
| BUG-3 (nhãn MSSV) | **PASS** | UI 375 px (bản dựng từ HEAD): "…Chúng tôi tìm thấy: 1 email, 1 MSSV." |

TC AI trả lời thread:
| TC | Kết quả | Chứng cứ |
| --- | --- | --- |
| 19 | PASS | thread khớp tài liệu → `ANSWERED`, đúng 1 bài `AI` `PENDING` (Staff thấy `confidence 1,000`, 1 trích dẫn), 1 lời gọi `CHAT` làn `NEAR_REALTIME`, nhúng ≤ 1 lần |
| 20 | PASS (một phần) | bình luận thêm của SV khác → `201`, vẫn đúng 1 bài AI, 1 bài `HUMAN`; giao lại `outbox` chưa ép được (cột `processed_at` không tồn tại, QC không sửa DB) |
| 21 | KHÔNG KIỂM ĐƯỢC | `error_rate=1` đặt ở gateway không áp dụng cho worker (thread vẫn `ANSWERED`); cần công tắc cho worker (Q-QC) |
| 22 | PASS (một phần) | `GET /me/today` (Teacher) `200`, nhưng chưa có thread `SKIPPED` nào để thấy mục `AI_CONFIRM` (US-P3-08) |
| 23 | PASS | SV không có trường `confidence`; nhãn "Chờ xác nhận" (`PENDING`) |
| 29 | PASS | `THREAD_ANSWERED` (1 mỗi thread, `dedupe_key` riêng), `THREAD_REPLY` 1 khi có bình luận, `THREAD_VERIFIED` 1 sau hai lần `verify` (không nhân đôi) |
| 30 | KHÔNG KIỂM ĐƯỢC | embedding giả không có độ tương đồng gần; "thread tương tự" luôn rỗng; bài bị loại không xuất hiện (đúng) |

**Kết luận:** FAIL còn BUG-1b. Đề nghị dev đăng ký handler (hoặc bỏ phát sự kiện) cho `thread.post_decided`; sau đó chấm lại TC-24–28 chỉ cần kiểm log worker không còn dead-letter.


## Chấm lại vòng 3 sau fix `78e8a11` (2026-10-11)
| Lỗi | Kết quả | Chứng cứ |
| --- | --- | --- |
| BUG-1b (`thread.post_decided` chưa có handler) | **PASS (mức test)** | `go test -count=1 ./cmd/worker -run TestEveryEmittedTopicHasHandler` → PASS (0,22 s): test quét mọi topic outbox được phát ra và buộc có handler đăng ký (chặn cả loại lỗi này ở các topic sau, ví dụ `llm.budget.warn` dev thêm); `go test -race ./cmd/worker ./internal/thread` 69 test xanh. Chưa lặp lại ca `verify` trên stack runtime (stack QC đã dọn); nếu PM muốn xác nhận runtime, QC dựng lại ở lượt P3-08 |
Các TC khác của P3-06 không đổi. **Kết luận:** PASS; TC-21, 30 (cần embedding thật / công tắc worker) và 34, 36, 37, 43 vẫn chưa kiểm được, chuyển P3-08 / gate.
