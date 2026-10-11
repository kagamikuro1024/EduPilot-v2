# Cổng nghiệm thu P3 — danh sách lệnh (viết ở pha 1, **chưa chạy**)
Nguồn: `docs/phases/P3.md` mục "Cổng nghiệm thu" + "Bạn tự kiểm"; `.claude/skills/gate/SKILL.md` bước 1–6 (DoD chung 4, luật mở rộng 4b, phản mẫu UI 4c, cổng UX 4d, mức luồng 4e); `docs/specs/FEAT-private-chat-pii/US.md` US-P3-08 AC6, AC7, AC9 (`scripts/gate-p3.sh`); `docs/UX.md` mục 6; `docs/design/DESIGN.md` §14.2–§14.4, §21–§22; `docs/FLOWS.md` F3, F4.

**Cách dùng:** pha 2 chạy **từng lệnh đúng như viết** (không sửa lệnh, không bỏ bước, không sửa test cho xanh), điền cột **Kết quả** (`PASS` / `FAIL` / `SKIP` + 10 dòng đầu ra liên quan) rồi kết luận ở mục cuối. Cột Kết quả để **trống** ở pha 1.

**Biến:** `GW` gốc gateway; `C1` lớp 1 (`761987`); `SVA`/`SVB` token sinh viên seed, `TA_`, `TCH`, `ADM`; `j='curl -sk -H "Content-Type: application/json"'`; `PSQL`; `$PW = pnpm -C frontend exec playwright test`. Giao diện nghiệm thu tay bằng **`playwright-cli`** (toàn cục; **không** chạy `playwright-cli install`). Sau mỗi lượt test có Docker: `docker volume prune -f`.

## A. Lệnh cổng của `docs/phases/P3.md`

| # | Lệnh | Kết quả mong đợi | TC liên quan | Kết quả |
| --- | --- | --- | --- | --- |
| A1 | `cd backend-go && go test -race ./internal/privacy/...` | `ok`, không cảnh báo đua dữ liệu | `tc-US-P3-02.md` | |
| A2 | `cd backend-go && go test ./internal/privacy -run TestUnmaskStream -v` | `ok` — token cắt ở **mọi** vị trí, kể cả giữa một placeholder | `tc-US-P3-02.md`; TC-P3-05-56 | |
| A3 | `cd backend-go && go test ./internal/integration -run TestNoPayloadLeak -v` | `ok` — quét mọi payload gửi provider giả: **0** MSSV / họ tên roster | `tc-US-P3-03.md`; TC-P3-05-39, TC-P3-06-47 | |
| A4 | `python benchmarks/eval_pii.py --min-recall 0.95 --max-false-block 0.05` | `E1 PASS recall=0.9x false_block=0.0x`, `echo $?` → `0`; sinh `benchmarks/reports/e1.json` + `e1.md` | TC-P3-07-16…19, 26 | |
| A5 | `pnpm -C frontend exec playwright test privacy.spec.ts` | Tất cả pass (tường lửa, hộp thoại hai lối, dòng che, không placeholder) | TC-P3-05-35, 36; TC-P3-06-07…13, 17 | |

## B. Bước của `scripts/gate-p3.sh` (US-P3-08 AC6; dừng ở lỗi đầu)

