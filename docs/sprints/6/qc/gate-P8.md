# QC gate — P8 (Tài liệu, thư viện, lịch)

Nguồn: `docs/phases/P8.md` mục "Cổng nghiệm thu" + "Bạn tự kiểm"; `.claude/skills/gate/SKILL.md` bước 1–6 (gồm 4, 4b, 4c, 4d, 4e); `docs/team/QC.md` pha 2 điểm 2–4 (kiểm chéo bắt buộc); `docs/specs/FEAT-docs-calendar/US.md` US-P8-03 AC16 (`scripts/gate-p8.sh`); `docs/UX.md` mục 6; `docs/design/DESIGN.md` §14.14–§14.16, §21, §22; `docs/FLOWS.md` F6, F13 (cả hai **hoàn tất ở P8**).

**Viết ở pha 1 (trước khi có code).** Cột `Kết quả` để **trống**, điền ở pha 2 khi có handoff. Mỗi dòng ghi `PASS` / `FAIL` / `SKIP (lý do)`; `SKIP` không tự động là PASS — xem Q-QC-P803-7.

## Luật chạy cổng (bắt buộc đọc trước khi bật bất cứ container nào)

- **RAM `docling-serve`:** máy 18 GB, colima đang chia 8 GiB với project khác; `docling-serve` có `mem_limit: 3g` và lúc OCR bản scan còn ngốn thêm bộ nhớ của worker. `plan.md` mục "Rủi ro" chốt: **QC và `dev` không dựng hai stack cùng lúc**. Trước khi chạy cổng, QC nhắn `dev`/PM, chờ xác nhận `dev` đã `docker compose down`, rồi mới `docker compose --profile ingest up -d`. Chạy xong: `docker compose down` + `docker volume prune -f`. **Không** `docker system prune -a --volumes`, **không** xoá volume có tên (`CONTEXT.md` §4).
- Ingest chỉ bật khi cần: profile `ingest` tắt mặc định. Các bước không cần docling thật chạy với docling **giả** hoặc profile tắt (tài liệu nằm `QUEUED` — US-P8-01 AC17).
- Chạy cổng trên **bản build** của frontend (`pnpm -C frontend build && pnpm -C frontend exec next start -p 3300`), cổng 3300 của QC; không đụng cổng của `dev`.
- Sau **mỗi** lượt test có Docker: `docker volume prune -f` (`CONTEXT.md` §4).
- Không sửa code, không nới assertion, không bỏ test. Thấy `dev` sửa test cho xanh → báo PM là lỗi nghiêm trọng (`QC.md` luật).

## A. Cổng nghiệm thu của phase (`docs/phases/P8.md`)

Chạy **từng lệnh, đúng như viết**, không sửa, không bỏ qua. Mỗi lệnh ghi PASS/FAIL + 10 dòng đầu ra.

| # | Lệnh | Kết quả mong đợi | TC liên quan | Kết quả |
| --- | --- | --- | --- | --- |
| A1 | `cd backend-go && go test ./internal/rag -run TestAnswerKeyNeverRetrieved -v` | `ok` — không đoạn `ANSWER_KEY` nào được truy xuất. Cần `-tags integration` để chạy thật (xem Q-QC-P803-7): chạy **cả hai** dạng và ghi rõ dạng nào `no test files`/`SKIP` | TC-P8-01-33, TC-P8-01-34 | |
| A2 | `cd backend-go && go test -race ./internal/library/... ./internal/calendar/... ./internal/document/...` | `ok` cả ba gói, **0** cảnh báo đua dữ liệu | TC-P8-02-*, TC-P8-03-* | |
| A3 | `curl -s "$API/calendar/feed.ics?token=$TOK" \| head -1` | `BEGIN:VCALENDAR` | TC-P8-03-23 | |

## B. Cổng tổng hợp `scripts/gate-p8.sh` (US-P8-03 AC16)

