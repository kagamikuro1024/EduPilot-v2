# DEV handoff — US-P3-05 (chat riêng) — **giao đủ (backend + UI)**
Nhánh `sprint/6-p3-p8`. Commit backend `US-P3-05: internal/chat …` (243ac9b) + commit UI `US-P3-05: giao diện /chat thật …`.

## Đã làm
**Backend (`internal/chat`, `internal/httpapi/chathttp`)**
- `Service`: phiên (liệt kê theo con trỏ, tạo, xoá mềm / hoàn tác, phạm vi tài liệu), `Send` (thứ tự SRS 4.7.1: phiên → lớp lưu trữ → kiểm nội dung → **khoá giờ thi trước mọi thứ** → tốc độ → phát lại theo `client_msg_id` → `CHAT_BUSY` → một giao dịch USER + ASSISTANT → `G`), `Retry` (cùng hàng, đặt lại mọi cột, `attempt`+1), `Cancel` (PUBLISH; không ai nghe thì ghi `CANCELLED` thẳng), `Feedback`, `Messages`, `Reap` + `ReapTask` (worker, 30 s).
- `G` tách khỏi request (`WithoutCancel`, hạn `CHAT_STREAM_MAX_SECONDS`, danh tính / trace / phiên che gắn rõ), mọi sự kiện đi qua Redis Stream `ep:chat:buf:{mid}:{attempt}`; người xem (kết nối đầu và nối lại) cùng đọc luồng đó → không khe. Bộ đệm mất → `snapshot` từ DB + thăm dò 500 ms. Khoá `CHAT_BUSY` nhả **trước** khi phát sự kiện cuối. Đường suy giảm tự dựng câu trích (không dùng chữ của `llm`, không hứa "giảng viên sẽ xem"). `partial_content` mỗi `FlushEvery` khi có đổi; mọi lệnh ghi cuối có điều kiện `STREAMING` + đúng `attempt`.
- `exam.Locker.RecordChatBlocked` (dedupe 1/lượt/phút), `llm.Response.MaskedCurrent` (chỉ tin hiện tại + kết quả tool), `llm.Chunk.Degraded`, `llm.EmbedRequest.System` (mẫu câu cá nhân của bộ phân loại, không có dữ liệu người dùng), config `CHAT_MAX_INPUT_CHARS` / `CHAT_RATE_PER_MIN` / `CHAT_STREAM_MAX_SECONDS`, mã lỗi `EXAM_IN_PROGRESS`, `CHAT_BUSY`, `MESSAGE_TOO_LONG`, `CHAT_UNAVAILABLE`, `MESSAGE_NOT_RETRYABLE`.
- 10 route + schema trong `openapi.yaml` (3 route SSE gắn **ngoài** nhóm nghiệp vụ trong `newRouterWith`, không `RequireIdempotencyKey`); contract scenario `scenarios_chat_test.go`, op 131 → 141.

**Giao diện (`frontend/src/features/chat`, `shared/data/chatStream.ts`)**
- `ChatScreen`: phiên đăng nhập thật có lớp → `RealChat`; phiên mô phỏng → `DemoChat` (bản mô phỏng cũ, giữ để các màn demo còn dùng `CHAT_DRAFT_KEY`).
- `chatStream.ts` (fetch + ReadableStream, token chỉ trong header, làm mới phiên khi `TOKEN_EXPIRED`); `useChatRun` (gửi / thử lại / nối lại; token gom theo khung hình bằng rAF; `off` chống lặp; rớt mạng tự nối lại bằng `Last-Event-ID`; lỗi trước khi mở luồng ném lại để **giữ chữ** trong ô soạn; gửi lại cùng chữ = cùng `Idempotency-Key`); `RealChat` (danh sách phiên + xoá mềm có Hoàn tác, trạng thái tải / rỗng / lỗi, Dừng, "AI đang bận. Thử lại sau khoảng N giây." + `Thử lại`, "Câu trả lời bị gián đoạn.", dòng "Đã ẩn n thông tin cá nhân trước khi gửi cho AI" + `Tìm hiểu`, khối kết quả tool, trích nguồn mở tại chỗ + `Xem tài liệu`, Hữu ích / Không hữu ích lạc quan + Hoàn tác, nháp tự lưu 2 s theo phiên, khoá giờ thi từ `GET /me/exam-lock` mỗi 30 s và khi nhận 409, tự mở lại tới hạn). **Không có** nút / chữ "Nhờ giảng viên" ở bất kỳ trạng thái nào.
- Mã lỗi chat thêm vào `ApiError.ts` / `i18n/vi.ts`.

