# DEV handoff — US-P3-08 (seed, "Hôm nay", k6, cổng) + `#15` — **mã giao đủ; phần cần stack compose CHƯA chạy**
Nhánh `sprint/6-p3-p8`. Commit `US-P3-08: …`; luật intent `#15` ở commit `US-P8-03: #15 …`.

## Đã làm
**"Hôm nay" (Go, có test)**
- AC3 `continue[]`: `TodayStudentFeed` (một truy vấn gộp buổi học + ≤ 3 phiên chat RIÊNG của chính người xem, có ≥ 1 tin trong 7 ngày, chưa xoá, mới nhất trước; giữ ngân sách ≤ 5 truy vấn của `TestTodayQueryBudget`). Mục `{kind:"CHAT", id, title, href:"/chat?session=<id>", course:{id,class_code}, at}`; rỗng → `[]`. `openapi.yaml` ghi kiểu mục (maxItems 3). Giao diện: vùng "Tiếp tục học" ở `StudentToday` (không có trong DOM khi rỗng). Test `TestContinueChatSessions`, `TestContinueOnlyOwn`, `TestContinueEmptyHidden`; e2e `today.spec.ts -g 'continue learning'`.
- AC4 `AI_CONFIRM` (bậc 50, Giảng viên + TA): cột `ai_pending` / `ai_skipped` / `ai_oldest` gộp vào `TodayStaffPending` (không thêm truy vấn); title "{a} câu trả lời AI chờ xác nhận · {b} câu hỏi AI chưa trả lời được · lớp X" (chỉ số khác 0), href `/threads?state=pending&course=<id>`, lý do nêu thread cũ nhất. Worker nối `thread.created` / `thread.post_decided` với `today.Invalidator` (`thread.TopicPostDecided` thay literal). Test `TestAIConfirmProviderStaffOnly`, `TestAIConfirmInvalidatedOnDecision`. Bình luận của Staff và bài AI mới (việc nền) không có sự kiện riêng: mục đổi trong ≤ 60 s nhờ TTL (đúng AC).
- Hai pin cũ cập nhật vì thêm topic: `TestTodayInvalidatedByOutbox` 21 → 24 (calendar.changed, thread.created, thread.post_decided).

**`#15` (luật intent)** — `reWindow` ("tuần tới", "N ngày/tuần tới", "sắp tới") + `reHasGi` ("có gì", "có việc gì"…) và KHÔNG có từ nội dung môn (`reContent`: chương, bài giảng, tài liệu, thuật toán, quy chế, "là gì", "như thế nào"…) → `UPCOMING_EVENTS`; có từ nội dung vẫn `COURSE_QA` (hoặc `LIBRARY_SEARCH` nếu nhắc slide / tài liệu). Thêm `UpcomingDays(text)`: "30 ngày tới" → `days=30`, "2 tuần tới" → 14, còn lại 7 (tool vẫn kẹp 1–30; trước đây luôn 7). Ca test ở `routeCases` (+5 UPCOMING, +6 COURSE_QA) và `TestUpcomingDays`.

**Seed (qua API thật, idempotent — CHƯA chạy trên stack)**
- `scripts/seed.mjs` bước 11–13 (+ `scripts/chat-seed-data.mjs`): 11 = 5 tài liệu (3 PDF trong `seed/documents/` + 2 PDF tối giản sinh bằng script: `EXAM_PAPER`, `ANSWER_KEY` chứa `CANARY-7Q2X`; tải lần lượt qua presign → PUT → complete, chờ `READY`), chia sẻ bài giảng sang lớp 2 bằng `share-from`, 2 sự kiện lớp 1 (`Thi giữa kỳ` +14 ngày, `Thi cuối kỳ` +56 ngày) + 1 `OTHER` lớp 2; 12 = 150 câu chat (A 70, B 45, khác 25, 10 ngoài tài liệu; sv.gioi 4 phiên × 2 tin, 29 người × 5 tin = 153) bằng `POST /chat/sessions` + `POST …/messages` đọc SSE tới hết (đợi nếu 409 giờ thi); 13 = 12 thread lớp 1 (4 PENDING, 3 VERIFIED, 2 CORRECTED, 1 REJECTED, 2 SKIPPED theo `want`) + 3 thread lớp 2, chờ AI trả lời rồi Staff quyết định; chạy lại chỉ bổ sung phần thiếu (khoá tự nhiên = tiêu đề / số tin của phiên).
- `scripts/check-chat-seed.mjs` (`chat=150±5 … threads=12/3 idempotent=ok`; `threads` in bảng đếm; sinh viên thấy 12 thread, không có khoá cấm), `scripts/check-docs-seed.mjs` (`docs=… READY answer_key=1 shared=… reembedded=0 events=2/1 idempotent=ok`; `timing`: giây / trang), `scripts/seed-check-lib.mjs`. `idempotent=ok` = chạy lại `seed.mjs` rồi so số liệu (`--no-rerun` bỏ qua).

