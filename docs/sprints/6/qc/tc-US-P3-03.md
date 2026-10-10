# QC test case — US-P3-03 (cắm che / khôi phục ở **một chỗ** trong `internal/llm`, `pii_masked_count`, `TestNoPayloadLeak`)
Nguồn: `docs/specs/FEAT-private-chat-pii/US.md` US-P3-03 AC1–AC10 (v1.2 APPROVED) + `SRS.md` 3.3, 4.3, 4.7.1 bước 9, 8 ("Riêng tư"), 9.2, 9.4; `TL-REVIEW.md` TLR-7 (bỏ `Passages` khỏi danh sách che; chat tự dựng câu suy giảm), TLR-8 (`CourseID == nil` → `MASK_FAILED`), TLR-15 (`llm_audit` `status='error'`, `error_kind='MASK_FAILED'`), ghi chú "Một chỗ che"; `docs/phases/P3.md` L3 + cổng `TestNoPayloadLeak`; `CLAUDE.md` nguyên tắc bất biến 3 và 4. Viết trước khi có code (pha 1), hộp đen: QC tự dựng kịch bản qua API thật rồi **tự quét** mọi payload do provider giả ghi lại bằng `docs/sprints/6/qc/scripts/p603-payload-scan.sh`; test Go của dev chỉ chạy **thêm**.