| # | Lệnh / bước trong script | Kết quả mong đợi | TC liên quan | Kết quả |
| --- | --- | --- | --- | --- |
| B1 | `go vet ./...` | sạch | — | |
| B2 | `go test -race ./internal/rag/... ./internal/library/... ./internal/calendar/... ./internal/document/... ./internal/ingest/...` | `ok` năm gói | TC-P8-01-*, TC-P8-02-*, TC-P8-03-* | |
| B3 | `TestAnswerKeyNeverRetrieved` | `ok` | TC-P8-01-33 | |
| B4 | `go test ./internal/contract/...` | `ok`; đủ 23 thao tác của SRS 6 có trong `backend-go/api/openapi.yaml` | TC-P8-01-58, TC-P8-02-45, TC-P8-03-56 | |
| B5 | `curl "$API/calendar/feed.ics?token=$TOK" \| head -1` | `BEGIN:VCALENDAR` | TC-P8-03-23 | |
| B6 | Playwright `documents.spec.ts library.spec.ts calendar.spec.ts` | pass | TC-P8-02-01…09, TC-P8-03-44…49 | |
| B7 | `bash scripts/ui-antipatterns.sh` | `rc=0`, mọi mục `✓` | TC-P8-02-37, TC-P8-03-49 | |
| B8 | `bash scripts/gate-p8.sh; echo $?` | in bảng PASS/FAIL theo đúng thứ tự B1→B7, **dừng ở lỗi đầu**, dòng cuối `GATE P8: PASS`, mã thoát `0`; bước thiếu stack/docling → `SKIP` kèm lý do + cờ cuối `PASS (có SKIP)` | TC-P8-03-50 | |
| B9 | Gieo lỗi: `$PSQL -c "update content_chunks set audience='ALL' where document_id='<ANSWER_KEY>'"` rồi chạy lại `gate-p8.sh` | dòng cuối `GATE P8: FAIL`, mã thoát ≠ `0`; **phục hồi DB sau khi đo** | TC-P8-03-50 | |

## C. Definition of Done chung (gate SKILL bước 4)

| # | Kiểm | Lệnh | Kết quả mong đợi | TC liên quan | Kết quả |
| --- | --- | --- | --- | --- | --- |
| C1 | Migration chạy sạch trên DB trống | `goose -dir backend-go/db/migrations postgres "$DATABASE_URL" up && … down && … up` | ba lượt `OK`; `00009_calendar` trước `00010_chunk_search`; `down` xoá đúng constraint + hai bảng + cột `tsv` + hàm `vn_bigram_query` | TC-P8-01-45, TC-P8-03-01, TC-P8-03-02, TC-P8-03-03 | |
| C2 | Test cũ không đỏ | `cd backend-go && go test ./...` và `pnpm -C frontend exec playwright test` | số pass ≥ số trước sprint; hai test P2 bị đổi theo TLR-2 (`TestSchema…` thêm `tsv`, `TestNoUnscopedChunkQuery` đổi sang danh sách truy vấn) phải **có số proposal / TLR dẫn chiếu** trong diff, không phải nới cho xanh | TC-P8-01-45 | |
| C3 | API mới có trong `backend-go/api/openapi.yaml` | `grep -c 'operationId' backend-go/api/openapi.yaml` trước/sau; đối chiếu 23 route của SRS 6 | đủ 23 thao tác, đủ mã lỗi 4xx của bảng SRS 6 | TC-P8-01-58, TC-P8-02-45, TC-P8-03-56 | |
| C4 | STUDENT gọi API giảng viên → 403 | `j $Q/documents -H "$SVA"`; `j -X POST $Q/calendar/events -H "$SVA" -d '{…}'`; `j -X POST $Q/uploads/presign -H "$SVA" -d '{…}'` | cả ba `403` | TC-P8-01-58, TC-P8-02-45, TC-P8-03-57 | |
| C5 | Seed có dữ liệu cho màn mới | `node scripts/check-docs-seed.mjs` | `docs=…READY answer_key=1 shared=…reembedded=0 events=2/1 idempotent=ok`; `/documents`, `/library`, `/calendar` đều có dữ liệu thật để xem | TC-P8-03-51 | |
| C6 | Không PII / secret trong log và diff | `docker compose logs --no-color --since 2h \| grep -ciE '20[0-9]{6}\|@edupilot\.local\|CANARY-7Q2X\|ics_token=\|token=[A-Za-z0-9_-]{20,}'`; `git diff 48315e1.. \| grep -niE 'sk-\|password\|secret\|BEGIN .*PRIVATE KEY'` | **0** ở cả hai (token ICS trong query phải bị Caddy che); dùng thêm mẫu của `scripts/canary-scan.sh` cho khoá nhà cung cấp | TC-P8-01-34, TC-P8-01-56, TC-P8-03-21 | |

