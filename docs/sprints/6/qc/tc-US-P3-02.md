# QC test case — US-P3-02 (`internal/privacy`: `Detect` / `Redact` / `Mask` / `Unmask` / `StreamUnmasker`)
Nguồn: `docs/specs/FEAT-private-chat-pii/US.md` US-P3-02 AC1–AC14 (v1.2 APPROVED) + `SRS.md` 3.3 (nhánh lỗi), 4.1 (hằng số), 4.2.1–4.2.6, 5.7 (khoá Redis), 6 (#6, #14), 8, 9.1, 9.4; `TL-REVIEW.md` TLR-6 (máy trạng thái theo tiền tố, bất biến chỉ áp cho **dạng placeholder**), TLR-9 (vô hiệu roster không bị việc AI chặn), TLR-15 (`llm_audit` `MASK_FAILED`); `QUESTIONS.md` Q8 (không che tên giảng viên / TA), Q14. Viết trước khi có code (pha 1), hộp đen: QC tấn công qua **API thật** (`POST …/threads/precheck` cho `Detect`/`Redact`, `POST /chat/sessions/{sid}/messages` + payload của provider giả cho `Mask`/`Unmask`), đọc Redis bằng `redis-cli`; test Go của dev chỉ chạy **thêm**.

**Tiền điều kiện chung.** Stack riêng của QC (Postgres + Redis + gateway + worker + **provider giả có ghi lại mọi payload**), seed đã chạy (`scripts/seed.mjs`). Biến theo `SRS.md` 9.1: `GW`, `Q=$GW/api/v1/courses/$C1`, `Q2=$GW/api/v1/courses/$C2`, `SVA` (sv.gioi@edupilot.local — **Vũ Hoàng Giang**, MSSV `20229001`), `SVB` (sv.kha@edupilot.local — **Bùi Thanh Khải**, MSSV `20229002`), `SVC` (sv.nguyco@ — **Ngô Ngọc Cẩm**, `20229003`), `TA_`, `TCH`, `ADM`, `j`, `idem`, `PSQL`, `RDS`. `C1` = lớp `761987` (roster 30 sinh viên), `C2` = lớp `761988`. `SID` = id phiên chat riêng của SVA ở C1 (`POST /chat/sessions`). `SVX` = một sinh viên **chỉ** thuộc C2 (`sv31@edupilot.local`); tên và MSSV của sv04…sv30, sv31 sinh ngẫu nhiên lúc seed → QC lấy bằng `$PSQL -c "select u.full_name, e.student_code_snapshot from enrollments e join users u on u.id=e.user_id where e.course_id='<C1>' and e.status='ACTIVE' and e.role_in_course='STUDENT'"` (`[CẦN SEED: tên 27 sinh viên sv04…sv30 không cố định trong mã seed]`). Công cụ: **S** shell / `curl`, **R** `redis-cli`, **D** `psql`, **G** `go test`, **P** `docs/sprints/6/qc/scripts/p602-attacks.py`.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P3-02-01 | AC1 | SVA ở C1 | **S** `j -X POST $Q/threads/precheck -H "$SVA" -d '{"body":"Bài lab 3 của 20221234 nộp chưa?"}'` | `200`; `allowed=false`; `reasons` có `{"type":"MSSV","count":1}`; `redacted_text` = `"Bài lab 3 của [đã ẩn] nộp chưa?"` |
| TC-P3-02-02 | AC1 | – | **S** cùng lệnh với `"body":"Mã B21DCAT123 có trong danh sách không?"` (mã chữ-số 6–15 theo ràng buộc `student_code_snapshot`) | `allowed=false`, `reasons[].type='MSSV'` |
| TC-P3-02-03 | AC1 | – | **S** ba lượt: `"MSSV 123456 là ai"`, `"mã số sinh viên 1234567890123 ạ"`, `"mssv: abc123xyz"` | cả ba `allowed=false` với `MSSV` (bắt theo cụm dẫn, không cần đúng dạng `20…`) |
| TC-P3-02-04 | AC1 | – | **S** ba lượt email: `nam.nt@edupilot.local`, `abc.xyz+tag@gmail.com`, `sv@cntt.hust.edu.vn` | cả ba `allowed=false`, `type='EMAIL'`; `redacted_text` thay đúng một `[đã ẩn]` mỗi lượt |
| TC-P3-02-05 | AC1 | – | **S** bốn lượt SĐT: `0912345678`, `+84 912 345 678`, `0912.345.678`, `09 1234 5678` | cả bốn `allowed=false`, `type='PHONE'` |
| TC-P3-02-06 | AC1 | – | **S** hai lượt CCCD: `001203004567`, `001 203 004 567` | cả hai `allowed=false`, `type='CCCD'` |
| TC-P3-02-07 | AC1 (âm tính) | – | **P** `p602-attacks.py --set negatives-number`: `"Mở cổng 8080"`, `"Đề thi năm 2022"`, `"IP nội bộ 192.168.1.10"`, `"CVE-2021-44228 là lỗi gì"`, `"dãy 1234567890 có ý nghĩa gì"`, `"khoá RSA 2048 bit"`, `"Bài 2022 trong giáo trình"`, `"hash 4f3c2b1a9d8e7f6a5b4c3d2e"` | **mọi** mẫu `allowed=true`, `reasons=[]`, `personal_question=false`; `redacted_text` **bằng đúng từng byte** `body` |
| TC-P3-02-08 | AC1 | – | **S** `"CCCD của tôi 0912345678901"` (11 số) và `"0912345678"` trong cùng câu | chỉ bắt đúng loại tương ứng; **không** đếm trùng một khoảng vừa là `PHONE` vừa là `CCCD` (`count` tổng = số khoảng thật) |
| TC-P3-02-09 | AC1 | – | **S** `"Gọi văn phòng khoa 0241234567"` (đầu số cố định 02x) | Theo SRS 4.2.2 ("đầu số di động 03/05/07/08/09") → `allowed=true`. Chấm theo Q-QC-P3-02-1 |
| TC-P3-02-10 | AC1 | – | **G** `go test ./internal/privacy -run 'TestDetectRegexPositive\|TestDetectRegexNegative' -v` và **đếm** số ca in ra | `ok`; ≥ **20** mẫu dương và ≥ **15** mẫu âm (AC1 yêu cầu); ít hơn là FAIL |
| TC-P3-02-11 | AC2 | SVA ở C1, SVB trong roster C1 | **S** `precheck` với `"Bùi Thanh Khải được mấy điểm lab?"` | `allowed=false`, `type='NAME'`, `count=1`; `redacted_text` = `"[đã ẩn] được mấy điểm lab?"` |
| TC-P3-02-12 | AC2 | – | **S** `"bui thanh khai lam bai nay chua"` (không dấu, chữ thường) | `allowed=false`, `type='NAME'` |
| TC-P3-02-13 | AC2 | – | **S** `"BÙI THANH KHẢI"` (hoa toàn bộ) | `allowed=false`, `type='NAME'` |
| TC-P3-02-14 | AC2 | – | **S** hai lượt đảo thứ tự: `"Khải Bùi Thanh nộp bài chưa"`, `"Khải Bùi có đi học không"` | cả hai `allowed=false`, `type='NAME'` |
| TC-P3-02-15 | AC2 | – | **S** dạng rút: `"Bùi Khải"` | `allowed=false`, `type='NAME'` |
| TC-P3-02-16 | AC2 (âm tính) | – | **P** `p602-attacks.py --set negatives-name`: `"Khải"`, `"An"`, `"anh hiểu sai chỗ này"`, `"Hoa Kỳ dùng chuẩn nào"`, `"cần minh chứng cho luận điểm"`, `"MIT công bố bài báo"`, `"Mai thi lại được không"` (một âm tiết) | **mọi** mẫu `allowed=true`, `reasons=[]` (chỉ khớp ≥ 2 âm tiết, theo ranh giới từ) |
| TC-P3-02-17 | AC2 (ranh giới từ) | – | **D** `select full_name from users u join enrollments e on e.user_id=u.id where e.course_id='<C1>' and u.full_name ~ ' An$' limit 1` → tên `X`; **S** `precheck` với `X` và với `X` đổi âm cuối `An`→`Anh` | `X` → `allowed=false`; bản `…Anh` → `allowed=true` (không khớp tiền tố âm tiết). `[CẦN SEED: nếu roster C1 không có sinh viên tên cuối "An", QC gieo một sinh viên như vậy bằng API Giảng viên trước khi chạy]` |
| TC-P3-02-18 | AC2 | – | **G** `go test ./internal/privacy -run 'TestRosterNameVariants\|TestRosterNoCommonWordFalsePositive\|TestRosterWordBoundary' -v` | `ok` |
| TC-P3-02-19 | AC3 (theo lớp) | SVX chỉ ở C2; SVA ở C1; một tài khoản ở cả hai lớp | **S** `precheck` ở C1 với tên đầy đủ của SVX; rồi `precheck` ở C2 (bằng token của một thành viên C2) với **cùng** chuỗi | ở C1: `allowed=true`; ở C2: `allowed=false` `type='NAME'` — từ điển **chỉ** của lớp đang hỏi |
| TC-P3-02-20 | AC3 (cache) | Redis sạch khoá roster (`$RDS DEL ep:roster:$C1`) | **S** một `precheck` ở C1; **R** `$RDS EXISTS ep:roster:$C1`; `$RDS TTL ep:roster:$C1`; `$RDS TYPE ep:roster:$C1` | `1`; TTL là số dương **≤ 3600**; kiểu `string` (JSON) đúng SRS 5.7 |
| TC-P3-02-21 | AC3 (vô hiệu khi thêm thành viên) | sinh viên `sv.moi` (Lý Thu Minh, `20229004`) chưa ở C1 | **S** ghi lại `t0`; duyệt `sv.moi` vào C1 bằng API Giảng viên; **R** vòng lặp 0,5 s: `$RDS EXISTS ep:roster:$C1`; ngay khi `0` thì **S** `precheck` ở C1 với `"Lý Thu Minh"` | khoá bị xoá trong **≤ 5 s** kể từ `t0`; `precheck` kế tiếp → `allowed=false` `type='NAME'` |
| TC-P3-02-22 | AC3 (mời ra) | `sv.moi` đang ở C1 | **S** mời `sv.moi` ra khỏi C1; chờ ≤ 5 s; `precheck` ở C1 với `"Lý Thu Minh"` | `allowed=true` (không còn trong từ điển) |
| TC-P3-02-23 | AC3 (không bị việc AI dài chặn — TLR-9) | – | **S** đăng ≥ 3 thread ở C1 để hàng việc AI bận (provider giả đặt `FAKE_LLM_TTFT_MS=15000`); **ngay sau đó** đổi thành viên như TC 21 và đo lại thời gian xoá khoá | vẫn **≤ 5 s** (việc vô hiệu chạy ở `outbox.Chain`, việc AI ở hàng việc dài riêng); > 5 s là FAIL mức nghiêm trọng (vấn đề riêng tư, không chỉ trễ) |
| TC-P3-02-24 | AC3 (Q8) | – | **S** `precheck` ở C1 với họ tên đầy đủ của **giảng viên** và của **TA** lớp đó (`$PSQL` lấy tên) | `allowed=true` cả hai — giảng viên / TA **không** vào từ điển |
| TC-P3-02-25 | AC3 | – | **G** `go test ./internal/privacy -run 'TestRosterScopedToCourse\|TestRosterExcludesStaff' -v`; `go test -tags integration ./internal/privacy -run 'TestRosterInvalidatedOnMemberChange\|TestRosterInvalidateNotBlockedByLongJob' -v` | `ok` cả hai lượt |
| TC-P3-02-26 | AC4 | – | **S** `precheck` với `"Bùi Thanh Khải 20229002 bui.khai@edupilot.local 0912345678"` | `redacted_text` = `"[đã ẩn] [đã ẩn] [đã ẩn] [đã ẩn]"`; đúng **4** dấu; mỗi khoảng **một** dấu, **không** suy ra được độ dài gốc (mọi `[đã ẩn]` giống nhau); phần chữ còn lại giữ nguyên |
| TC-P3-02-27 | AC4 (idempotent) | – | **S** gửi lại chính `redacted_text` của TC 26 vào `precheck` | `allowed=true`; `redacted_text` **bằng đúng từng byte** đầu vào (`Redact(Redact(x)) == Redact(x)`) |
| TC-P3-02-28 | AC4 (không PII → nguyên văn) | – | **S** `precheck` với chuỗi có tiếng Việt có dấu, xuống dòng, emoji và mã: `"So sánh AES-GCM với ChaCha20 🙂\n`[[ -f \"$f\" ]]`"`; so `redacted_text` với `body` bằng `cmp` trên byte thô | `allowed=true`; **0** byte khác (`TestRedactIdentityWhenClean`) |
| TC-P3-02-29 | AC4 | – | **G** `go test ./internal/privacy -run 'TestRedact' -v` | `ok` (có cả ca idempotent và ca sạch) |
| TC-P3-02-30 | AC5 | phiên `SID` của SVA; provider giả ghi payload | **S** lượt 1: gửi `"Bùi Thanh Khải học nhóm với em"`; lượt 2 **cùng phiên**: gửi `"bui thanh khai có nộp bài chưa"`; **S** đọc hai payload của provider giả | cả hai payload chứa **cùng** placeholder `[[SV_1]]`; **0** lần chuỗi `Bùi Thanh Khải` / `bui thanh khai`; số thứ tự **không đổi** giữa hai lượt |
| TC-P3-02-31 | AC5 | – | **S** một tin có đủ 5 loại: tên SVB, tên SVC, MSSV `20229002`, email, SĐT, CCCD | payload chứa `[[SV_1]]`, `[[SV_2]]`, `[[MSSV_1]]`, `[[EMAIL_1]]`, `[[SDT_1]]`, `[[CCCD_1]]`; số thứ tự theo **thứ tự xuất hiện đầu tiên**; `notice{masked:6}` trên SSE |
| TC-P3-02-32 | AC5 | – | **S** trong cùng phiên gửi lần lượt `"BÙI THANH KHẢI"`, `"bui thanh khai"`, `"Bùi Thanh Khải"` | cả ba payload dùng **một** placeholder duy nhất (khoá chuẩn hoá `vn_fold(lower)`) |
| TC-P3-02-33 | AC5 (tin sạch) | – | **S** gửi `"Thuật toán RSA dựa trên bài toán nào?"`; so payload với chuỗi gốc | payload chứa **đúng từng byte** chuỗi gốc; `masked_count=0`; **không** có sự kiện `notice{masked:…}` |
| TC-P3-02-34 | AC6 | – | **S** SVA gửi `"Em là Vũ Hoàng Giang, MSSV 20229001, cho em xem điểm danh"` | payload **0** lần `Vũ Hoàng Giang` và `20229001` (chủ phiên cũng bị che, không có ngoại lệ); system prompt có chữ `bạn`; câu trả lời hiện cho SVA **có** tên thật sau khi khôi phục |
| TC-P3-02-35 | AC7 | sau TC 30 | **R** `$RDS HGETALL ep:mask:$SID`; `$RDS TTL ep:mask:$SID`; gửi thêm một tin rồi đo TTL lần hai | HASH có khoá dạng `p:[[SV_1]]`, `r:<sha256>`, `n:SV`; TTL dương **≤ 86400**; sau lượt mới TTL được đặt lại (gần 86400 — "24 h kể từ lần dùng cuối") |
| TC-P3-02-36 | AC7 (không log) | log gateway + worker ở mức `debug` | **S** chạy trọn một lượt chat có PII rồi `grep -icE 'Vũ Hoàng Giang\|Bùi Thanh Khải\|bui thanh khai\|20229001\|20229002\|@edupilot.local\|0912345678\|p:\[\[' <log>` | **0** ở mọi mẫu; log chỉ có số đếm (`masked_count`), không giá trị thật, không ánh xạ, không nội dung tin nhắn |
| TC-P3-02-37 | AC7 (không phiên) | – | **R** đếm `$RDS --scan --pattern 'ep:mask:*' \| wc -l` → `n0`; **S** chạy một `precheck` cần nhúng (văn bản ≥ 20 ký tự, luật chưa quyết) và chờ worker trả lời một thread mới; **R** đếm lại | `n1 == n0` — việc không có phiên dùng ánh xạ trong bộ nhớ của yêu cầu, **không** tạo khoá Redis |
| TC-P3-02-38 | AC8 | provider giả trả nguyên văn placeholder nhận được | **S** SVA gõ `"bui thanh khai"` rồi đọc văn bản cuối trên SSE | văn bản hiện `Bùi Thanh Khải` (bản gốc **lần thấy đầu tiên**), không phải chuỗi không dấu đã gõ |
| TC-P3-02-39 | AC8 (dạng lỏng) | provider giả trả `"Bạn hỏi về [[ SV_1 ]] và [[sv_1]]"` | **S** đọc văn bản cuối | cả hai dạng được khôi phục thành tên thật; không còn ký tự `[[` nào thuộc dạng placeholder |
| TC-P3-02-40 | AC9 | – | **G** `go test ./internal/privacy -run 'TestUnmaskStream' -v`; đọc số tổ hợp in ra | `ok`; **≥ 50.000** tổ hợp; đọc tên ca xác nhận có: 0–5 placeholder, hai placeholder liền nhau, placeholder đầu/cuối chuỗi, `[` và `[[` thường, tiếng Việt có dấu, emoji; chia hai đoạn, chia ba đoạn, chia từng rune. Thiếu nhóm nào → FAIL |
| TC-P3-02-41 | AC9 (QC tự cắt) | provider giả phát theo kịch bản: chuỗi `"Xin chào [[SV_1]], điểm của bạn…"` cắt tại **mọi** vị trí rune (QC lặp qua từng vị trí) — `[CẦN: công tắc kịch bản của provider giả, ví dụ `FAKE_LLM_SCRIPT` — xem handoff]` | **S** với mỗi vị trí cắt: đọc **toàn bộ** khung SSE `token`, ghép lại và so với `Unmask(toàn chuỗi)`; đồng thời `grep -cE '\[\[ *(SV\|MSSV\|EMAIL\|SDT\|CCCD)'` trên **từng khung** | văn bản ghép **bằng đúng** bản không cắt ở mọi vị trí; **0** khung nào chứa nửa placeholder (`[[SV`, `[[SV_`, `[`) |
| TC-P3-02-42 | AC9 (nội dung hợp lệ) | provider giả trả `"Dùng `[[ -f \"$f\" ]]` để kiểm tệp, và `a[[i]]` là chỉ mục lồng"` | **S** đọc văn bản cuối | chuỗi đi qua **nguyên vẹn** từng byte: `[[ -f "$f" ]]` và `a[[i]]` vẫn còn, **không** bị thay bằng `bạn` (TLR-6) |
| TC-P3-02-43 | AC10 | provider giả trả 40 rune sau `[[` mà không đóng | **S** đo: (a) lô token không chứa `[` có tới SSE ngay trong cùng lần phát không (so dấu thời gian khung với dấu thời gian provider giả); (b) khi gặp `[[`, số rune bị giữ lại | (a) không bị giữ (chênh < 50 ms); (b) giữ **tối đa 32 rune**, rune thứ 33 trở đi được phát ra; cuối luồng phần giữ lại xử lý theo AC11 |
| TC-P3-02-44 | AC11 (mô hình bịa) | provider giả trả `"Chào [[SV_9]] nhé"` (không có trong ánh xạ) | **S** đọc văn bản cuối; **S** `grep -c 'placeholder sót' <log>`; đọc nội dung dòng log | người dùng thấy `"Chào bạn nhé"`; **đúng một** dòng `warn` chứa `count`, **không** kèm ánh xạ hay nội dung |
| TC-P3-02-45 | AC11 (ánh xạ hết hạn) | đang sinh | **R** `$RDS DEL ep:mask:$SID` giữa lúc stream chạy | mọi placeholder còn lại → `bạn`; không lỗi 5xx; **một** dòng `warn` |
| TC-P3-02-46 | AC11 (mở dở ở cuối) | provider giả kết thúc luồng ngay sau `"… gửi cho [[MSSV_"` | **S** đọc văn bản cuối sau `done` | phần mở dở **không** tới người dùng; được thay bằng `bạn`; `Flush()` không để sót |
| TC-P3-02-47 | AC11 (bất biến) | sau mỗi TC 38–46 | **S** quét **mọi** khung SSE, **mọi** thân JSON (`GET …/messages`) và `content` trong DB bằng `grep -cPi '\[\[\s*(SV\|MSSV\|EMAIL\|SDT\|CCCD)(_\d*)?'` | **0** ở mọi nguồn |
| TC-P3-02-48 | AC11 | – | **G** `go test ./internal/privacy -run 'TestScannerUnknownPlaceholder\|TestScannerUnclosedAtEnd\|TestScannerExpiredMapping\|TestScannerLeavesBashDoubleBracket' -v` | `ok` |
| TC-P3-02-49 | AC12 (Redis hỏng) | – | **S** chặn thao tác HASH `ep:mask:*` của gateway (`[CẦN: cách cô lập Redis chỉ cho khoá ánh xạ — ACL theo mẫu khoá hoặc công tắc thử; xem handoff]`), gửi một tin có PII; đọc payload provider giả + log | lượt chat **vẫn chạy**; payload **vẫn đã che** (0 tên / MSSV); có dòng `warn`; ánh xạ dùng trong bộ nhớ của yêu cầu |
| TC-P3-02-50 | AC12 (hỏng an toàn) | công tắc làm bước che vượt 50 ms hoặc panic (`[CẦN: công tắc thử — xem handoff]`) | **S** gửi một tin; đếm lời gọi tới provider giả; **D** `select status, error_kind from llm_audit order by created_at desc limit 1` | **0** lời gọi provider; người dùng thấy `"Chưa gửi được tin nhắn. Thử lại."` và chữ trong ô soạn giữ nguyên; `llm_audit` có đúng một dòng `status='error'`, `error_kind='MASK_FAILED'` |
| TC-P3-02-51 | AC12 | – | **G** `go test ./internal/privacy -run 'TestMaskRedisDownFallsBackInMemory\|TestMaskFailsClosed' -v` | `ok` |
| TC-P3-02-52 | AC13 (không chéo lớp) | một sinh viên thuộc **cả** C1 và C2; SVX chỉ ở C2 | **S** trong phiên chat của lớp C1, gõ tên SVX | tên SVX **không** bị che ở phiên C1 (không thuộc roster C1) — và tương ứng `precheck` ở C1 cũng cho qua (TC 19); từ điển lấy theo `course_id` của phiên |
| TC-P3-02-53 | AC13 (không chéo phiên) | hai phiên `S1`, `S2` của SVA | **S** ở `S1` tạo ánh xạ `[[SV_1]]`; cấu hình provider giả trả `[[SV_1]]` khi đang ở **`S2`** | ở `S2` placeholder bị coi là **sót** → thay bằng `bạn`; **không** lộ giá trị của phiên khác |
| TC-P3-02-54 | AC13 | – | **G** `go test ./internal/privacy -run 'TestMappingNotSharedAcrossSessions' -v` | `ok` |
| TC-P3-02-55 | AC14 (hiệu năng) | roster C1 60 tên (`[CẦN SEED: gieo thêm 30 sinh viên vào C1 bằng API Giảng viên để đạt 60]`) | **G** `go test ./internal/privacy -bench 'BenchmarkMask4k' -run '^$' -benchtime=200x`; **S** 100 lượt `precheck` tin 4.000 ký tự, tính p95 bằng `curl -w '%{time_total}'` | `ns/op` quy ra **≤ 5 ms**; p95 của `precheck` khi luật quyết định **≤ 150 ms** (SRS 8) |
| TC-P3-02-56 | AC14 (chống ReDoS) | – | **G** `go test ./internal/privacy -run 'TestDetectLinearTime' -v`; **S** `precheck` với `body` = 8.000 ký tự `a1a1…` rồi `(((…` (chạm `THREAD_BODY_MAX_CHARS`), đo `time_total` | `ok`; mỗi lượt API trả về < 1 s, **không** treo; thời gian tăng tuyến tính khi tăng độ dài (đo ở 1.000 / 4.000 / 8.000 ký tự). Chuỗi 100.000 ký tự chỉ kiểm được ở tầng đơn vị — xem Q-QC-P3-02-3 |
| TC-P3-02-57 | AC14 (cắt 20.000 ký tự) | – | **G** đọc ca kiểm `Detect` với văn bản > 20.000 ký tự: PII đặt ở ký tự thứ 25.000 | PII vẫn bị bắt (cửa sổ trượt quét phần còn lại, SRS 4.2.2), không bỏ sót vì cắt cứng |
| TC-P3-02-58 | phân quyền | SVX (chỉ ở C2) | **S** `j -X POST $Q/threads/precheck -H "$SVX" -d '{"body":"test"}'` | `403` (người ngoài lớp), không thân dữ liệu, **không** rò roster lớp C1 qua thông điệp lỗi |
| TC-P3-02-59 | phân quyền | ADMIN | **S** `j -X POST $Q/threads/precheck -H "$ADM" -d '{"body":"test"}'` | `403` (guard `Member` loại ADMIN — `SRS.md` 2, Q1) |
| TC-P3-02-60 | phân quyền | SVB | **S** `curl -s -o /dev/null -w '%{http_code}' -H "$SVB" $GW/api/v1/chat/sessions/$SID/messages` (phiên của SVA) | `404`; **R** `$RDS HGETALL ep:mask:$SID` không có đường nào lộ ra API (không route nào trả ánh xạ) |
| TC-P3-02-61 | phân quyền | TA, TEACHER | **S** `j -X POST $GW/api/v1/chat/sessions -H "$TA_" -d '{"course_id":"<C1>"}'`; lặp với `$TCH` | `403` cả hai (chat riêng chỉ STUDENT) |

## Nhánh lỗi (SRS 3.3 — phần thuộc `internal/privacy`)
| Tình huống | TC |
| --- | --- |
| Redis lỗi khi che → ánh xạ trong bộ nhớ, payload vẫn đã che | 49 |
| Che lỗi / quá 50 ms → `MASK_FAILED`, 0 lời gọi provider, `llm_audit status='error'` | 50 |
| Ánh xạ hết hạn giữa luồng | 45 |
| Mô hình bịa placeholder (`[[SV_9]]`) | 44 |
| Placeholder mở dở ở cuối luồng (`[[MSSV_`) | 46 |
| Token bị cắt giữa placeholder ở mọi vị trí | 40, 41 |
| Nội dung hợp lệ `[[ -f … ]]` / `a[[i]]` bị bộ quét ăn nhầm | 28, 42 |
| Chặn nhầm số giống PII (cổng, năm, IP, CVE, hex, kích thước khoá) | 07 |
| Chặn nhầm từ thường trùng tên (`anh`, `minh chứng`, `Hoa Kỳ`, `MIT`) | 16 |
| Sinh viên vừa vào lớp chưa bị che (cache cũ) | 21, 23 |
| Regex thảm hoạ / văn bản rất dài | 56, 57 |

## Phân quyền
| Ca | TC |
| --- | --- |
| Sinh viên ngoài lớp gọi `precheck` → 403 | 58 |
| ADMIN gọi `precheck` → 403 | 59 |
| SVB đọc phiên / ánh xạ của SVA → 404, không đường nào lộ ánh xạ | 60 |
| TA / TEACHER tạo chat riêng → 403 | 61 |
| Từ điển và ánh xạ không chéo lớp, không chéo phiên | 19, 52, 53 |
| Tên giảng viên / TA không bị che (Q8) | 24 |

## Script chạy được
- `docs/sprints/6/qc/scripts/p602-attacks.py` — bộ tấn công dữ liệu cá nhân do QC soạn (không lấy từ test của dev): MSSV (5 dạng), email (3), SĐT (4), CCCD (2), họ tên roster × {có dấu, không dấu, HOA, đảo, rút} cho **mọi** sinh viên lấy trực tiếp từ DB, cộng hai tập âm tính `negatives-number` và `negatives-name`. Gửi từng mẫu tới `POST /courses/{cid}/threads/precheck`, so `allowed` / `reasons[].type` / `redacted_text` với nhãn, in bảng sai sót. Chạy: `python3 docs/sprints/6/qc/scripts/p602-attacks.py --gw "$GW" --course "$C1" --token "$SVA_RAW" --set all`.
- Quét payload provider giả dùng chung với US-P3-03: `docs/sprints/6/qc/scripts/p603-payload-scan.sh` (TC 30–34).
- Test Go của dev chạy **thêm** ở TC 10, 18, 25, 29, 40, 48, 51, 54, 55, 56.

## Câu hỏi cho BA (Q-QC-…)
- **Q-QC-P3-02-1** — AC1 liệt kê `0912345678`, `+84 912 345 678`, `0912.345.678`, `09 1234 5678`, còn SRS 4.2.2 thêm điều kiện "đầu số di động 03/05/07/08/09". Số cố định (`0241234567`, `02838221234`) là PII hay không? QC đang chấm **không** (TC 09) — *chờ BA*.
- **Q-QC-P3-02-2** — AC5 nói số thứ tự "tăng theo thứ tự xuất hiện đầu tiên **trong phiên**", AC13 nói ánh xạ theo phiên. Khi `Session.ID()==""` (precheck, worker) số thứ tự đếm lại từ 1 cho mỗi yêu cầu, hay tiếp tục theo lớp? QC đang chấm "đếm lại từ 1 mỗi yêu cầu" — *chờ BA*.
- **Q-QC-P3-02-3** — AC14 yêu cầu chuỗi 100.000 ký tự ≤ 50 ms, nhưng `THREAD_BODY_MAX_CHARS` = 8000 và `CHAT_MAX_INPUT_CHARS` = 4000 nên không đường API nào đưa được 100.000 ký tự vào. Phép đo này chỉ kiểm được bằng test đơn vị của dev; QC có được chấm AC14 dựa trên test của dev không (trái luật "không lấy số từ test của dev")? — *chờ BA / PM*.
- **Q-QC-P3-02-4** — AC12 "che quá 50 ms" và AC7 "không log ánh xạ" cần một công tắc thử để QC ép lỗi từ ngoài. Spec không nêu công tắc nào. Dev có cung cấp biến môi trường thử (ví dụ `PRIVACY_MASK_TIMEOUT_MS`, `FAKE_LLM_SCRIPT`) không? Nếu không, TC 41, 49, 50 ghi "KHÔNG KIỂM ĐƯỢC" ở pha 2 (tính là FAIL) — *chờ BA / dev ghi vào handoff*.
- **Q-QC-P3-02-5** — AC2 cấm bắt `Nguyễn Văn Anh` khi roster chỉ có `Nguyễn Văn An`, nhưng khoá chuẩn hoá là `vn_fold(lower)` nên `Vũ Hoàng Giáng` sẽ gập thành `vu hoang giang` và **bị bắt** dù là người khác. Đây là hành vi mong muốn (chấp nhận chặn nhầm để không lọt) hay lỗi? — *chờ BA*.

## Lịch sử sửa TC
(chỉ sửa khi SPEC đổi: ghi ngày, TC nào, lý do, số proposal PM đã chấp nhận)
- 2026-10-10 — tạo mới theo `US.md` v1.2 / `SRS.md` v1.2 (APPROVED).

Tổng TC: 61
