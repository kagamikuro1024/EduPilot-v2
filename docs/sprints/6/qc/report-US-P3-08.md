# QC report — US-P3-08 (seed, "Hôm nay", k6, `gate-p3.sh`, `gate-p8.sh`)  · Kết luận: FAIL (3 lỗi: seed, k6, gate P8)

Handoff: `docs/sprints/6/handoff/dev-US-P3-08.md` (HEAD `fd0ff79`). Bộ TC: `tc-US-P3-08.md` (29 TC).
**Môi trường:** stack `s55`/dev đã bị tắt trong lúc chạy (`edupilot-postgres-1`… biến mất), nên QC dựng **stack riêng** bằng container QC (`qc-pg` 15432, `qc-redis` 16379, `qc-minio` 19000, `qc-mailpit`, `qc-docling`, `qc-judge`) + gateway/worker `-tags testroutes` build từ HEAD (cổng 18080) — không đụng `edupilot-test-*`. `LLM_PROVIDER=fake`. Playwright của cổng dùng `pnpm build:gate` của dev. Log gate: `gate-p3-run.log`, `gate-p8-run.log`.
**Hạn chế môi trường (không phải lỗi dev):** judge Docker của QC thiếu `cc1plus` (`g++: fatal error: cannot execute 'cc1plus'`) nên bước 10 của `seed.mjs` (bài thi code) không chạy được; QC chạy bản sao `scripts/.seed-qc.mjs` bỏ bước 10 (không commit; nhóm `INSERT` thi mẫu thuộc P-E).

## Kết quả cổng
| Cổng | Kết quả | Chứng cứ |
| --- | --- | --- |
| `bash scripts/gate-p3.sh` | **PASS** (`GATE P3: PASS`, rc=0) | `go vet`, `go test -race privacy+agent+chat+thread`, `TestChatMatrix`, `TestThreadsMatrix`, `TestUnmaskStream`, `TestNoPayloadLeak` (146 payload, 1.042 placeholder, 0 rò), `TestThreadsAgentHasNoPersonalTools`, `TestStudentNeverSeesConfidence`, `go test ./internal/contract/...`, **E1** `eval_pii.py` (215 s, PASS), Playwright privacy + private-chat + threads (17 s), `ui-antipatterns.sh` — 12/12 PASS (chưa bật `GATE_K6`) |
| `GATE_DOCLING=1 bash scripts/gate-p8.sh` | **FAIL** (`GATE P8: FAIL`, rc=1) | 7 bước PASS (vet, test rag+library+calendar+document+ingest, `TestAnswerKeyNeverRetrieved`, contract, feed ICS `BEGIN:VCALENDAR`, Playwright documents+library+calendar, antipatterns) rồi **FAIL** ở `check-docs-seed.mjs` |
| k6 `first_event` | **FAIL** (rc=99) | `first_event_ms` p95 = **113,5 ms** (< 300 ✓) nhưng `http_req_failed` = **99,82 %** (124.357/124.573; ngưỡng < 1 %) |
| k6 `ttft` | PASS (rc=0) | `ttft_cache_ms` p95 20,5 ms (< 1500), `ttft_rag_ms` p95 324 ms (< 4000), 0 % lỗi |
| `go test ./...` | PASS khi chạy riêng | chạy toàn bộ song song: `internal/jobs` và `internal/today` đỏ do tải (dev đã khai); chạy riêng từng gói: `ok` |