## D. Luật mở rộng trên diff của phase (gate SKILL bước 4b)

Báo **từng** vi phạm kèm `file:dòng`.

| # | Luật | Lệnh | Kết quả mong đợi | Kết quả |
| --- | --- | --- | --- | --- |
| D1 | Không `fetch(` ngoài `frontend/src/shared/` | `grep -rn 'fetch(' frontend/src --include=*.ts --include=*.tsx \| grep -v '^frontend/src/shared/'` | 0 dòng | |
| D2 | Không `OFFSET`, không danh sách thiếu `limit` | `grep -rni 'offset' backend-go/internal/store/queries/`; rà các truy vấn danh sách mới (`documents`, `library`, `calendar`, `chunks`) | 0 `OFFSET`; mọi truy vấn danh sách có `LIMIT` + con trỏ | |
| D3 | Không ghi đĩa cục bộ trong `internal/` | `grep -rnE 'os\.(Create\|WriteFile\|MkdirTemp)\|ioutil\.WriteFile' backend-go/internal/` | 0 dòng mới (ingest phải stream `io.Pipe`, SRS 4.2) | |
| D4 | Lời gọi LLM mới có `task` và qua Scheduler | `grep -rn 'Embed(' backend-go/internal/ingest backend-go/internal/document`; `grep -rn 'openai-go' backend-go/internal/ \| grep -v internal/llm` | mọi lời gọi qua `internal/llm` với `task`/làn rõ ràng; **0** import SDK provider ngoài `internal/llm` | |
| D5 | POST có tác dụng phụ mới nhận `Idempotency-Key` | đối chiếu SRS 6: `#2 complete` **[K]**, và kiểm thực tế `retry`, `reindex`, `calendar/events` | `complete` bắt buộc `[K]`; các POST còn lại hoặc nhận `[K]` hoặc idempotent theo khoá DB — ghi rõ cái nào | |
| D6 | Việc dài trả 202 | `complete`, `retry`, `reindex`, `reindex` cả lớp | cả bốn trả `202 {job_id}` | |
| D7 | Lan phạm vi / test bị sửa / thư viện lạ | `git diff 48315e1.. --stat`; `git diff 48315e1.. -- go.mod frontend/package.json`; `git diff 48315e1.. -- '*_test.go' 'frontend/e2e/**'` | không file ngoài phạm vi P3/P8; không thư viện ICS (SRS 4.8: viết tay); mọi test bị sửa có dẫn chiếu TLR / proposal | |

## E. Giao diện (gate SKILL bước 4c)