## File đổi
`backend-go/internal/chat/*`, `internal/httpapi/{chathttp,chatwire.go,router.go,routes.go,apierr}`, `internal/exam/chatblocked.go`, `internal/llm/{mask.go,types.go,stream.go,gateway.go}`, `internal/privacy/classify.go`, `internal/agent/{agent.go,intent.go}`, `internal/platform/config`, `cmd/worker/tasks.go`, `api/openapi.yaml`, `internal/store/queries/{chat,ingest}.sql` + sinh, `internal/contract/*`, `internal/integration/no_payload_leak_test.go`; `frontend/src/features/chat/*`, `frontend/src/shared/data/{chatStream.ts,index.ts,ApiError.ts}`, `shared/i18n/vi.ts`, `frontend/e2e/{private-chat,privacy}.spec.ts`, `e2e/support/{chat-fixtures,session}.ts`, `e2e/ui-foundation.spec.ts` (ca `/chat` viết lại theo chat thật).

## Lệnh QC
```bash
cd backend-go && export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true
go test -count=1 -race ./internal/chat -v                       # 40+ test (EP_SKIP_TIMING=1 khi chạy song song; bỏ khi chạy riêng để kiểm ngưỡng 300 ms / 1 s)
go test -count=1 ./internal/exam -run TestRecordChatBlocked
go test -count=1 -race -tags testroutes ./internal/contract ./internal/integration -run 'Contract|Spec|NoPayloadLeak'
cd ../frontend && pnpm build:gate && E2E_PORT=3410 E2E_API_PORT=3412 npx playwright test e2e/private-chat.spec.ts e2e/privacy.spec.ts e2e/ui-foundation.spec.ts -g "chat|placeholder|/chat:" --workers=2
bash scripts/ui-antipatterns.sh   # rc=0
```

## Test đã chạy
Go: `internal/chat` xanh (gồm `TestChatSSENotBuffered` qua router thật, `TestChatContentOnlyInMessages`, `TestChatMatrix`, `TestTrustedContextFromJWT`, `TestBodyIdentityFieldsRejected`, 20 song song cùng khoá → 1 cặp hàng), `TestNoPayloadLeak` mở rộng qua chat thật (146 payload, 1.042 placeholder, 0 rò, 0 placeholder trên khung SSE), contract xanh. Frontend: `tsc`, `eslint .`, `ui-antipatterns` rc=0, `pnpm build` + `build:gate` xanh; Playwright `private-chat` 8/8 (desktop; ca 375 gộp trong cùng tệp), `privacy` 2/2, `ui-foundation /chat` 1/1; `shell|panels|class-join|safe-next|lcp|a11y` xanh.

## AC tự đánh giá
AC1 ✓ (test Go; k6 `first_event` làm ở US-P3-08) · AC2 ✓ · AC3 ✓ · AC4 ✓ (Go; e2e `reload mid-stream` dùng gateway giả — 3G chậm thật là việc của QC trên stack) · AC5 ✓ · AC6 ✓ · AC7 ✓ · AC8 ✓ · AC9 ✓ (`RecordChatBlocked` có test riêng ở `internal/exam`) · AC10 ✓ (e2e `exam lock`; đồng hồ giả: dùng `until` cách 40 phút, chưa chạy "tới hạn tự mở" bằng đồng hồ giả) · AC11 ✓ · AC12 ✓ · AC13 ✓ · AC14 ◐ (khối tool vẽ tổng quát: nhãn + giá trị theo khoá `Facts`; khoá cụ thể chốt khi P5 / P6 / P8-03 nối nguồn) · AC15 ◐ (trích `[n]` + mở tại chỗ + `Xem tài liệu`; "Nguồn đã gỡ" cần `/library` ở US-P8-02) · AC16 ✓ · AC17 ◐ (phiên / xoá / hoàn tác ✓; nút "Hỏi AI về tài liệu này" ở `/library/{id}` thuộc US-P8-02, phía API `document_id` đã có) · AC18 ✓ · AC19 ✓ · AC20 ◐ (cột ≤ 840 px, một Panel, nháp, 375 px không tràn, vùng chạm ≥ 44 px ✓; `AUDIT_SRC` 1440/1024/390 chưa chạy) · AC21 ✓.