## TC
| TC | Kết quả | Chứng cứ |
| --- | --- | --- |
| 01 | **FAIL** | seed trên DB trống **dừng** ở bước 13: `Seed lỗi: POST …/threads → 422 PII_DETECTED` (xem BUG-1); `check-chat-seed.mjs` sau khi vá bằng bản sao QC: `chat=153 … threads=11/3 states={"SKIPPED":11}` với 7 dòng ✗ (câu ngoài tài liệu 11 thay vì 10; thread lớp 1 11 thay vì 12; trạng thái PENDING 0/4, VERIFIED 0/3, CORRECTED 0/2, REJECTED 0/1, SKIPPED 11/2) |
| 02 | PASS (một phần) | chạy seed lần hai (bản sao QC): số `chat_sessions / chat_messages / forum_threads / forum_posts / documents / calendar_events` trước và sau **giống hệt** (154 / 14.722 / 14 / 0 / 5 / 3; số tin lớn do k6 `first_event` đã chạy) |
| 03 | KHÔNG KIỂM ĐƯỢC | thiếu mốc seed sprint 5 trên máy này |
| 04 | **FAIL** | không có bài AI nào ở trạng thái `PENDING/VERIFIED/CORRECTED/REJECTED`: cả 11 thread `SKIPPED LOW_SCORE` (xem BUG-2) |
| 05, 06 | KHÔNG KIỂM ĐƯỢC | phụ thuộc TC-04; quét seed không PII chưa chạy |
| 07 | PASS | `GET /me/today` của `sv.gioi`: `continue` 3 mục `{kind:"CHAT", id, title, href:"/chat?session=…", course:{id, class_code}, at}`, mới nhất trước; 4 phiên có tin trong 7 ngày |
| 08 | PASS | `sv.moi` (chưa chat) → `continue: []` |
| 09 | PASS | mọi `continue[].id` thuộc phiên của chính `sv.gioi`; `sv.kha` nhận phiên khác, không giao nhau |
| 10 | PASS (sát ngưỡng) | phiên mới xuất hiện trong `continue` sau **60 s** (đúng TTL) |
| 11 | PASS | Teacher: `AI_CONFIRM` "11 câu hỏi AI chưa trả lời được · lớp 761987", `href /threads?state=pending&course=…`, lý do "Thread cũ nhất đã chờ 6 phút." |
| 12 | KHÔNG KIỂM ĐƯỢC | không có bài `PENDING` để xử lý (BUG-2) |
| 13 | PASS | SV và ADMIN không có `AI_CONFIRM`; TA có `AI_CONFIRM` + `QUESTION_REVIEW` |
| 14 | **FAIL** | xem k6 |
| 15 | PASS | xem k6 `ttft` (mọi truy xuất ở đây trúng cache sau khi ấm; "rag" 324 ms gần trễ 300 ms của provider giả) |
| 16 | KHÔNG KIỂM ĐƯỢC | chưa ép vượt ngưỡng |
| 17 | PASS | gate-p3 chạy đúng thứ tự, bảng + `GATE P3: PASS` |
| 18 | KHÔNG KIỂM ĐƯỢC | chưa phá cố ý `TestNoPayloadLeak` |
| 19 | **FAIL** | khi bật hai bước k6 thì `first_event` FAIL (rc=99) nên gate sẽ `FAIL` (không chạy được `GATE_K6=1` trọn vẹn cho tới khi sửa) |
| 20 | PASS | Playwright privacy + private-chat + threads PASS trong cổng |
| 21 | PASS (đối chiếu) | 5 kịch bản F3 + F4 đã nghiệm tay ở report P3-05 / P3-06 |
| 22, 23 | PASS (một phần) | `TestChatThreadsContract` nằm trong `go test ./internal/contract/...` PASS; golden SV không có khoá cấm (quét ở P3-07 TC-06) |
| 24 | PASS (đọc cổng) | `TestChatMatrix`, `TestThreadsMatrix` (ma trận vai) PASS trong cổng |
| 25, 26 | KHÔNG KIỂM ĐƯỢC | chưa thử gỡ k6 / dừng stack |
| 27 | PASS (một phần) | migration chạy sạch (version 11); `go test ./...` PASS khi chạy từng gói, đỏ `jobs`/`today` khi song song toàn cây (nhiễu tải đã biết) |
| 28 | PASS | `fetch(` ngoài `shared/` chỉ là `refetch(`; OFFSET: không thấy trong code đọc |
| 29 | PASS | `ui-antipatterns.sh` rc=0; 375 px các màn đã kiểm ở P3-05/06, P8-02/03 |