| # | Lệnh / bước | Kết quả mong đợi | TC liên quan | Kết quả |
| --- | --- | --- | --- | --- |
| B0 | `bash scripts/gate-p3.sh; echo $?` | Bảng PASS/FAIL theo đúng thứ tự B1…B10, dòng cuối `GATE P3: PASS`, mã thoát `0` | TC-P3-08-17 | |
| B1 | `go vet ./...` (trong `backend-go`) | Không cảnh báo | TC-P3-08-17 | |
| B2 | `go test -race ./internal/privacy/... ./internal/agent/... ./internal/chat/... ./internal/thread/...` | `ok` cả bốn gói, không đua dữ liệu | TC-P3-05-01, 06, 08, 12, 30, 51; TC-P3-06-03, 06, 16, 20, 21, 28, 29, 30, 35, 39, 43 | |
| B3 | `go test ./internal/privacy -run TestUnmaskStream -v` | `ok` | A2 | |
| B4 | `go test ./internal/integration -run TestNoPayloadLeak -v` | `ok` | A3 | |
| B5 | `go test ./internal/agent -run TestThreadsAgentHasNoPersonalTools -v` | `ok` — agent Threads **không** đăng ký tool cá nhân nào | TC-P3-06-46 | |
| B6 | `go test ./internal/chat ./internal/thread ./internal/contract -run TestStudentNeverSeesConfidence -v` | `ok` — duyệt cây JSON mọi route chat + threads của sinh viên, kể cả SSE | TC-P3-07-06 | |
| B7 | `go test ./internal/contract/...` | `ok` — 20 thao tác của SRS 6, phép chiếu sinh viên, phản hồi 4xx khớp schema | TC-P3-08-22, 23 | |
| B8 | `python benchmarks/eval_pii.py --min-recall 0.95 --max-false-block 0.05` | như A4 | TC-P3-07-16 | |
| B9 | `pnpm -C frontend exec playwright test privacy.spec.ts private-chat.spec.ts threads.spec.ts` | Tất cả pass, kể cả bước ngắt mạng và 3G chậm | TC-P3-08-20 | |
| B10 | `bash scripts/ui-antipatterns.sh` | `rc=0`, không vi phạm | TC-P3-05-52, TC-P3-06-36, TC-P3-08-29 | |
| B11 | `GATE_K6=1 k6 run benchmarks/load/chat.js --env SCENARIO=first_event` | `thresholds ✓`, `first_event_p95 < 300` | TC-P3-08-14 | |
| B12 | `GATE_K6=1 FAKE_LLM_TTFT_MS=300 k6 run benchmarks/load/chat.js --env SCENARIO=ttft` | `thresholds ✓`, `ttft_p95 < 1500` (cache trúng) / `< 4000` (có truy xuất) | TC-P3-08-15 | |
| B13 | Nhánh lỗi của cổng: cố ý làm hỏng `TestNoPayloadLeak` → `bash scripts/gate-p3.sh; echo $?` | `GATE P3: FAIL`, mã ≠ 0, các bước sau **không** chạy | TC-P3-08-18 | |
| B14 | Nhánh thiếu công cụ: gỡ `k6` khỏi `PATH` → `GATE_K6=1 bash scripts/gate-p3.sh` | Dòng `SKIP k6: chưa cài`, dòng cuối `GATE P3: PASS (có SKIP)` — không âm thầm xanh | TC-P3-08-25, 26 | |

## C. Definition of Done chung (SKILL bước 4)

| # | Mục | Lệnh | Kết quả mong đợi | TC liên quan | Kết quả |
| --- | --- | --- | --- | --- | --- |
| C1 | Migration sạch trên DB trống | `pnpm dev:down && docker volume prune -f && pnpm dev` | `00007`, `00008`, `00009` chạy sạch, không lỗi | TC-P3-08-27 | |
| C2 | Test cũ không đỏ | `cd backend-go && go test ./... 2>&1 \| tail -5`; `pnpm -C frontend lint` | Không gói nào FAIL; lint `rc=0` | TC-P3-08-27 | |
| C3 | API mới có trong `openapi.yaml` | `cd backend-go && go test ./internal/contract/... -run TestChatThreadsContract -v` | `ok`; đủ **20** thao tác của SRS 6 | TC-P3-08-22 | |
| C4 | STUDENT gọi API giảng viên → 403 | `j -X POST "$GW/api/v1/courses/$C1/posts/$P/verify" -H "$SVA" -i \| head -1` | `403` | TC-P3-06-31 | |
| C5 | Seed có dữ liệu cho màn mới | `node scripts/check-chat-seed.mjs` và `… threads` | `chat=150±5 … threads=12/3 idempotent=ok`; bảng đếm trạng thái khớp | TC-P3-08-01, 04 | |
| C6 | Không PII / secret trong log và diff | `bash docs/sprints/6/qc/scripts/scan-pii-leak.sh` (biến `NAME`, `MSSV`, `EMAIL`, `PHONE`, `CCCD` của `sv.gioi`); `git diff <base>... \| grep -niE 'sk-\|BEGIN .*PRIVATE KEY\|20229001'` | Mọi dòng quét = `0` (trừ mục "được phép"); diff không có secret / MSSV | TC-P3-05-39, TC-P3-06-14, TC-P3-08-06 | |

## D. Luật mở rộng trên diff của phase (SKILL bước 4b)