## Nợ / cần hỏi
1. **Chưa chạy trên stack compose thật** (stack của chủ đang chạy; chưa xin dừng): e2e dùng gateway giả theo hợp đồng thật. Các phép đo trên stack (k6 `first_event`, 3G, `AUDIT_SRC`, đồng hồ giả của khoá giờ thi) để QC / US-P3-08.
2. `visual.spec.ts` đỏ 14 ảnh (cả màn không liên quan: home, threads, inbox, gradebook, settings-llm…) → ảnh mốc lệch theo máy, không phải do chat; ảnh `chat@1440/390` cần dựng lại khi chủ chốt máy chuẩn.
3. Đề xuất **D3** (`chat_sessions.title` vs AC9) chờ PM.
4. `DemoChat` (có `AskTeacher` cũ) còn cho phiên mô phỏng; gỡ cùng các màn mock khi hết mock.
5. Mỗi người một khoá hạn mức theo phút dùng cố định cửa sổ phút (SRS 4.1) — đã khớp; `ponytail:` mỗi `G` giữ một kết nối pub/sub Redis cho kênh huỷ (đủ T1; gom kênh khi đo thấy thiếu kết nối).
6. `internal/today` `TestQuestionReviewProvider` / `…Invalidates` đỏ khi chạy `./...` song song, xanh khi chạy riêng (đã biết, ngoài phạm vi).

## Vòng sửa 1 (PM, theo `qc/report-US-P3-05.md`)
- **BUG-1 (Cao) — COURSE_QA lỗi `PROVIDER_ERROR`.** Gốc: `internal/rag` truyền `document_ids` là `[]uuid.UUID` (kể cả rỗng) trong khi pool runtime chạy `QueryExecModeExec` (PgBouncer) — pgx không mã hoá được kiểu đó ("unable to encode []uuid.UUID{} … OID 0"). Sửa ở gốc: `RagSearch` nhận `text[]` rồi ép `::uuid[]` ở SQL, `rag.search` đổi `[]uuid.UUID` → `[]string`. Vì sao test cũ không bắt được: mọi test dùng pool mặc định của pgx (đoán kiểu qua Describe) và contract chỉ khẳng định "có sự kiện cuối" (lỗi cũng là sự kiện cuối). Test mới: `testutil.RuntimePool` (pool đúng cấu hình runtime); `TestSearchWorksOnRuntimePool` (`internal/rag`, nil / rỗng / một / tài liệu khác — **đỏ trước khi sửa, xanh sau**, đã chạy cả hai chiều); `TestNoPayloadLeak` (qua chat COURSE_QA thật, Postgres) giờ dùng `RuntimePool` và khẳng định mỗi lượt đi tới `stage:"generating"` + `done`. Nợ còn lại: các test khác vẫn dùng pool mặc định — nên đổi dần sang `RuntimePool` ở gói có SQL mảng.
- **BUG-2 — luật EXAM_SCHEDULE quá rộng.** Nguyên nhân thật: `phòng thi` / `ngày thi` / `giờ thi` đứng một mình (từ fix B1 của P3-04) khớp "…được mang vào **phòng thi**?". Giờ `(ngày|giờ|phòng) thi` chỉ tính khi đi kèm từ hỏi (`khi nào, bao giờ, ở đâu, phòng nào, mấy giờ…`) hoặc sau "cho hỏi / xem"; dạng đảo `thi … khi nào` giữ nguyên. `routeCases` thêm 4 câu COURSE_QA ("Quy chế thi cuối kỳ nói gì về tài liệu được mang vào phòng thi?", bản rút gọn, "Thi lại được mấy lần theo quy chế", "Trong phòng thi có được dùng máy tính không"); các câu EXAM_SCHEDULE cũ (gồm "phòng thi cuối kỳ ở đâu", "cho hỏi ngày thi giữa kỳ") vẫn đúng.