**k6** `benchmarks/load/chat.js`: `first_event` (100 luồng, 60 s, `first_event_ms` p95 < 300 = `timings.waiting` vì gateway ghi `event: status` ngay) và `ttft` (`ttft_cache_ms` p95 < 1500, `ttft_rag_ms` p95 < 4000; TTFT = first_event + (mốc ms của khung `token` đầu − mốc của khung `status` đầu), lấy từ `id:` của SSE vì k6 không thấy từng khung). Vượt ngưỡng → k6 thoát ≠ 0. `k6 inspect` đọc được cả hai kịch bản.

**Cổng** `scripts/gate-p3.sh`, `scripts/gate-p8.sh` (+ `gate-lib.sh`): đúng thứ tự AC6 / AC16, dừng ở lỗi đầu, bảng + dòng cuối. `strict_test` bắt buộc test chạy thật (Docker không tới → `FAIL TestNoPayloadLeak: thiếu stack`; test bị SKIP → FAIL). SKIP chỉ ở k6 (`GATE_K6=1` khi chưa cài k6 → `PASS (có SKIP)`) và `check-docs-seed.mjs` (cần `GATE_DOCLING=1`). Đã thử: Docker giả → `GATE P3: FAIL`, mã 1; test thật + SKIP → `GATE P3: PASS (có SKIP)`, mã 0.

**Hợp đồng** `TestChatThreadsContract` (AC8; trước đó chưa có): đúng 20 thao tác tag `chat` + `threads`, mọi status đã khai báo được gọi hoặc miễn trừ, 0 lỗi schema, 0 khoá cấm.

## Lệnh QC
```bash
cd backend-go && export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true
go test -count=1 -race ./internal/today ./internal/agent ./cmd/worker -v
go test -count=1 -tags testroutes ./internal/contract -run 'TestChatThreadsContract|Contract|Spec'
cd ../frontend && E2E_API_PORT=3412 pnpm build:gate && E2E_PORT=3410 E2E_API_PORT=3412 npx playwright test e2e/today.spec.ts --workers=2
cd .. && bash scripts/ui-antipatterns.sh
# cần stack seed (PM chưa cho dựng compose):
node scripts/seed.mjs && node scripts/check-chat-seed.mjs && node scripts/check-docs-seed.mjs
bash scripts/gate-p3.sh ; GATE_K6=1 K6_BASE=https://localhost:773 bash scripts/gate-p3.sh ; GATE_DOCLING=1 bash scripts/gate-p8.sh
```

## AC tự đánh giá
AC3 ✓ · AC4 ✓ · AC5 ✓ về mã (k6 chưa chạy) · AC6 ✓ về mã (đã thử nhánh FAIL / SKIP; chưa chạy trọn trên stack) · AC8 ✓ · AC9 ✓ (ma trận + `strict_test`) · **AC1, AC2, AC7 chưa kiểm** (cần seed chạy trên stack thật) · AC16/AC17 của P8-03 như trên.

## Nợ / cần hỏi
1. **Chưa dựng compose** (PM yêu cầu báo trước; stack s55 của chủ đang chạy). Cần một lượt có stack (gateway + worker `-tags testroutes` + `docling-serve` + Mailpit + provider `fake`) để: chạy `seed.mjs`, `check-*-seed.mjs`, `eval_pii.py` (sinh `benchmarks/reports/e1.{json,md}`), k6 hai kịch bản, `gate-p3.sh`, `gate-p8.sh`.
2. Rủi ro seed thread (chưa quan sát): trạng thái bài AI phụ thuộc điểm truy xuất của provider `fake`; nếu AI bỏ qua câu trong tài liệu hoặc trả lời câu ngoài tài liệu, seed in `CẢNH BÁO seed thread` (không dừng) và `check-chat-seed.mjs` báo số lệch — khi đó chỉnh `want` / câu hỏi trong `chat-seed-data.mjs`.
3. SRS ghi `FAKE_LLM_TTFT_MS=300`, mã chỉ có `FAKE_LLM_LATENCY` (trễ trước token đầu): `chat.js` dùng `FAKE_LLM_LATENCY=300-300` (`first_event`: `5000-15000`). Đề nghị BA sửa tên biến trong SRS.
4. `ttft` của `chat.js` giả định `id:` của khung SSE là `<lượt>:<ms>-<seq>` (Redis Stream) và gateway / k6 cùng đồng hồ máy chủ (stack local); chưa kiểm trên stack thật.
5. Thời gian seed: bước 11 (docling thật) ≈ 2,5–4 phút (đọc tuần tự để đo giây / trang), tính riêng khỏi mốc "≤ 60 s" của AC1 (chat + thread).