| # | Kiểm | Lệnh | Kết quả mong đợi | TC liên quan | Kết quả |
| --- | --- | --- | --- | --- | --- |
| D1 | Không `fetch(` ngoài `frontend/src/shared/` | `grep -rn "fetch(" frontend/src --include=*.ts --include=*.tsx \| grep -v "frontend/src/shared/"` | 0 kết quả (luồng chat dùng `shared/data/streamRequest.ts` + `useChatStream` — TLR-12) | TC-P3-08-28 | |
| D2 | Không `OFFSET` / danh sách thiếu `limit` | `grep -rn "OFFSET" backend-go/internal/store/queries/` | 0 kết quả; mọi danh sách phân trang bằng con trỏ | TC-P3-06-49, TC-P3-08-28 | |
| D3 | Không ghi đĩa cục bộ trong `internal/` | `grep -rniE "os\.(Create\|WriteFile\|MkdirAll)" backend-go/internal/` | 0 kết quả | TC-P3-08-28 | |
| D4 | Lời gọi LLM mới có `task` và đi qua Scheduler | Đọc diff `internal/chat`, `internal/thread`, `internal/agent`; `$PSQL "select distinct task, lane from llm_audit where created_at > now() - interval '1 hour'"` | Mỗi lời gọi có `task`; Threads ở làn `NEAR_REALTIME`, chat riêng ở `INTERACTIVE`; không lời gọi nào bỏ qua Scheduler | TC-P3-06-19 | |
| D5 | POST có tác dụng phụ nhận `Idempotency-Key` | Đối chiếu SRS 6 dấu **[K]**: #2, #11, #15, #16; #6/#7/#9 là ngoại lệ có chủ đích (4.7.0, TLR-1) | Mọi POST ghi khác đều yêu cầu khoá; thiếu khoá → `422` | TC-P3-05-28, 30, 34; TC-P3-06-13, 35 | |
| D6 | Việc dài trả 202 | Đọc diff đường ingest / AI trả lời Threads | Handler outbox chỉ **xếp hàng**, không chặn consumer (TLR-9) | TC-P3-06-20 | |
| D7 | Đọc diff tìm lan phạm vi / test bị sửa / thư viện lạ | `git diff <base>... --stat`; `git diff <base>... -- '*_test.go' 'frontend/e2e/*'`; `git diff <base>... -- go.mod package.json` | Không test cũ bị nới / xoá; không thư viện mới ngoài spec; không đổi file ngoài phạm vi sprint 6 | — | |

## E. Phản mẫu UI và cổng UX (SKILL bước 4c, 4d; `UX.md` mục 6; DESIGN §21–§22)