**Tiền điều kiện chung.** Như `tc-US-P3-02.md` (stack riêng của QC, seed đã chạy, biến `SRS.md` 9.1). Thêm: provider giả ghi **mọi** payload gửi đi ra tệp (một JSON mỗi lời gọi, gồm `messages[]` đủ vai `system`/`user`/`assistant`, `tools`, `input` của `Embed`, lời nhắc của `Structured`) tại `$DUMP` (`[CẦN: đường ghi payload của provider giả — biến môi trường / thư mục; xem handoff]`); hai provider giả `fake-1`, `fake-2` xếp chuỗi dự phòng; `$DUMP` được xoá trước mỗi TC. `$ROSTER` = tệp do QC sinh từ DB: 30 họ tên lớp C1 × 3 biến thể (có dấu / không dấu / đảo) + 30 MSSV + 30 email + SĐT + CCCD đã gieo. Công cụ: **S** shell / `curl`, **D** `psql`, **G** `go test`, **K** `p603-payload-scan.sh`.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P3-03-01 | AC1 | – | **G** `cd backend-go && go test ./internal/llm -run 'TestMaskOnlyInLLMGateway' -v` | `ok` |
| TC-P3-03-02 | AC1 | – | **S** `grep -rnE 'privacy\.(Mask\|Unmask\|NewStreamUnmasker)\|\.Mask\(\|NewStreamUnmasker\(' backend-go/internal backend-go/cmd --include=*.go \| grep -v '_test.go'` | Chỉ xuất hiện ở `internal/llm/**`, `internal/privacy/**` và dây nối `cmd/gateway`, `cmd/worker`. Bất kỳ dòng nào ở `internal/agent`, `internal/chat`, `internal/thread`, `internal/ingest`, `internal/rag` → **FAIL** (nguyên tắc bất biến 4) |
| TC-P3-03-03 | AC1 | – | **S** `go list -deps ./internal/agent \| grep -c internal/privacy` (lặp cho `chat`, `thread`, `ingest`, `rag`) rồi đọc **dùng gì**: `grep -rn 'privacy\.' internal/agent internal/thread --include=*.go \| grep -v _test.go` | Có import `privacy` là hợp lệ, nhưng **chỉ** để gọi `Classify` / `Detect` / `Redact` (hai việc đọc văn bản). Mọi lời gọi `Mask*` / `Unmask*` ở các gói đó → FAIL |
| TC-P3-03-04 | AC1 (phép thử có sức phân biệt) | – | **S** QC gieo tạm `internal/chat/qc_probe.go` gọi `privacy.Mask(...)` rồi chạy lại TC 01; sau đó xoá tệp và chạy lại | lần gieo: `TestMaskOnlyInLLMGateway` **đỏ**; sau khi xoá: **xanh**. Nếu gieo mà vẫn xanh → test của dev vô nghĩa, FAIL AC1 |
| TC-P3-03-05 | AC2 (tin nhắn hiện tại) | phiên `SID` của SVA; `$DUMP` rỗng | **S** gửi `"Bùi Thanh Khải 20229002 bui.khai@edupilot.local 0912345678 001203004567 học nhóm với em"`; **K** `p603-payload-scan.sh --dump "$DUMP" --roster "$ROSTER"` | `0` lần trùng bất kỳ chuỗi nào trong `$ROSTER`; payload có `[[SV_1]]`, `[[MSSV_1]]`, `[[EMAIL_1]]`, `[[SDT_1]]`, `[[CCCD_1]]` |
| TC-P3-03-06 | AC2 (lịch sử) | sau TC 05, cùng phiên | **S** gửi tiếp một tin **sạch** (`"Ý trên giải thích thêm giúp em"`); **K** quét payload lượt hai (có `CHAT_HISTORY_TURNS` lượt cũ) | lượt cũ trong `messages[]` **đã che** (`[[SV_1]]`), `0` tên / MSSV thật |
| TC-P3-03-07 | AC2 (kết quả tool) | tool trả dữ liệu có tên người (`search_library` với tài liệu mà tiêu đề chứa tên roster — `[CẦN SEED: một tài liệu C1 có họ tên sinh viên trong tiêu đề hoặc đoạn trích]`) | **S** hỏi `"Tìm tài liệu về chữ ký số"` để tool trả `Facts`; **K** quét payload | khối `<ngữ_cảnh>` chứa kết quả tool **đã che**; `0` tên thật |
| TC-P3-03-08 | AC2 (đầu vào nhúng) | – | **S** đăng một thread ở C1 có nội dung chứa tên roster (dùng lối `redact:true` để qua tường lửa) và một `precheck` cần nhúng; **K** quét payload của lời gọi `Embed` | payload `Embed.input` **đã che**; `0` tên / MSSV thật (Q14) |
| TC-P3-03-09 | AC2 (lời nhắc có cấu trúc) | – | **S** kích hoạt đường dùng `Structured` ở sprint 6 nếu có; nếu không có đường nào → ghi `KHÔNG KIỂM ĐƯỢC` kèm lý do và dựa vào TC 12 | payload `Structured` **đã che**. Xem Q-QC-P3-03-2 |
| TC-P3-03-10 | AC2 (`Passages` không nằm trong danh sách che — TLR-7) | mọi provider giả lỗi trước token đầu (đường suy giảm) | **S** gửi một câu `COURSE_QA` có ngữ cảnh; **K** quét payload; **S** đọc câu trả lời trên SSE | `0` payload nào chứa trường `Passages`; câu trả lời là dòng `"Trả lời tạm thời, trích nguyên văn từ tài liệu của lớp."` + trích dẫn từ `Hit`; **không** có câu `"giảng viên sẽ xem"` (quyết định PM, phương án (a)) |
| TC-P3-03-11 | AC2 (thiếu lớp → hỏng an toàn) | – | **G** `go test ./internal/llm -run 'TestMaskCoversAllPayloadParts\|TestMaskNoCourseFailsClosed' -v`; **S** đọc test để xác nhận ca "quên gắn lớp" khẳng định **0** payload tới provider | `ok`; bảng phủ **5 thành phần × 4 hàm** (`Chat`, `Stream`, `Structured`, `Embed`); ca `CourseID == nil` → `MASK_FAILED`, 0 lời gọi provider |
| TC-P3-03-12 | AC3 (một lần, trước chuỗi dự phòng) | `fake-1` luôn lỗi `500`, `fake-2` trả lời | **S** gửi một tin có 3 thực thể; **S** so hai tệp payload: `diff <(jq -S . $DUMP/fake-1-*.json) <(jq -S . $DUMP/fake-2-*.json)` | cả hai **đã che**; `diff` **rỗng** (giống hệt nhau) — che chạy **một** lần trước vòng dự phòng |
| TC-P3-03-13 | AC3 | – | **D** `select pii_masked_count from llm_audit where trace_id='<trace>' order by created_at` | số lần thay **không** bị nhân đôi theo số lần thử provider |
| TC-P3-03-14 | AC3 | – | **G** `go test ./internal/llm -run 'TestMaskOnceBeforeFallback' -v` | `ok` |
| TC-P3-03-15 | AC4 (Stream) | provider giả trả placeholder | **S** đọc **mọi** khung SSE `token` và `done`; `grep -cPi '\[\[\s*(SV\|MSSV\|EMAIL\|SDT\|CCCD)(_\d*)?'` | `0` — người gọi (`internal/chat`) chỉ nhận chữ đã khôi phục |
| TC-P3-03-16 | AC4 (`Chat` không stream) | worker AI trả lời thread (đường `llm.Chat`) | **D** `select body from forum_posts where kind='AI' order by created_at desc limit 1`; quét dạng placeholder | `0` placeholder trong `body`; tên thật (nếu có trong ngữ cảnh) đã khôi phục đúng |
| TC-P3-03-17 | AC4 (`Structured`) | – | **G** `go test ./internal/llm -run 'TestUnmaskBeforeCaller\|TestStructuredUnmaskAfterSchema' -v`; đọc test xác nhận **thứ tự**: kiểm schema trên bản **thô** rồi mới khôi phục | `ok`; nếu test khôi phục trước khi kiểm schema → FAIL (đầu ra thật của mô hình mới là thứ phải hợp schema) |
| TC-P3-03-18 | AC5 | – | **S** gửi tin có đúng **6** thực thể bị thay (2 tên + MSSV + email + SĐT + CCCD); **D** `select pii_masked_count, status from llm_audit order by created_at desc limit 1` | `pii_masked_count = 6`; khớp với `notice{masked:6}` trên SSE và `chat_messages.masked_count = 6` |
| TC-P3-03-19 | AC5 (audit không nội dung) | tin chứa canary `CANARY-7Q2X` | **D** quét **mọi** cột text/jsonb của `llm_audit` tìm canary và tìm `[[SV_` | `0` lần cả hai (audit chỉ có số đếm, trạng thái, `trace_id`) |
| TC-P3-03-20 | AC5 | – | **G** `go test ./internal/llm -run 'TestAuditPIIMaskedCount' -v`; `go test -tags integration ./internal/llm -run 'TestAuditPIIMaskedCountPG' -v` | `ok` cả hai |
| TC-P3-03-21 | AC6 (cổng G1 — test của dev) | roster 30 sinh viên C1 đã seed | **G** `cd backend-go && go test -tags integration ./internal/integration -run 'TestNoPayloadLeak' -v` | `ok`; in **≥ 30** payload đã quét và **≥ 10** placeholder; số nhỏ hơn → FAIL (phép đo rỗng nghĩa) |
| TC-P3-03-22 | AC6 (QC tự chạy kịch bản) | `$DUMP` rỗng | **S** chạy trọn kịch bản của AC6 do QC tự dựng: (a) chat riêng: tên + MSSV + email + SĐT + CCCD **của chính mình** và **của người khác**, mỗi tên ở **ba** biến thể; (b) Threads: đăng thread → worker AI trả lời; (c) nhúng câu hỏi; (d) đường suy giảm (mọi provider lỗi); (e) mỗi kênh một lượt có tool trả dữ liệu mang tên; **K** `p603-payload-scan.sh --dump "$DUMP" --roster "$ROSTER" --min-payloads 30 --min-placeholders 10` | **0** lần xuất hiện của: họ tên đầy đủ (3 biến thể) của **bất kỳ** sinh viên nào trong roster, MSSV, email, SĐT, CCCD; số payload quét ≥ 30; số placeholder ≥ 10 |
| TC-P3-03-23 | AC6 (phép thử có sức phân biệt) | – | **K** chạy `p603-payload-scan.sh` trên một thư mục mồi do QC tạo, trong đó **một** tệp payload có chèn sẵn `"Bùi Thanh Khải"` và một tệp khác có `"20229002"` | script báo **FAIL** và chỉ đúng tệp + chuỗi. Nếu báo PASS → script sai, phải sửa script trước khi chấm TC 22 |
| TC-P3-03-24 | AC6 (biến thể không dấu / đảo) | – | **K** xác nhận `$ROSTER` có đủ **3 biến thể mỗi tên** (`wc -l` = 30×3 + 30 MSSV + email + SĐT + CCCD) và quét cả dạng không phân biệt hoa-thường (`grep -iF`) | đủ số dòng; quét không phân biệt hoa-thường; `0` trùng |
| TC-P3-03-25 | AC7 (thiếu dây nối) | – | **G** `go test ./cmd/gateway ./cmd/worker -run 'TestStartupRequiresMasker' -v` | `ok` |
| TC-P3-03-26 | AC7 | – | **S** `grep -rn 'NoMask' backend-go --include=*.go \| grep -v '_test.go'` | Chỉ ở định nghĩa trong `internal/llm` và hai điểm dùng: `POST /admin/llm/providers/test`, ping. Bất kỳ điểm dùng nào trên đường có nội dung người dùng → FAIL |
| TC-P3-03-27 | AC7 | ADMIN | **S** `j -X POST $GW/api/v1/admin/llm/providers/test -H "$ADM" -d '{…}'`; **K** quét payload lời gọi đó | lời gọi **không** chứa nội dung do người dùng nhập (chỉ chuỗi thử cố định); `NoMask` dùng ở đây là hợp lệ |
| TC-P3-03-28 | AC8 (log không rò) | provider giả trả `400` và `500` có **thân phản hồi lặp lại payload**, và một ca quá thời hạn | **S** chạy ba ca; `grep -icE 'Vũ Hoàng Giang\|Bùi Thanh Khải\|bui thanh khai\|20229001\|20229002\|@edupilot.local\|0912345678\|CANARY-7Q2X\|p:\[\[' <log gateway> <log worker>` (mức `debug`) | `0` ở mọi mẫu; log chỉ có mã lỗi, nhà cung cấp, `trace_id`, thời gian |
| TC-P3-03-29 | AC8 | – | **G** `go test ./internal/llm -run 'TestProviderErrorsNotLogged' -v` | `ok` |
| TC-P3-03-30 | AC9 (phân quyền) | sau TC 22 | **S** `j $GW/api/v1/admin/llm/usage -H "$ADM"`; `j $GW/api/v1/admin/observability/requests -H "$ADM"`; quét cả hai thân JSON tìm canary, tên roster, `[[SV_` | `200`; thân **chỉ** có số đếm, `pii_masked_count`, trạng thái; `0` chuỗi nội dung / tên / placeholder |
| TC-P3-03-31 | AC9 (phân quyền) | – | **S** lặp hai route trên với `$TCH`, `$TA_`, `$SVA` | `403` cả sáu lượt |
| TC-P3-03-32 | AC9 | – | **G** `go test ./internal/llmconfig/... ./internal/contract/... -run 'TestUsageNoContent' -v` | `ok` |
| TC-P3-03-33 | AC10 (hiệu năng) | – | **G** `go test ./internal/llm -bench 'BenchmarkGatewayMask' -run '^$' -benchtime=200x` (payload 6 tin × 1.500 ký tự) | **≤ 5 ms/op** |
| TC-P3-03-34 | AC10 (không thêm lời gọi) | `$DUMP` rỗng | **S** gửi **một** tin `COURSE_QA`; đếm số tệp payload loại `Chat`/`Stream` và loại `Embed` trong `$DUMP` | đúng **1** lời gọi sinh chữ và **≤ 1** lời gọi nhúng cho một tin (D47); **G** `go test ./internal/agent ./internal/chat -run 'TestOneGenerationPerMessage' -v` → `ok` |
| TC-P3-03-35 | AC2 + AC5 (chéo lớp) | SVA có phiên ở C1 và ở C2 | **S** gửi ở phiên C2 một tin chứa tên của một sinh viên **chỉ** thuộc C1; **K** quét payload | tên đó **không** bị che (ngoài roster C2) nhưng cũng **không** phải lỗi; điều phải đúng: phạm vi roster lấy từ `CourseID` của **phiên đang dùng**, không phải lớp khác (đối chiếu `pii_masked_count`) |
| TC-P3-03-36 | AC9 (chat riêng) | – | **S** `curl -s -o /dev/null -w '%{http_code}'` các route chat của SVA bằng `$ADM` | `403`/`404` — ADMIN không có route đọc nội dung chat (`SRS.md` 2); không có đường nào cho Staff đọc nội dung đã che hay chưa che |