| # | Kiểm | Lệnh | Kết quả mong đợi | TC liên quan | Kết quả |
| --- | --- | --- | --- | --- | --- |
| E1 | `scripts/ui-antipatterns.sh` | `bash scripts/ui-antipatterns.sh` | `rc=0`, mọi mục `✓` | TC-P8-02-37 | |
| E2 | Hợp đồng route `DESIGN.md` §14.14 `/documents` | `playwright-cli -s=gv goto $APP/documents` → `snapshot` | bảng đủ 7 cột; vùng thả tại chỗ không modal; `ANSWER_KEY` đúng hai dòng chữ | TC-P8-02-01, 05, 09 | |
| E3 | §14.15 `/library` | `playwright-cli -s=sv goto $APP/library` → `snapshot` | tìm kiếm trước; danh sách gọn, dấu loại tệp bằng chữ; không thẻ kiểu chợ ứng dụng; **không** nút `Luyện đề này` | TC-P8-02-22, 35 | |
| E4 | §14.16 `/calendar` | `playwright-cli -s=sv goto $APP/calendar` → `snapshot` ở 1280 và 375 | tuần ≥ 720 px / danh sách < 720 px / có tháng; màu tiết chế, đỏ chỉ cho bài thi ≤ 48 h; `Thêm vào lịch` là tiện ích phụ | TC-P8-03-44, 45, 46, 47 | |
| E5 | 10 điều kiện `DESIGN.md` §22 | đối chiếu từng điều cho `/documents`, `/library`, `/library/[id]`, `/calendar`; nêu rõ điều nào chưa đạt | 10/10 mỗi màn | TC-P8-02-01…09, 35…38; TC-P8-03-44…49 | |
| E6 | Phản mẫu §21 | `eval "document.querySelectorAll('[data-ep-panel] [data-ep-panel]').length"`; đếm nút chính đỏ mỗi vùng; tìm thẻ KPI | panel lồng = 0; ≤ 1 nút chính đỏ mỗi vùng; không dải 3 thẻ KPI (thống kê tài liệu là **một dải gọn**); không modal cho sửa đơn giản | TC-P8-02-18, TC-P8-03-11, 46 | |
| E7 | Provider "Hôm nay" cho module mới | module lịch có việc cần người xử lý? | nhắc 24 h đi qua `notifications` (chuông), không cần Provider mới; nếu `dev` thêm việc dạng "cần xử lý" mà chưa đăng ký Provider → báo thiếu | — | |

## F. Cổng UX (gate SKILL bước 4d, `docs/UX.md` mục 6)

| # | Kiểm | Lệnh | Kết quả mong đợi | TC liên quan | Kết quả |
| --- | --- | --- | --- | --- | --- |
| F1 | axe không lỗi nghiêm trọng | `playwright-cli run-code` nạp `axe-core` trên 4 route mới × 2 vai | 0 lỗi mức `serious`/`critical` | TC-P8-02-08 | |
| F2 | 375 px: không tràn ngang, chạm ≥ 44 px | `playwright-cli resize 375 812` + hai đoạn `eval` trên `/chat`, `/threads`, `/library`, `/library/[id]`, `/calendar`, `/documents` | `scrollWidth ≤ innerWidth` ở mọi route; 0 vùng chạm < 44 px (`/documents` xem Q-QC-P802-4) | TC-P8-02-37, TC-P8-03-49 | |
| F3 | Đủ trạng thái tải / rỗng / lỗi, rỗng có nút hành động | `route … --status 500` / `--delay 3000` / lớp trống trên 4 route | mọi màn dùng `PageState`; rỗng có hành động | TC-P8-02-04, 36; TC-P8-03-48 | |
| F4 | Bốn mốc bề rộng + zoom 200% + chữ Việt dài | `resize` 375 / 800 / 1200 / 1440; `eval` đặt zoom 2 | không vỡ bố cục, dấu không bị cắt | TC-P8-02-38, TC-P8-03-49 | |
| F5 | Mạng chậm 3G + ngắt mạng giữa thao tác chính | `playwright-cli run-code` CDP `Network.emulateNetworkConditions` (download 400 kbps, latency 400 ms) khi tải tài liệu và khi stream chat; `network-state-set offline` giữa lúc tải lên | không mất dữ liệu đã nhập; thông báo dễ hiểu; hàng tải lên giữ tên tệp + `Tải lại` | TC-P8-02-39, 40, 41 | |
| F6 | Lighthouse mobile của route mới | `pnpm -C frontend lighthouse` (nếu đã cấu hình) | đạt ngân sách `UX.md` mục 3; chưa cấu hình → `SKIP` kèm lý do | — | |
| F7 | Văn bản là tiếng Việt, đã đọc lại một lượt | rà chữ trên 4 route | đúng chữ trong SRS 7; **không** câu giải thích dưới tiêu đề khối / dưới từng dòng; tooltip chỉ ở cú pháp hoặc lý do nút khoá (`CLAUDE.md` luật chữ trên UI) | TC-P8-02-09, 36; TC-P8-03-22, 47 | |