## AC
AC1 **FAIL**, AC2 **FAIL**, AC3 PASS, AC4 PASS (nhánh `SKIPPED`; nhánh `PENDING` chưa kiểm vì BUG-2), AC5 **FAIL** (k6 `first_event`), AC6 PASS (gate P3), AC7 PASS, AC8 PASS, AC9 một phần; AC16/AC17 của P8-03 (`gate-p8`, seed docs): **FAIL** (BUG-3).

## Lỗi
- **BUG-1 (Cao) — seed không chạy xong trên DB trống.** `node scripts/seed.mjs` dừng ở bước 13: `POST /courses/{c1}/threads → 422 PII_DETECTED`. Thread mẫu `"Điều kiện dự thi cuối kỳ theo quy chế"` có nội dung `"Em vắng 3 buổi thì còn được dự thi cuối kỳ không?"` bị tường lửa PII chặn là `PERSONAL_QUESTION` (đúng luật AC8 của P3-06). `seedThreads` chỉ chấp nhận 201 nên mất cả `pnpm dev` (`SEED_ON_EMPTY_DB`). Tái hiện: `precheck` đúng câu đó bằng token `sv.gioi` → `allowed:false, reasons:[PERSONAL_QUESTION]`. Sửa: đổi câu mẫu thành câu học thuật không chứa "em/của em". (Cũng nhiều thread khác trong seed có dạng "Em…" — kiểm lại toàn bộ `THREADS_1/THREADS_2`.)
- **BUG-2 (Cao) — với nhà cung cấp `fake` không thread mẫu nào có bài AI.** 11/11 thread `ai_state=SKIPPED LOW_SCORE` (vectơ nhúng giả không có độ tương đồng gần), nên không thể đạt AC2 (4 PENDING, 3 VERIFIED, 2 CORRECTED, 1 REJECTED, 2 SKIPPED) bằng đường "đúng luồng AI". Dev đã nêu nợ 2; đề xuất: seed dùng đoạn văn của tài liệu làm tiêu đề + nội dung thread (khớp vectơ giả) cho thread cần bài AI.
- **BUG-3 (Trung bình) — `check-docs-seed.mjs` FAIL xác định, làm đỏ `gate-p8.sh` với `GATE_DOCLING=1`.** Lỗi: `GET /courses/{c1}/calendar?from=…&to=… → 422` — script tính `to − from` = 62 ngày + vài ms (> 62 ngày = `RANGE_TOO_LARGE`). Tái hiện: `node scripts/check-docs-seed.mjs` ở `scripts/seed-check-lib.mjs:60` / `check-docs-seed.mjs:12`. Sửa: rút `to` còn ≤ 61 ngày. (Lượt `--no-rerun` chạy qua do khác vài ms.)
- **BUG-4 (Trung bình) — k6 `first_event` không thể đạt `http_req_failed < 1 %`.** Số luồng 100 > 30 sinh viên; mỗi sinh viên chỉ một lượt `STREAMING` nên các yêu cầu còn lại `409 CHAT_BUSY` (trả ngay) và bị tính lỗi (124.357/124.573). Độ trễ sự kiện đầu vẫn đạt (113,5 ms). Sửa kịch bản: ≤ 30 VU hoặc không đếm `409` là lỗi, hoặc dùng nhiều tài khoản. Lưu ý: `CHAT_RATE_PER_MIN` tối đa 600 (dev ghi stack TEST tăng giới hạn nhưng không nêu số).
- **Ghi chú:** tính toán seed "≤ 60 s" (TC-03) và phần "chat=150±5 / outside 10" chưa thể đối chiếu do BUG-1/2; `check-chat-seed.mjs` báo ✗ đúng các sai lệch đó (script nhận diện được).

## Đề nghị
FAIL tới khi dev sửa BUG-1, 2 (seed), 3 (check-docs-seed), 4 (k6). Phần "Hôm nay" (`continue[]`, `AI_CONFIRM`) và gate P3 (không k6) PASS. Sau khi sửa: QC chạy lại `seed.mjs` trên DB trống, `check-*-seed.mjs`, `gate-p3.sh` có `GATE_K6=1`, `gate-p8.sh` có `GATE_DOCLING=1`.