## Nhánh lỗi (SRS 3.3 — phần thuộc hook che)
| Tình huống | TC |
| --- | --- |
| Che lỗi / quá 50 ms / `CourseID == nil` → `MASK_FAILED`, 0 lời gọi provider, `llm_audit status='error'` | 11 (+ `tc-US-P3-02.md` TC 50) |
| Nhà cung cấp 1 chết, chuỗi dự phòng nhận payload khác | 12 |
| Mọi provider chết → đường suy giảm, `Passages` không gửi đi, không hứa "giảng viên sẽ xem" | 10 |
| Provider trả `400`/`500` có thân, quá thời hạn → log rò | 28 |
| Gateway / worker khởi động thiếu `Masker` | 25 |
| `NoMask` bị dùng trên đường có nội dung người dùng | 26, 27 |
| Placeholder lọt ra người gọi (Stream / Chat / Structured) | 15, 16, 17 |
| `llm_audit` / `/admin/*` chứa nội dung | 19, 30 |

## Phân quyền
| Ca | TC |
| --- | --- |
| TEACHER / TA / STUDENT gọi `/admin/llm/usage`, `/admin/observability/requests` → 403 | 31 |
| ADMIN đọc được số đếm nhưng **không** đọc được nội dung | 30, 32 |
| ADMIN không có route chat riêng | 36 |
| Phạm vi roster theo lớp của phiên, không chéo lớp | 35 |