## G. Mức luồng (gate SKILL bước 4e)

P8 **hoàn tất** F6 và F13 (`FLOWS.md` mục 1). Mỗi luồng cần spec E2E đi trọn đường chính **và ít nhất một nhánh lỗi**.

| # | Luồng | Kiểm | Kết quả mong đợi | TC liên quan | Kết quả |
| --- | --- | --- | --- | --- | --- |
| G1 | F6 tài liệu: tải lên → xử lý nền → RAG + thư viện | `pnpm -C frontend exec playwright test documents.spec.ts library.spec.ts` | có spec đi trọn: tải lên → `Sẵn sàng` → sinh viên tìm thấy ở `/library` → hỏi AI có trích nguồn; có ≥ 1 nhánh lỗi (tệp sai loại hoặc mất mạng giữa chừng) | TC-P8-01-15, 16, 33; TC-P8-02-22, 30, 39 | |
| G2 | F13 lịch và nhắc | `pnpm -C frontend exec playwright test calendar.spec.ts` + `p8-03-reminder.sh` | có spec đi trọn: tạo sự kiện → thấy trên lịch → feed ICS → nhắc 24 h không trùng; có ≥ 1 nhánh lỗi (giờ sai hoặc token hỏng) | TC-P8-03-09, 23, 34, 52, 54 | |
| G3 | F3 (phần P8 góp vào) | sinh viên hỏi "khi nào thi?" và "tài liệu nào về …?" | tool lịch + `search_library` chạy, giờ đúng VN, không lộ `ANSWER_KEY` | TC-P8-02-33, TC-P8-03-30, 33 | |
| G4 | Cập nhật `docs/PROGRESS.md` | đọc dòng tick luồng | F6 và F13 được tick sau khi G1, G2 PASS; nếu chưa → báo thiếu (QC **không** tự sửa) | — | |

## H. Kiểm chéo bắt buộc của QC (`QC.md` pha 2 điểm 3) — làm dù AC không nêu

