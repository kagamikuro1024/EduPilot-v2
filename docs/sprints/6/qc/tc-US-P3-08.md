# QC test case — US-P3-08 (seed, "Hôm nay", k6, cổng `gate-p3.sh`)
Nguồn: `docs/specs/FEAT-private-chat-pii/US.md` US-P3-08 (AC1…AC9) + `SRS.md` 6 (20 thao tác), 8, 9.2, 9.4; `docs/sprints/6/plan.md` story 11 và "Nợ đem vào" (5.5 #2 "Tiếp tục học"); `docs/phases/P3.md` cổng nghiệm thu + "Bạn tự kiểm"; `.claude/skills/gate/SKILL.md`; `docs/FLOWS.md` F3, F4, F14; `QUESTIONS.md` Q21 (`AI_CONFIRM` đăng ký ở **P3**); `docs/sprints/6/proposals.md` #5 (`ACCEPTED`). Viết trước khi có code, **không đọc code của dev** (hộp đen).

**Biến shell** (SRS 9.1): `GW`; `C1` = lớp 1 (`761987`), `C2` = lớp 2 (`761988`); `Q=$GW/api/v1/courses/$C1`; `SVA` = `sv.gioi@edupilot.local`, `SVB` = `sv.kha@edupilot.local`, `TA_`, `TCH`, `ADM`; `j='curl -sk -H "Content-Type: application/json"'`; `PSQL`; `$PW = pnpm -C frontend exec playwright test`. Giao diện nghiệm thu tay bằng **`playwright-cli`** (toàn cục; **không** chạy `playwright-cli install`). Cổng: `bash scripts/gate-p3.sh`, chi tiết ở `docs/sprints/6/qc/gate-P3.md`.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P3-08-01 | AC1 | DB trống, `LLM_PROVIDER=fake` | `pnpm dev` rồi `node scripts/check-chat-seed.mjs` | In `chat=150±5 … threads=12/3 idempotent=ok`; seed đi qua **đúng luồng chat thật** (không `INSERT` thẳng); ≈ 150 câu hỏi chat riêng của sinh viên seed, lệch về hai chủ đề A, B + 10 câu ngoài tài liệu |
| TC-P3-08-02 | AC1 (idempotent) | tiếp TC-01 | `$PSQL "select (select count(*) from chat_sessions), (select count(*) from chat_messages), (select count(*) from forum_threads), (select count(*) from forum_posts)"` → chạy `time node scripts/seed.mjs` lần hai → đếm lại | Bốn số **không đổi** sau lần chạy thứ hai; không hàng nào thêm |
| TC-P3-08-03 | AC1 (thời gian) | máy QC, stack như TC-01 | `time node scripts/seed.mjs` (so với mốc trước P3 ghi ở sprint 5) | Tổng thời gian seed tăng **≤ 60 s** so với trước P3 |
| TC-P3-08-04 | AC2 (thread mẫu đủ trạng thái) | seed xong | `node scripts/check-chat-seed.mjs threads`; `$PSQL "select verification_state, count(*) from forum_posts where kind='AI' group by 1"`; `$PSQL "select ai_state, count(*) from forum_threads where course_id='$C1' group by 1"` | Lớp 1 có 12 thread: 4 bài AI `PENDING`, 3 `VERIFIED`, 2 `CORRECTED`, 1 `REJECTED`, 2 thread `ai_state='SKIPPED'`; lớp 2 có 3 thread |
| TC-P3-08-05 | AC2 (sinh viên không thấy `REJECTED`) | như trên | `j "$Q/threads" -H "$SVA" \| jq '[.items[].id] \| length'` | `11` (thread có bài `REJECTED` vẫn hiện nhưng **bài** biến mất; nếu seed làm cả thread ẩn thì số là 11 theo AC2) |
| TC-P3-08-06 | AC2 (không PII thô trong seed) | seed xong | `NAME='Vũ Hoàng Giang' NAME_NOACC='Vu Hoang Giang' MSSV=20229001 EMAIL=sv.gioi@edupilot.local PHONE=0912345678 CCCD=001099012345 bash docs/sprints/6/qc/scripts/scan-pii-leak.sh` | Mọi dòng `forum_threads`, `forum_posts`, `pii_events`, `outbox`, `jobs`, `audit_log`, `llm_audit`, `log compose` = **0**; thread mẫu không chứa PII thô (quét canary) |
| TC-P3-08-07 | AC3 (`continue[]` — nợ 5.5 #2) | `sv.gioi` có ≥ 4 phiên chat có tin trong 7 ngày | `j "$GW/api/v1/me/today" -H "$SVA" \| jq '.continue'` | Tối đa **3** mục, mới nhất trước, mỗi mục `{kind:"CHAT", id, title, href:"/chat?session=<id>", course:{id,class_code}, at}`; chỉ phiên **chưa xoá** có ≥ 1 tin trong 7 ngày |
| TC-P3-08-08 | AC3 (rỗng → ẩn hẳn) | sinh viên chưa chat bao giờ (ví dụ `sv.moi`) | `j "$GW/api/v1/me/today" -H "$SVMOI" \| jq '.continue'`; `playwright-cli goto $GW/` → `playwright-cli find "Tiếp tục học"` | `[]`; vùng "Tiếp tục học" **không có trong DOM** (0 kết quả) |
| TC-P3-08-09 | AC3 (phân quyền — không bao giờ phiên của người khác) | `sv.gioi` và `sv.kha` đều có phiên | `j "$GW/api/v1/me/today" -H "$SVB" \| jq '[.continue[].id]'`; đối chiếu `$PSQL "select id from chat_sessions where user_id='$UA'"` | Không id nào của `sv.gioi` xuất hiện trong `continue` của `sv.kha`; `go test ./internal/today -run 'TestContinueChatSessions\|TestContinueOnlyOwn\|TestContinueEmptyHidden' -v` → `ok` |
| TC-P3-08-10 | AC3 (độ trễ cache) | `sv.gioi` | Tạo phiên mới + gửi một tin; đếm ≤ 60 s rồi gọi lại `GET /me/today` | Phiên mới xuất hiện trong `continue` ≤ 60 s |
| TC-P3-08-11 | AC4 (`AI_CONFIRM` — nguồn việc Staff) | lớp 1 có 4 bài AI `PENDING` + 2 thread `SKIPPED` chưa có bình luận Staff | `j "$GW/api/v1/me/today" -H "$TCH" \| jq '.tasks[] \| select(.kind=="AI_CONFIRM")'`; `playwright-cli goto $GW/` (vai teacher) → `playwright-cli find "chờ xác nhận"` | **Một** mục `AI_CONFIRM` (bậc 50) với chữ "{a} câu trả lời AI chờ xác nhận · {b} câu hỏi AI chưa trả lời được" — chỉ nêu số **khác 0**; liên kết `/threads?state=pending`; lý do nêu thread **cũ nhất** |
| TC-P3-08-12 | AC4 (vô hiệu sau quyết định) | tiếp TC-11 | Xử lý hết (xác nhận / sửa / loại bài AI; Staff bình luận vào thread `SKIPPED`); đếm ≤ 60 s rồi gọi lại `GET /me/today` | Mục `AI_CONFIRM` biến mất ≤ 60 s; `go test ./internal/today -run 'TestAIConfirmProviderStaffOnly\|TestAIConfirmInvalidatedOnDecision' -v` → `ok` |
| TC-P3-08-13 | AC4 (phân quyền) | như trên | `j "$GW/api/v1/me/today" -H "$SVA" \| jq '[.tasks[].kind]'`; `j "$GW/api/v1/me/today" -H "$ADM" \| jq '[.tasks[].kind]'`; `j "$GW/api/v1/me/today" -H "$TA_" \| jq '[.tasks[].kind]'` | Sinh viên và Admin **không bao giờ** nhận `AI_CONFIRM`; TA của lớp có; giảng viên lớp khác không có |
| TC-P3-08-14 | AC5 (k6 `first_event`) | stack test đã seed, `k6` đã cài | `k6 run benchmarks/load/chat.js --env SCENARIO=first_event` (100 VU, 60 s, provider giả trễ 5–15 s) | `thresholds ✓` với `first_event_p95 < 300`; k6 thoát `0` |
| TC-P3-08-15 | AC5 (k6 `ttft`) | như trên, `FAKE_LLM_TTFT_MS=300` | `FAKE_LLM_TTFT_MS=300 k6 run benchmarks/load/chat.js --env SCENARIO=ttft` | `thresholds ✓`: `ttft_p95 < 1500` ms ở trường hợp **cache trúng** và `< 4000` ms khi **có truy xuất**; k6 thoát `0` |
| TC-P3-08-16 | AC5 (nhánh lỗi) | như trên | Hạ ngưỡng giả (sửa `--env` để ép vượt, ví dụ `FAKE_LLM_TTFT_MS=3000`) rồi chạy lại `SCENARIO=ttft`; `echo $?` | k6 thoát **khác 0** khi vượt ngưỡng (không âm thầm xanh) |
| TC-P3-08-17 | AC6 (cổng chạy đủ và đúng thứ tự) | repo + stack | `bash scripts/gate-p3.sh; echo $?` | In bảng PASS/FAIL theo đúng thứ tự: `go vet` → `go test -race ./internal/privacy/... ./internal/agent/... ./internal/chat/... ./internal/thread/...` → `TestUnmaskStream` → `TestNoPayloadLeak` → `TestThreadsAgentHasNoPersonalTools` → `TestStudentNeverSeesConfidence` → `go test ./internal/contract/...` → `python benchmarks/eval_pii.py --min-recall 0.95 --max-false-block 0.05` → Playwright `privacy.spec.ts private-chat.spec.ts threads.spec.ts` → `ui-antipatterns.sh`; dòng cuối `GATE P3: PASS`; `echo $?` → `0` |
| TC-P3-08-18 | AC6 (dừng ở lỗi đầu) | repo | Cố ý làm hỏng `TestNoPayloadLeak` (ví dụ đặt biến môi trường khiến che bị tắt, **không** sửa test); `bash scripts/gate-p3.sh; echo $?` | Dòng cuối `GATE P3: FAIL`, mã thoát ≠ 0; các bước **sau** bước hỏng không chạy |
| TC-P3-08-19 | AC6 (`GATE_K6=1`) | k6 đã cài, stack test seed | `GATE_K6=1 bash scripts/gate-p3.sh` | Thêm hai bước k6 (`first_event`, `ttft`) vào bảng; cả hai PASS thì dòng cuối `GATE P3: PASS` |
| TC-P3-08-20 | AC7 (F3 + F4 đi trọn) | stack seed | `pnpm -C frontend exec playwright test privacy.spec.ts private-chat.spec.ts threads.spec.ts` | Tất cả pass, **kể cả** bước ngắt mạng; phủ đủ 5 kịch bản "Bạn tự kiểm" của `P3.md`: đăng thread `"Em Nguyễn Văn A 20221234 được mấy điểm?"` bị chặn + chuyển kênh giữ nguyên chữ; gõ tên + MSSV trong chat → trả lời hiện tên thật, payload provider giả chỉ `[[SV_1]]`; hỏi điểm MSSV khác bị từ chối; 3G chậm + tải lại giữa chừng không mất; khoá giờ thi |
| TC-P3-08-21 | AC7 (nghiệm thu tay) | stack seed, `playwright-cli` | Chạy tay 5 kịch bản trên bằng `playwright-cli` (xem `tc-US-P3-05.md` TC-09, 22, 35, 38, 55 và `tc-US-P3-06.md` TC-09, 11, 40) | Kết quả giống spec E2E; ảnh chụp lưu ở `docs/sprints/6/qc/shots/` |
| TC-P3-08-22 | AC8 (OpenAPI) | repo | `grep -c "operationId" backend-go/api/openapi.yaml` trước/sau; `cd backend-go && go test ./internal/contract/... -run 'TestChatThreadsContract' -v` | **20** thao tác mới của SRS 6 đều có trong `openapi.yaml`; contract test `ok`, phản hồi khớp schema **kể cả 4xx** |
| TC-P3-08-23 | AC8 (golden của sinh viên) | như trên | Kiểm golden phản hồi sinh viên: `grep -cE '"(confidence\|retrieval_score\|groundedness\|ai_body\|hidden_reason\|verified_by)"' <golden của route sinh viên>` | `0` khoá cấm trong mọi golden của phép chiếu sinh viên |
| TC-P3-08-24 | AC9 (gate chạy với cả bốn vai) | stack seed | Xem bảng của `gate-p3.sh`: bước ma trận quyền | Bước ma trận quyền chạy với **cả bốn** tài khoản seed (Sinh viên, TA, Giảng viên, Admin), không chỉ một vai |
| TC-P3-08-25 | AC9 (nhánh thiếu công cụ) | gỡ / ẩn `k6` khỏi `PATH` | `GATE_K6=1 bash scripts/gate-p3.sh; echo $?` | Có dòng `SKIP k6: chưa cài` kèm lý do; dòng cuối `GATE P3: PASS (có SKIP)`; **không** âm thầm xanh và không báo PASS trơn |
| TC-P3-08-26 | AC9 (nhánh thiếu stack) | dừng stack (`pnpm dev:down`) | `bash scripts/gate-p3.sh; echo $?` | Các bước cần stack ghi `SKIP` kèm lý do (không FAIL mơ hồ, không im lặng); cờ cuối `PASS (có SKIP)` hoặc `FAIL` nếu bước không-cần-stack hỏng |
| TC-P3-08-27 | Chéo (DoD chung — SKILL bước 4) | DB trống | `pnpm dev` trên DB trống (migration `00007`, `00008`, `00009` chạy sạch); `cd backend-go && go test ./... 2>&1 \| tail -5` | Migration chạy sạch, không lỗi; test cũ **không đỏ**; seed có dữ liệu cho mọi màn mới (`/chat`, `/threads`) |
| TC-P3-08-28 | Chéo (luật mở rộng — SKILL bước 4b) | diff của sprint | `grep -rn "fetch(" frontend/src --include=*.ts --include=*.tsx \| grep -v "frontend/src/shared/"`; `grep -rn "OFFSET" backend-go/internal/store/queries/`; `grep -rniE "os\.(Create\|WriteFile)" backend-go/internal/` | Cả ba lệnh **0 kết quả** (mọi gọi mạng qua `apiClient` / `useSSE` / `shared/data/streamRequest.ts` — TLR-12); phân trang bằng con trỏ; không ghi đĩa cục bộ trong `internal/` |
| TC-P3-08-29 | Chéo (phản mẫu UI — SKILL bước 4c, 4d) | stack chạy | `bash scripts/ui-antipatterns.sh; echo $?`; `playwright-cli resize 375 812` rồi duyệt `/chat`, `/threads`, `/threads/[id]`, `/` | `rc=0`; không màn nào tràn ngang ở 375 px, vùng chạm ≥ 44 px, đủ trạng thái tải / rỗng / lỗi (`<PageState>`); đối chiếu DESIGN §14.2–§14.4 và 10 điều kiện §22 |

## Nhánh lỗi
- Seed chạy lần hai (idempotent) → TC-02; seed chậm quá ngưỡng → TC-03.
- k6 vượt ngưỡng → TC-16 (phải thoát ≠ 0).
- Gate dừng ở lỗi đầu → TC-18; thiếu `k6` → TC-25; thiếu stack → TC-26.
- `continue[]` rỗng → TC-08; `AI_CONFIRM` hết việc → TC-12.

## Phân quyền
- `continue[]` chỉ chứa phiên của **chính** sinh viên, không bao giờ của người khác (TC-09).
- `AI_CONFIRM` chỉ cho TA / TEACHER **của lớp**; Sinh viên và Admin không bao giờ nhận (TC-13).
- Golden / contract test xác nhận phản hồi sinh viên không có khoá cấm (TC-23).
- Bước ma trận quyền của cổng chạy đủ bốn vai seed (TC-24).

## Script chạy được
- `scripts/gate-p3.sh` (cùng `GATE_K6=1`) — TC-17…19, 24…26; bảng lệnh đầy đủ ở `docs/sprints/6/qc/gate-P3.md`.
- `scripts/check-chat-seed.mjs` (`threads`) — TC-01, 04.
- `docs/sprints/6/qc/scripts/scan-pii-leak.sh` — TC-06.
- `benchmarks/load/chat.js` (`SCENARIO=first_event|ttft`) — TC-14…16.
- `frontend/e2e/privacy.spec.ts private-chat.spec.ts threads.spec.ts today.spec.ts` — TC-08, 11, 20.
- Go: `./internal/today`, `./internal/contract` như ghi trong từng TC.

## Câu hỏi cho BA (Q-QC-…)
| # | Chỗ mơ hồ | Câu hỏi |
| --- | --- | --- |
| Q-QC-P308-1 | AC1 "`chat=150±5`" | Dung sai ±5 áp cho **tổng** số câu hỏi hay cho từng chủ đề A / B? Và "≈ 150 câu hỏi" là số **tin `USER`** hay số **phiên**? |
| Q-QC-P308-2 | AC2 vs AC5 của chính story | AC2 nói lớp 1 có 12 thread, trong đó 1 bài `REJECTED` "sinh viên **không** thấy"; phần Kiểm lại ghi `GET $Q/threads` của sinh viên trả `11`. Vậy bài `REJECTED` làm **cả thread** biến mất khỏi danh sách, hay chỉ bài AI biến mất còn thread vẫn hiện? Hai cách cho hai con số khác nhau (11 hoặc 12). |
| Q-QC-P308-3 | AC5 kịch bản `ttft` | Một kịch bản có **hai** ngưỡng ("< 1500 ms cache trúng" và "< 4000 ms có truy xuất"). k6 phân biệt hai trường hợp bằng tag nào (`ttft_cache_p95` / `ttft_rag_p95`)? Không có tên ngưỡng thì QC không kiểm được bằng mắt trong đầu ra k6. |
| Q-QC-P308-4 | AC6 + AC9 | AC6 nói cổng "dừng ở lỗi đầu", AC9 nói bước thiếu công cụ ghi `SKIP` và cờ cuối là `PASS (có SKIP)`. Nếu một bước `SKIP` nằm **trước** các bước khác thì cổng chạy tiếp hay dừng? (Đề nghị: `SKIP` không dừng, chỉ `FAIL` mới dừng — xin xác nhận.) |
| Q-QC-P308-5 | AC9 "bước ma trận quyền" | `gate-p3.sh` không có bước nào tên "ma trận quyền" trong danh sách của AC6 (`TestChatMatrix` / `TestThreadsMatrix` nằm trong `go test -race ./internal/chat/... ./internal/thread/...`). Có thêm bước riêng chạy bằng **token thật của bốn tài khoản seed** không, hay chỉ dựa vào test Go? |
| Q-QC-P308-6 | AC3 | "phiên chat riêng … có ≥ 1 tin trong 7 ngày" — tính theo tin của **sinh viên** hay bất kỳ tin nào (kể cả tin `ASSISTANT` `FAILED`)? Phiên chỉ có tin `FAILED` có vào `continue` không? |

## Lịch sử sửa TC
(chưa có — TC viết lần đầu 2026-10-10 theo spec `APPROVED` v1.1; chỉ sửa khi SPEC đổi, kèm số `proposals.md`)

Tổng TC: 29