## Script chạy được
- `docs/sprints/6/qc/scripts/p603-payload-scan.sh` — nhận `--dump <thư mục payload provider giả>`, `--roster <tệp chuỗi cấm>`, `--min-payloads`, `--min-placeholders`; quét **mọi** tệp payload (kể cả `system`, `tools`, `input` của `Embed`, lời nhắc `Structured`) bằng `grep -iF -f`, in tệp + chuỗi vi phạm, trả `rc=1` khi có vi phạm hoặc khi số payload / số placeholder dưới ngưỡng. Tệp `$ROSTER` sinh bằng `--build-roster "$DATABASE_URL" "<course_id>"` (30 tên × 3 biến thể + MSSV + email + SĐT + CCCD). Dùng ở TC 05–08, 22–24 và kiểm chính script ở TC 23.
- Test Go của dev chạy **thêm** ở TC 01, 11, 14, 17, 20, 21, 25, 29, 32, 33, 34.
- Cổng phase: `cd backend-go && go test ./internal/integration -run TestNoPayloadLeak -v` (`docs/phases/P3.md` dòng cổng nghiệm thu).

## Câu hỏi cho BA (Q-QC-…)
- **Q-QC-P3-03-1** — AC6 yêu cầu QC quét "mọi payload" nhưng spec không nói provider giả ghi payload ra đâu và theo định dạng nào. Dev ghi đường dẫn / biến môi trường vào handoff được không? Không có thì TC 05–10, 22–24, 34 ghi `KHÔNG KIỂM ĐƯỢC` (tính FAIL) — *chờ BA / dev*.
- **Q-QC-P3-03-2** — AC2 bắt buộc che "lời nhắc có cấu trúc" của `Structured`, nhưng sprint 6 không có đường nghiệp vụ nào gọi `Structured` (trích công thức điểm là P6). QC kiểm AC này thế nào ngoài test đơn vị của dev? — *chờ BA*.
- **Q-QC-P3-03-3** — AC1 cho phép `internal/agent` dùng `privacy.Classify` / `Detect`, còn AC "một chỗ duy nhất" chỉ cấm `Mask`/`UnmaskStream`. Khi `Classify` phải **nhúng** văn bản (SRS 4.4 mục 2) thì lời gọi `Embed` đó đi qua `internal/llm` (có che) — đúng không, hay `agent` được nhúng thẳng? QC đang chấm "phải qua `internal/llm`, có che" (TC 08) — *chờ BA*.
- **Q-QC-P3-03-4** — AC7 nói "tiến trình **không khởi động**" khi thiếu `Masker`, nhưng không có cách cấu hình nào từ ngoài để bỏ `Masker` (nó là dây nối trong mã). QC chỉ kiểm được bằng test của dev (TC 25) + quét `NoMask` (TC 26). Chấp nhận không? — *chờ BA*.

## Lịch sử sửa TC
(chỉ sửa khi SPEC đổi: ghi ngày, TC nào, lý do, số proposal PM đã chấp nhận)
- 2026-10-10 — tạo mới theo `US.md` v1.2 / `SRS.md` v1.2 (APPROVED).

Tổng TC: 36