| # | Kiểm chéo | Lệnh | Kết quả mong đợi | TC liên quan | Kết quả |
| --- | --- | --- | --- | --- | --- |
| H1 | Phân quyền: vai khác / sinh viên ngoài lớp | `docs/sprints/6/qc/scripts/p8-02-perm.sh` (ma trận vai × route cho `/documents`, `/library`, `/calendar`, upload, token ICS) | mọi ô đúng bảng SRS 2 / AC phân quyền; ngoài lớp → 403; tài nguyên của người khác → 404 | TC-P8-01-58…62, TC-P8-02-45…49, TC-P8-03-56…60 | |
| H2 | Sinh viên A đọc dữ liệu của sinh viên B | A gọi job / phiên chat / `personal_state` / feed ICS của B | tất cả → 404 hoặc không thấy; feed của B chỉ dữ liệu của B | TC-P8-01-62, TC-P8-02-49, TC-P8-03-59, 60 | |
| H3 | Không PII trong log và payload LLM | `grep` canary + MSSV + họ tên roster trên `/tmp/p8-llm.jsonl` và log compose | **0** MSSV, **0** họ tên roster, **0** `CANARY-7Q2X` | TC-P8-01-33, 34; TC-P8-03-33 | |
| H4 | Idempotency với POST có tác dụng phụ | `complete` (hai lần cùng khoá, hai lần khác khoá song song), `retry`, `reindex`, nhắc 24 h | một dòng `documents`; một job; một chuông mỗi (người, nguồn, giờ) | TC-P8-01-09, 14, 25, 57; TC-P8-03-35 | |
| H5 | Phân trang mọi danh sách | `/documents`, `/documents/{id}/chunks`, `/library`, `/calendar` với `limit=10` → đi hết bằng `next_cursor`; `limit=101` | không lặp, không sót, `limit` kẹp ≤ 100, không `OFFSET` | TC-P8-02-03, 26; TC-P8-03-07 | |
| H6 | Trạng thái tải / rỗng / lỗi + 375 px với màn mới | như F2, F3 | đủ ba trạng thái, 375 px sạch | TC-P8-02-04, 36, 37; TC-P8-03-48, 49 | |
| H7 | Migration chạy sạch trên DB trống | như C1 | sạch | TC-P8-01-45, TC-P8-03-01 | |
| H8 | Khoá giờ thi | sinh viên có lượt thi `IN_PROGRESS` → chat riêng, `Hỏi AI về tài liệu`, đăng thread mới bị khoá; thread **cũ** vẫn đọc được | khoá đúng ba lối; đọc thread cũ vẫn được | TC-P8-02-32, TC-P8-02-34 (+ TC của US-P3-05, US-P3-06) | |
| H9 | ANSWER_KEY không bao giờ truy xuất được | `p8-01-canary.sh` (4 đường tấn công) | 0 lần canary | TC-P8-01-33, 34 | |
| H10 | Token ICS không lưu thô | `$PSQL` kiểm cột khớp `^[0-9a-f]{64}$`; xoay → link cũ 404 | 100 % giá trị là băm; link cũ hết hiệu lực ngay | TC-P8-03-18, 19, 21 | |
| H11 | Nhắc không gửi trùng | chạy job 2 lần và 20 lần song song | `reminder_log` / chuông / thư không tăng | TC-P8-03-35 | |
| H12 | Ingest lặp cùng `content_hash` | tải lại cùng tệp; đẩy lại việc vào `ep:ingest` | 409 `DOCUMENT_DUPLICATE`; không hai bộ đoạn | TC-P8-01-11, 25 | |
| H13 | Tệp scan tiếng Việt | `Quyche.pdf` qua lượt OCR | `READY`, giữ đúng dấu ("học vụ" ≥ 1 lần) | TC-P8-01-19, 42 | |

## I. "Bạn tự kiểm" của chủ dự án (in lại khi tất cả PASS — gate SKILL bước 6)

| # | Việc chủ dự án làm tay | QC đã dựng sẵn gì | Kết quả |
| --- | --- | --- | --- |
| I1 | Đăng ký link ICS vào Google Calendar thật | URL feed của một tài khoản seed (QC in ra ở báo cáo); TC-P8-03-28 | |
| I2 | Sinh viên hỏi AI nội dung **chỉ có trong đáp án** → AI không trả lời được từ đáp án | tài liệu `dapan-giuaky.pdf` có canary; câu hỏi mẫu ở `p8-01-canary.sh`; TC-P8-01-33 | |
| I3 | Đổi hạn một bài → hỏi AI lại → ngày mới (ở sprint 6 là **đổi giờ sự kiện / bài thi**; hạn bài tập ở P7) | TC-P8-03-17 | |

Nhắc ghi `docs/thesis-notes/P8.md` sau khi cổng PASS.

## J. Dọn dẹp sau cổng

`docker compose down` → `docker volume prune -f` → báo `dev`/PM là đã nhả máy. Không `pkill node` chung, không đụng cổng của `dev`, chỉ commit file trong `docs/sprints/6/qc/**`.

## Kết luận

| Mục | Số PASS / tổng | Ghi chú |
| --- | --- | --- |
| A. Cổng phase | / 3 | |
| B. `gate-p8.sh` | / 9 | |
| C. DoD chung | / 6 | |
| D. Luật mở rộng | / 7 | |
| E. Giao diện | / 7 | |
| F. Cổng UX | / 7 | |
| G. Mức luồng | / 4 | |
| H. Kiểm chéo | / 13 | |

**GATE P8: (PASS / FAIL)** — "gần được" là FAIL; một mục FAIL → cổng FAIL. Nguyên nhân gốc + đề xuất sửa ghi ở `report-*.md`; QC **không** tự sửa code.