| # | Kiểm | Lệnh | Kết quả mong đợi | TC liên quan | Kết quả |
| --- | --- | --- | --- | --- | --- |
| E1 | `ui-antipatterns.sh` | `bash scripts/ui-antipatterns.sh; echo $?` | `rc=0` | B10 | |
| E2 | Hợp đồng route DESIGN §14 | Đối chiếu `/chat` (§14.2), `/threads` (§14.3), `/threads/[id]` (§14.4) với màn thật qua `playwright-cli snapshot` | Một Panel mỗi vùng làm việc; cột hội thoại ≤ 840 px; rail lọc ≥ 1100 px → popover dưới 1100 px; soạn tại chỗ, không modal; xác nhận là vạch xanh 1 px + chữ | TC-P3-05-52, TC-P3-06-36 | |
| E3 | 10 điều kiện DESIGN §22 | Duyệt tay từng điều kiện cho 3 route mới | Nêu rõ điều kiện nào chưa đạt (nếu có) | TC-P3-05-52, TC-P3-06-36, 37 | |
| E4 | Trạng thái tải / rỗng / lỗi | `playwright-cli route "**/api/v1/**" --status 500` → `reload` cho từng route; và xem lớp chưa có dữ liệu | Mọi màn mới dùng `<PageState>`, trạng thái rỗng có nút hành động (`Đặt câu hỏi`, gợi ý ở `/chat`) | TC-P3-05-52, TC-P3-06-37 | |
| E5 | **375 px** cho `/chat`, `/threads`, `/threads/[id]` | `playwright-cli resize 375 812`; với mỗi route: `playwright-cli eval "() => document.documentElement.scrollWidth - document.documentElement.clientWidth"` và đếm vùng chạm < 44 px; `playwright-cli screenshot <route>-375.png` | Cuộn ngang = `0`; vùng chạm < 44 px = `0`; dấu tiếng Việt không bị cắt; ảnh lưu `docs/sprints/6/qc/shots/` | TC-P3-05-53, TC-P3-06-38 | |
| E6 | Bốn mốc bề rộng + zoom 200 % | `playwright-cli resize` 1440 / 1100 / 1024 / 719; `playwright-cli eval` kiểm bố cục | Không vỡ bố cục ở mốc nào; zoom 200 % vẫn thao tác được | TC-P3-06-36 | |
| E7 | Bàn phím + axe | Duyệt luồng chính chỉ bằng `playwright-cli press tab/enter`; `$PW a11y.spec.ts` | Đi trọn luồng bằng bàn phím; axe không lỗi nghiêm trọng | — | |
| E8 | Mạng chậm 3G + ngắt mạng | `playwright-cli run-code "async (page) => { const s = await page.context().newCDPSession(page); await s.send('Network.enable'); await s.send('Network.emulateNetworkConditions', {offline:false, latency:400, downloadThroughput:51200, uploadThroughput:51200}); }"`; rồi `playwright-cli network-state-set offline` | Không mất dữ liệu đã nhập (nháp chat và Threads); có dải báo dễ hiểu; tải lại giữa chừng không mất câu trả lời | TC-P3-05-09, 52; TC-P3-06-34 | |
| E9 | Provider "Hôm nay" cho module mới | `j "$GW/api/v1/me/today" -H "$TCH" \| jq '[.tasks[].kind]'`; `j "$GW/api/v1/me/today" -H "$SVA" \| jq '.continue'` | `AI_CONFIRM` đã đăng ký (P3 — Q21, `proposals.md` #5); `continue[]` của sinh viên có mục `CHAT` | TC-P3-08-11, 07 | |
| E10 | Sinh viên không thấy từ kỹ thuật AI | `playwright-cli eval "() => document.body.innerText"` \| `grep -ciE 'RAG\|PII\|provider\|fallback\|trace\|confidence\|độ tin cậy\|điểm nháp'` trên `/chat`, `/threads` với vai sinh viên | `0` | TC-P3-07-08 | |

## F. Kiểm chéo bắt buộc (QC.md mục 3) — dù AC không nêu

| # | Kiểm | Lệnh | Kết quả mong đợi | TC liên quan | Kết quả |
| --- | --- | --- | --- | --- | --- |
| F1 | Quét payload gửi provider giả | `NAME='Vũ Hoàng Giang' NAME_NOACC='Vu Hoang Giang' MSSV=20229001 EMAIL=sv.gioi@edupilot.local PHONE=0912345678 CCCD=001099012345 bash docs/sprints/6/qc/scripts/scan-pii-leak.sh` | 0 MSSV / họ tên roster ở log provider và mọi bảng; dòng `log: placeholder [[SV_` > 0 (đã che) | TC-P3-05-38, 39; TC-P3-06-47 | |
| F2 | Phân quyền — vai sai | `for H in "$TA_" "$TCH" "$ADM"; do j "$GW/api/v1/chat/sessions?course_id=$C1" -H "$H" -i \| head -1; done`; `j "$Q/threads" -H "$ADM" -i \| head -1` | Tất cả `403` | TC-P3-05-47, TC-P3-06-31 | |
| F3 | Phân quyền — sinh viên ngoài lớp | `j "$Q/threads" -H "$SVOUT" -i \| head -1`; `j "$GW/api/v1/chat/sessions?course_id=$C1" -H "$SVOUT" -i \| head -1` | `403` | TC-P3-05-49, TC-P3-06-32 | |
| F4 | Phân quyền — SV A đọc dữ liệu SV B | `j "$GW/api/v1/chat/sessions/$SID/messages" -H "$SVB" -i \| head -1` và 4 route ghi khác | `404` (không lộ tồn tại) | TC-P3-05-48, 16 | |
| F5 | Idempotency với POST có tác dụng phụ | 20 yêu cầu song song cùng `Idempotency-Key` cho `POST …/messages` và `POST …/threads` | Đúng một cặp hàng chat; đúng một thread | TC-P3-05-30, TC-P3-06-35 | |
| F6 | Phân trang danh sách | `j "$Q/threads?limit=5"` → đi hết con trỏ; `j "$GW/api/v1/chat/sessions?course_id=$C1&limit=5"` | Không hàng lặp / mất; `limit` ngoài 1–100 → `422` | TC-P3-06-49, TC-P3-06-01 | |
| F7 | Khoá giờ thi (D56, Q2) | `j -X POST "$GW/api/v1/chat/sessions/$SID/messages" …` và `j -X POST "$Q/threads" …` khi đang thi; rồi `GET $Q/threads/$T` | Cả hai `409 EXAM_IN_PROGRESS`; thread cũ **vẫn đọc được**; `exam_events CHAT_BLOCKED` ≤ 1 dòng / lượt / phút | TC-P3-05-22…27; TC-P3-06-40…43 | |
| F8 | RAG — `ANSWER_KEY` không bao giờ truy xuất | Hỏi 5 câu nhắm đáp án ở `/chat` và một thread; `$PSQL "select count(*) from chat_messages where content like '%CANARY-7Q2X%'"` | `0`; không `citations` nào trỏ tài liệu `ANSWER_KEY` | TC-P3-05-57, TC-P3-06-48 | |
| F9 | Tên người đăng công khai (Q3) | `j "$Q/threads/$T" -H "$SVB" \| jq '.author'` | Có `full_name`, `role`, `is_me`; **không** `email` / MSSV / `user_id` | TC-P3-06-33 | |
| F10 | Stream chịu mạng xấu | 3G chậm + tải lại giữa chừng; nút `Dừng`; `OVERLOADED` | Không mất chữ, không lặp; `Dừng` huỷ tới provider ≤ 1 s; "AI đang bận. Thử lại sau khoảng 20 giây." + `Thử lại` | TC-P3-05-09, 14, 17, 18 | |

## G. Mức luồng (SKILL bước 4e)

| # | Luồng | Spec E2E | Kết quả mong đợi | TC liên quan | Kết quả |
| --- | --- | --- | --- | --- | --- |
| G1 | F3 (hỏi riêng tư) | `frontend/e2e/private-chat.spec.ts` | Đường chính đi trọn + ít nhất một nhánh lỗi (quá tải / tải lại giữa chừng / khoá giờ thi) | TC-P3-08-20 | |
| G2 | F4 (hỏi bài công khai) | `frontend/e2e/threads.spec.ts`, `privacy.spec.ts` | Đường chính đi trọn + nhánh "AI không đủ tin cậy" và "bị Loại" | TC-P3-08-20; TC-P3-06-21, 26 | |
| G3 | Dòng tick luồng ở `docs/PROGRESS.md` | — | Báo cho PM / dev cập nhật (QC **không** sửa file đó) | — | |

## H. "Bạn tự kiểm" của `docs/phases/P3.md` (chủ dự án làm tay sau khi cổng xanh)

1. Đăng thread "Em Nguyễn Văn A 20221234 được mấy điểm?" → bị chặn; bấm chuyển kênh → nội dung còn nguyên trong chat riêng. (TC-P3-06-09, 11)
2. Trong chat riêng gõ tên + MSSV của mình: câu trả lời hiện tên thật; mở `llm_audit` / log provider: chỉ thấy `[[SV_1]]`. (TC-P3-05-35, 39)
3. Hỏi điểm của MSSV khác → bị từ chối. (TC-P3-05-55)
4. Đọc toàn bộ diff `internal/privacy`. (D7)
5. DevTools → mạng 3G chậm → hỏi một câu → tải lại trang giữa chừng: câu trả lời không mất. (TC-P3-05-09)
6. Dùng chat trên điện thoại thật. (TC-P3-05-53)

## I. Ghi cho luận văn (SKILL bước 6)
`docs/thesis-notes/P3.md`: kết quả E1 (bảng P/R/F1, ma trận nhầm lẫn kênh, recall theo nhóm S1…S8, mục "Giới hạn đã biết" cho S7 — D46); sơ đồ tuần tự mask / unmask; ví dụ prompt trước / sau khi che (dữ liệu mô phỏng, D44).

## Kết luận
(điền ở pha 2: số lệnh PASS / FAIL / SKIP, số lỗi theo mức, `GATE P3: PASS | FAIL`)

## Ghi chú pha 1
- Lệnh A1…A5 lấy **nguyên văn** từ `docs/phases/P3.md`; không sửa lệnh. Nếu lệnh lệch thực tế (ví dụ `eval_pii.py` chưa có bộ dữ liệu — xem `Q-QC-P307-1`), ghi một dòng vào `docs/sprints/6/proposals.md`, **không** tự đổi lệnh.
- B14 / C1 nhắc: sau mỗi lượt chạy Docker, `docker volume prune -f` (`docs/team/CONTEXT.md` mục 4).
