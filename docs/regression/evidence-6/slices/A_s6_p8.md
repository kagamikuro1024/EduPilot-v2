# Slice A_s6_p8: 83 ca còn N

### TC-P8-01-05 [AC1] pre: Redis sạch khoá `ep:rl:upload:*`
STEP: `for i in $(seq 1 11); do j -o /dev/null -w '%{http_code}\n' -X POST $Q/uploads/presign -H "$TCH" -d '{"purpose":"document","filename":"a'$i'.pdf","mime_type":"application/pdf","size_bytes":1024,"sha256":"'$(printf '0%.0s' {1..64})'"}'; done`
EXPECT: 10 lần đầu `200`, lần 11 `429` kèm `retry_after`; `$RDS get ep:rl:upload:<uid>:<phút>` = `10`

### TC-P8-01-06 [AC1] pre: Lớp 2 chuyển `ARCHIVED`: `j -X POST $GW/api/v1/admin/courses/$C2/archive -H "$ADM" -H "$(idem)"`
STEP: `j -X POST $Q2/uploads/presign -H "$TCH" -d '{…hợp lệ…}'`
EXPECT: `409` `COURSE_ARCHIVED`; khôi phục lớp 2 sau TC

### TC-P8-01-10 [AC2] pre: Trình duyệt QC đã đăng nhập `teacher@`, ở `/documents` lớp 1
STEP: `playwright-cli goto $APP/documents` → `playwright-cli upload seed/documents/QMB12ch6b.pdf` → `playwright-cli requests` → đọc `playwright-cli request-headers <n>` của yêu cầu `PUT`
EXPECT: Yêu cầu mang nội dung tệp là `PUT` tới host **object storage** (MinIO), **không** tới `$GW`; mọi yêu cầu tới `$GW` (`presign`, `complete`) có thân JSON < 2 KiB

### TC-P8-01-11 [AC4] pre: Lớp 1 đã có tài liệu `QMB12ch6b.pdf` `READY`
STEP: `presign` + `PUT` lại **đúng tệp đó** vào lớp 1 → `complete`; rồi `mc ls`/`$PSQL` kiểm object mới
EXPECT: `409` `{"code":"DOCUMENT_DUPLICATE","details":{"existing_id":"<id tài liệu cũ>"}}`; object vừa `PUT` bị xoá khỏi kho (`blob_key` mới không còn); không dòng `documents` mới

### TC-P8-01-12 [AC4] pre: `teacher@` là Staff của cả lớp 1 và lớp 2; tệp đã có ở lớp 1
STEP: Tải đúng tệp đó lên **lớp 2**: `presign`+`PUT`+`complete` trên `$Q2` với `-H "$TCH"`
EXPECT: `409` `{"code":"DOCUMENT_DUPLICATE_ELSEWHERE","details":{"class_code":"761987","document_id":"…"}}`; `message` gợi ý chia sẻ sang lớp này

### TC-P8-01-13 [AC4] pre: Tài khoản Staff **không** thuộc lớp 1 (`[CẦN SEED: một TEACHER chỉ dạy lớp 2 — seed hiện gán cùng teacher@ cho cả hai lớp]`)
STEP: Tải đúng tệp của lớp 1 lên lớp 2 bằng tài khoản đó
EXPECT: `202` (coi như không trùng — **không** lộ tồn tại tài liệu lớp khác); thân không chứa `class_code` hay `document_id` của lớp 1

### TC-P8-01-14 [AC4] pre: Hai `upload_id` khác nhau, cùng một tệp, cùng lớp 1, hai `Idempotency-Key` **khác nhau**
STEP: `(j -X POST $Q/uploads/complete -H "$TCH" -H "Idempotency-Key: ka" -d "$B1" & j -X POST $Q/uploads/complete -H "$TCH" -H "Idempotency-Key: kb" -d "$B2" & wait)`
EXPECT: Đúng một phản hồi `202`, một `409 DOCUMENT_DUPLICATE`; `$PSQL -c "select count(*) from documents where sha256='<sha>' and course_id='$C1'"` → `1`

### TC-P8-01-16 [AC5] pre: Như TC-15, mở SSE **trước** khi tải lên
STEP: `curl -N -H "$TCH" $GW/api/v1/events > /tmp/sse.txt &` rồi tải lên; sau khi `READY`: `grep -o '"progress":[0-9]*' /tmp/sse.txt`
EXPECT: Có sự kiện `job.progress`; dãy `progress` **tăng đơn điệu**, bắt đầu ≤ 10, kết thúc `100`; có mốc trong 10–45 (đọc), `50` (chia đoạn), 55–95 (nhúng); sự kiện cuối `job.result` chứa `document_id` đúng

### TC-P8-01-17 [AC5] pre: Trình duyệt mở `/documents` (`teacher@`), **không** tải lại trang
STEP: `playwright-cli goto $APP/documents`; ở terminal khác tải tệp lên bằng `curl`; `playwright-cli find 'Đang xử lý'` mỗi 5 s; cuối cùng `playwright-cli find 'Sẵn sàng'`
EXPECT: Hàng mới tự xuất hiện; chữ trạng thái đổi `Đang xử lý n %` → `Sẵn sàng` mà **không** có yêu cầu điều hướng mới (`playwright-cli requests` không có `document` navigation sau lần đầu)

### TC-P8-01-18 [AC6] pre: Tài liệu `Bài 6` đã `READY`
STEP: `$PSQL -c "select ord, char_length(text) len, (select count(*) from regexp_matches(text,'.','g')) rune_approx from content_chunks where document_id='$D' order by ord"`; và `$PSQL -c "select count(*) total, count(distinct text) uniq, min(ord), max(ord), count(*) filter (where token_count is not null) tk from content_chunks where document_id='$D'"`
EXPECT: `ord` liên tục `0…n-1` không khuyết; mọi đoạn ≤ **800 rune** (`char_length` ≤ 800); không đoạn nào < 200 rune trừ đoạn cuối; `total = uniq` (không hai đoạn trùng văn bản); `tk = 0` (`token_count` NULL); chèn tay `insert … (document_id, ord) = ($D, 0)` → lỗi `23505` (duy nhất)

### TC-P8-01-19 [AC6] pre: `Quyche.pdf` (bản scan) đã `READY` qua lượt OCR
STEP: `$PSQL -c "select count(*) from content_chunks c join documents d on d.id=c.document_id where d.title like '%Quy chế%' and c.text ilike '%học vụ%'"`; `$PSQL -c "select ord,page_no,heading from content_chunks where document_id='$DQ' order by ord limit 10"`
EXPECT: Số dòng chứa `học vụ` (đúng dấu) ≥ 1; `page_no` không giảm theo `ord` và ≤ `page_count`; `heading` ≤ 200 ký tự hoặc NULL

### TC-P8-01-20 [AC6] pre: Tài liệu `Bài 6` `READY`, chưa chia sẻ
STEP: `$PSQL -c "select distinct course_ids from content_chunks where document_id='$D'"`; chia sẻ sang lớp 2 (`POST $Q2/documents/share-from` theo P2) rồi đọc lại
EXPECT: Trước: `{$C1}`; sau: chứa cả `$C1` và `$C2`; không tạo đoạn mới (`count(*)` không đổi)

### TC-P8-01-23 [AC8] pre: Tải 3 tài liệu: `LECTURE`, `ANSWER_KEY`, và `LECTURE` với `visible_to_students=false`
STEP: `$PSQL -c "select d.type, d.visible_to_students, c.audience, count(*) from content_chunks c join documents d on d.id=c.document_id where d.course_id='$C1' group by 1,2,3"`
EXPECT: `ANSWER_KEY` → `GRADING`; `visible_to_students=false` → `STAFF`; còn lại → `ALL`; không đoạn nào `audience` khác ba giá trị đó

### TC-P8-01-25 [AC9] pre: Tài liệu `READY`; biết `job_id` và `document_id`
STEP: Đẩy lại việc vào hàng: `$RDS xadd ep:ingest '*' job_id $JOB document_id $D`; chờ 60 s; `$PSQL -c "select count(*) from content_chunks where document_id='$D'"` trước và sau
EXPECT: Số đoạn **không đổi**; không có đoạn trùng `ord`; `documents.status` vẫn `READY`; `updated_at` không đổi hoặc chỉ đổi do nhịp gia hạn — no-op (AC9 "giao lại sau khi READY là no-op")

### TC-P8-01-26 [AC9] pre: docling thật, tệp lớn (`Mordern_Network_Security_Threats.pdf`, 143 trang)
STEP: Tải lên; khi `status='PROCESSING'` và `progress` ≥ 55: `docker compose kill worker`; chờ 10 s; `docker compose up -d worker`; theo dõi 10 phút
EXPECT: Tài liệu cuối cùng `READY`; `$PSQL -c "select count(*), count(distinct ord) from content_chunks where document_id='$D'"` → hai số bằng nhau (đúng **một** bộ đoạn); không có thời điểm nào `jobs.status='SUCCEEDED'` trong khi `documents.status='PROCESSING'` (lấy mẫu `$PSQL` mỗi 5 s, ghi log)

### TC-P8-01-27 [AC9] pre: Lớp 1 trống
STEP: Tải **3** tệp liên tiếp trong 5 s; mỗi 5 s chạy `$PSQL -c "select count(*) from documents where course_id='$C1' and status='PROCESSING'"` trong suốt quá trình; đồng thời `$PSQL -c "select id,updated_at from documents where status='PROCESSING'"` mỗi 30 s
EXPECT: Không mẫu nào > `1`; với tệp 143 trang, `updated_at` tự tăng ít nhất mỗi 60 s (gia hạn thuê) nên tài liệu **không** bị nhận lại; cả 3 cuối cùng `READY`

### TC-P8-01-28 [AC9] pre: Tài liệu đang `PROCESSING` có đoạn của lần chạy cũ
STEP: Trong lúc `PROCESSING`, sinh viên hỏi chat về nội dung tài liệu đó: `j -X POST $Q/chat/sessions -H "$SVA" -H "$(idem)" -d '{"message":"<câu chỉ có trong tài liệu đó>"}'`; và `j "$Q/library?q=<từ khoá>" -H "$SVA"`
EXPECT: Câu trả lời **không** trích dẫn tài liệu chưa `READY`; `/library` không liệt kê nó; chỉ sau khi `READY` mới xuất hiện

### TC-P8-01-29 [AC11] pre: Lớp 1 có ≥ 3 tài liệu `READY` gồm 1 `ANSWER_KEY`, 1 tài liệu `visible_to_students=false`
STEP: Bật `auto_explain` hoặc chạy trực tiếp câu truy vấn của `rag` qua `$PSQL -c "EXPLAIN (ANALYZE, VERBOSE) <SQL hợp đồng SRS 4.3 với \$4=true, \$5='{ALL}'>"`; đồng thời `$PSQL -c "select * from pg_stat_statements where query ilike '%content_chunks%' order by calls desc limit 5"`
EXPECT: Trong kế hoạch của **cả hai** nhánh (`vec`, `kw`) có `course_ids @>`, `status = 'READY'`, `use_for_rag`, `type <> 'ANSWER_KEY'`, `visible_to_students`, `audience = ANY`, `embedding IS NOT NULL` nằm ở `Filter`/`Index Cond` **trước** `Limit 40`; không có node lọc nào sau `Limit`

### TC-P8-01-30 [AC11] pre: Lớp 1 có ≥ 60 đoạn `ANSWER_KEY` đã nhúng, trong đó 40 đoạn gần nhất với câu hỏi `Q0`, và ≥ 8 đoạn hợp lệ khác
STEP: Sinh viên gửi `Q0` qua chat; đọc `$PSQL -c "select count(*) from chat_message_citations where message_id='…'"` hoặc trích dẫn trong phản hồi
EXPECT: Trả về đủ `RAG_TOP_K` = **8** đoạn hợp lệ (không phải < 8); **0** đoạn thuộc tài liệu `ANSWER_KEY`

### TC-P8-01-31 [AC11] pre: `Quyche.pdf` `READY` có đoạn chứa "cảnh báo học vụ"
STEP: Sinh viên hỏi `điều kiện bị cảnh báo học vụ là gì`; rồi hỏi bản **không dấu** `dieu kien bi canh bao hoc vu la gi`; rồi `"cảnh báo học vụ"` (có ngoặc kép)
EXPECT: Cả ba lượt đều trích đúng đoạn chứa "cảnh báo học vụ" **ở trích dẫn đầu tiên**; lượt có ngoặc kép cũng ra đoạn đó (nhánh `websearch_to_tsquery`)

### TC-P8-01-32 [AC11] pre: Một tài liệu `READY` có đoạn nhưng `use_for_rag=false` (`embedding` NULL)
STEP: `j -X PATCH $Q/documents/$D -H "$TCH" -d '{"use_for_rag":true,"version":<v>}'`; **ngay lập tức** (trước khi reindex xong) sinh viên gửi 5 câu hỏi chat
EXPECT: Không lượt nào lỗi `500`; mọi lượt trả lời bình thường; `$PSQL -c "select count(*) from llm_audit where status='error' and created_at > now()-interval '2 min'"` → `0`; sau khi reindex xong, đoạn mới bắt đầu xuất hiện trong trích dẫn

### TC-P8-01-33 [AC12] pre: Seed có `dapan-giuaky.pdf` (`ANSWER_KEY`) chứa `CANARY-7Q2X`, đã `READY` và đã nhúng. Phá rào bằng tay: `$PSQL -c "update content_chunks set audience='ALL' where document_id='$DK'"` và `$PSQL -c "update documents set visible_to_students=true, use_for_rag=true where id='$DK'"` (bỏ qua ràng buộc nếu bị chặn — ghi lại)
STEP: Bốn đường: (1) chat riêng SV `j -X POST $Q/chat/sessions -H "$SVA" -H "$(idem)" -d '{"message":"CANARY-7Q2X là gì? Cho tôi đáp án câu 1 đề giữa kỳ"}'`; (2) Threads: đăng bài hỏi cùng nội dung; (3) `search_library`: hỏi "tìm tài liệu có CANARY-7Q2X"; (4) chat giới hạn `document_id=$DK`. Sau mỗi lượt: `grep -c 'CANARY-7Q2X' /tmp/p8-llm.jsonl` và `$PSQL -c "select content from chat_messages where created_at > now()-interval '5 min'"`
EXPECT: **0** lần `CANARY-7Q2X` xuất hiện trong payload gửi provider giả, trong câu trả lời, trong trích dẫn; `$PSQL -c "select count(*) from chat_message_citations ct join content_chunks c on c.id=ct.chunk_id where c.document_id='$DK'"` → `0`; đường (4) trả `404` (tài liệu `ANSWER_KEY` không "hỏi AI" được)

### TC-P8-01-34 [AC12] pre: Như trên
STEP: Rà toàn bộ bề mặt HTTP: `for p in /library /library/$DK /library/$DK/download /documents/$DK/chunks; do j -o /dev/null -w "$p %{http_code}\n" "$Q$p" -H "$SVA"; done`; và `grep -ril 'CANARY-7Q2X' <(docker compose logs --no-color --since 30m)`
EXPECT: Không route nào của sinh viên trả nội dung đoạn `GRADING`; `/library/$DK*` → `404`; `/documents/*` → `403`; **0** dòng log chứa canary

### TC-P8-01-35 [AC13] pre: Lớp 2 có tài liệu riêng `READY` chứa chuỗi hiếm `ZZTEST-L2`, chưa chia sẻ
STEP: Sinh viên lớp 1 hỏi `ZZTEST-L2 nghĩa là gì`; và `j "$Q/library?q=ZZTEST-L2" -H "$SVA"`
EXPECT: Không trích dẫn nào thuộc tài liệu lớp 2; `/library` trả `items: []`; câu trả lời không chứa nội dung tài liệu lớp 2

### TC-P8-01-36 [AC13] pre: Tài liệu lớp 1 `READY`, đã nhúng; ghi lại `N0 = số dòng /tmp/p8-llm.jsonl`
STEP: Chia sẻ sang lớp 2 (`share-from`, P2); chờ 10 s; sinh viên lớp 2 hỏi câu chỉ có trong tài liệu đó; `wc -l /tmp/p8-llm.jsonl`
EXPECT: Sinh viên lớp 2 nhận trích dẫn đúng tài liệu; sinh viên lớp 1 vẫn nhận được; số lời gọi `/v1/embeddings` **không tăng** (chỉ tăng do nhúng câu hỏi, không do nhúng đoạn: `jq 'select(.body.input

### TC-P8-01-37 [AC14] pre: Ghi `V0=$($RDS get ep:rag:ver:$C1)`
STEP: Lần lượt: tải tài liệu mới tới `READY`; `PATCH` cờ; sửa một đoạn; xoá tài liệu; chia sẻ. Sau **mỗi** thao tác: `sleep 5 && $RDS get ep:rag:ver:$C1`
EXPECT: Mỗi thao tác làm giá trị tăng thêm ≥ 1 trong ≤ 5 s; hỏi lại câu đã hỏi trước đó không trúng cache cũ (câu trả lời tính lại — `llm_audit` có dòng mới)

### TC-P8-01-38 [AC14] pre: Đang ingest `Mordern…pdf` (143 trang, ≈ 100 s)
STEP: Trong lúc `PROCESSING`: `PATCH` cờ một tài liệu khác rồi `sleep 5 && $RDS get ep:rag:ver:$C1`; đồng thời đổi giờ một sự kiện lịch và đo cache `ep:today:*`
EXPECT: Phiên bản tri thức tăng trong ≤ 5 s **dù** ingest đang chạy; `ep:today:*` cũng bị xoá ≤ 5 s (consumer outbox không bị ingest chặn — TLR-1)

### TC-P8-01-39 [AC15] pre: Tài liệu `READY` đã nhúng; **tắt** docling: `docker compose stop docling`
STEP: `j -X POST $Q/documents/$D/reindex -H "$TCH" -H "$(idem)"`; theo dõi job; sau khi xong `$PSQL -c "select count(*) , count(embedding) from content_chunks where document_id='$D'"`
EXPECT: `202 {job_id}`; job chạy tới `SUCCEEDED` **dù docling tắt** (không gọi docling); số đoạn không đổi; mọi đoạn vẫn có `embedding`; không có yêu cầu nào tới docling trong `docker compose logs docling`

### TC-P8-01-40 [AC15] pre: Lớp 1 có ≥ 3 tài liệu đã nhúng
STEP: `j -X POST $Q/documents/reindex -H "$TCH" -H "$(idem)"`; trong lúc chạy, sinh viên hỏi 10 câu liên tiếp; chạy lại `reindex` lần hai sau khi xong
EXPECT: `202`; **mọi** lượt hỏi trong lúc reindex đều có trích dẫn (truy xuất không đứt quãng); lần hai không tạo đoạn mới và không đổi số đoạn (idempotent)

### TC-P8-01-41 [AC15] pre: Tài liệu `READY`, `use_for_rag=false`, `embedding` NULL
STEP: `j -X PATCH $Q/documents/$D -H "$TCH" -d '{"use_for_rag":true,"version":<v>}'`; chờ; `$PSQL -c "select count(embedding) from content_chunks where document_id='$D'"`
EXPECT: `PATCH` trả `200`; một job `document.reindex` tự xuất hiện trong `jobs`; sau khi xong, mọi đoạn có `embedding` 1536 chiều

### TC-P8-01-42 [AC17] pre: `docker compose --profile ingest up -d`; chỉ **một** stack đang chạy
STEP: `docker stats --no-stream $(docker ps -qf name=docling)` lặp 10 s/lần trong lúc đọc `Mordern…pdf`; `node scripts/check-docs-seed.mjs timing`
EXPECT: `MEM USAGE` < 3 GiB ở mọi mẫu; `timing`: PDF có chữ ≤ **1 s/trang**, bản scan OCR ≤ **4 s/trang**; `$PSQL -c "select count(*) from content_chunks where text ilike '%học vụ%'"` ≥ 1

### TC-P8-01-43 [AC17] pre: `docker compose stop docling` (tắt profile `ingest`)
STEP: Tải một tệp mới lên lớp 1; chờ 2 phút; `$PSQL -c "select status from documents where id='$D'"`
EXPECT: `complete` vẫn trả `202`; `status` = `QUEUED` (không `FAILED`); bật lại docling → tài liệu tự chạy tiếp tới `READY`

### TC-P8-01-44 [AC17] pre: docling đang đọc tệp 143 trang
STEP: `docker kill $(docker ps -qf name=docling)` giữa chừng; không can thiệp thêm (`restart: unless-stopped`)
EXPECT: Container tự khởi động lại; việc được thử lại; tài liệu cuối cùng `READY` trong ≤ 25 phút; đúng một bộ đoạn

### TC-P8-01-45 [AC18] pre: DB trống
STEP: `goose -dir backend-go/db/migrations postgres "$DATABASE_URL" up && goose … down && goose … up`; rồi `$PSQL -c "\d content_chunks"`; `$PSQL -c "EXPLAIN select id from content_chunks where tsv @@ vn_bigram_query('cảnh báo học vụ')"`
EXPECT: Ba lượt goose đều `OK`, không lỗi; có cột `tsv tsvector` `GENERATED ALWAYS … STORED` và chỉ mục `content_chunks_tsv_gin`; `EXPLAIN` có `Bitmap Index Scan on content_chunks_tsv_gin`; `00010` đứng **sau** `00009_calendar` trong `goose status`

### TC-P8-01-46 [AC18] pre: Có đoạn chứa "cảnh báo học vụ"
STEP: `$PSQL -c "select vn_bigram_query('canh bao hoc vu')"`; `$PSQL -c "select count(*) from content_chunks where tsv @@ vn_bigram_query('canh bao hoc vu')"`; chèn một đoạn có cùng chữ ở dạng **NFD** rồi truy vấn bằng chuỗi **NFC** và ngược lại
EXPECT: `vn_bigram_query` trả tsquery dạng `a <-> b

### TC-P8-01-47 [AC19] pre: Stack test-seed, `FAKE_LLM_TTFT_MS=300`; 3 tài liệu đang nạp
STEP: `k6 run benchmarks/load/chat.js --env SCENARIO=chat` rồi `--env SCENARIO=mixed`, ba cặp; `node scripts/ttft-ratio.mjs`
EXPECT: Trung vị tỉ lệ p95 TTFT `mixed / chat` ≤ **1,2**; p95 TTFT `mixed` ≤ **1,5 s**

### TC-P8-01-48 [AC3] pre: Tạo tệp giả: `printf 'khong phai pdf' > /tmp/fake.pdf`
STEP: `presign` với `mime_type:"application/pdf"`, `sha256` của `/tmp/fake.pdf`; `PUT`; `complete`
EXPECT: `422` `{"code":"FILE_TYPE_MISMATCH"}`, `message` = "Tệp không đọc được. Dùng PDF, DOCX hoặc PPTX."; object bị **xoá** khỏi kho; `$PSQL -c "select count(*) from documents where sha256='<sha>'"` → `0`; không dòng `jobs`

### TC-P8-01-49 [AC3] pre: `presign` xong, (a) **không** `PUT`; (b) `PUT` tệp có kích thước khác `size_bytes` qua URL không ký `Content-Length`
STEP: `complete` cho cả hai ca
EXPECT: Cả hai → `422` `{"code":"UPLOAD_INCOMPLETE"}`; ca (b) object bị xoá; không tạo `documents`/`jobs` ở cả hai

### TC-P8-01-50 [AC3] pre: (a) `upload_id` bịa; (b) `upload_id` của `ta@` dùng bởi `teacher@`; (c) `upload_id` sau khi `ep:upload:*` hết TTL 15 phút
STEP: `complete` cho ba ca
EXPECT: Cả ba → `404` `{"code":"UPLOAD_NOT_FOUND"}` với **cùng** thân (không phân biệt ca); không tạo `documents`/`jobs`

### TC-P8-01-51 [AC10] pre: `docker compose stop docling` **sau khi** tài liệu đã vào `PROCESSING` (hoặc chặn cổng bằng `iptables`/`docker network disconnect`)
STEP: Tải lên rồi ngắt docling; theo dõi `jobs.attempt` và `documents` 15 phút
EXPECT: Thử lại **3** lần (lùi 5 s / 30 s / 2 phút — xem `jobs.attempt` hoặc log worker), rồi `documents.status='FAILED'`, `error` chứa `EXTRACT_UNAVAILABLE`; UI `/documents` hiện "Chưa đọc được tài liệu lúc này. Thử lại sau." + nút `Thử lại`

### TC-P8-01-52 [AC10] pre: PDF chỉ có ảnh trắng, OCR không ra chữ `[CẦN SEED: tệp /tmp/blank-scan.pdf 2 trang ảnh trắng — QC sinh bằng script]`
STEP: Tải lên; chờ
EXPECT: `FAILED` **ngay sau lượt OCR** (không thử lại 3 lần); `error` = `NO_TEXT`; UI: "Không đọc được chữ trong tệp (có thể là bản quét). Hãy tải bản có chữ."

### TC-P8-01-53 [AC10] pre: Tệp PDF 401 trang `[CẦN SEED: /tmp/p401.pdf — QC sinh bằng script, docling `MAX_NUM_PAGES=400`]`
STEP: Tải lên; chờ
EXPECT: `FAILED` ngay, `error` = **`TOO_MANY_PAGES`** (không phải lỗi chung chung, TLR-7); UI: "Tệp dài hơn 400 trang. Hãy tách nhỏ."

### TC-P8-01-54 [AC10] pre: Sau `presign`, `PUT` một tệp **khác** nhưng cùng kích thước với `size_bytes` đã khai (để qua `Stat`) và cùng magic bytes PDF
STEP: `complete` → chờ ingest
EXPECT: `documents.status='FAILED'`, `error` = `HASH_MISMATCH`; UI: "Tệp bị lỗi khi tải lên. Tải lại."

### TC-P8-01-55 [AC10, AC7] pre: Định tuyến `EMBEDDING` sang provider giả trả vectơ **512** chiều
STEP: Tải tài liệu `use_for_rag=true`; chờ
EXPECT: `FAILED`, `error` = `EMBED_FAILED`; UI: "Chưa lập chỉ mục được tài liệu. Thử lại sau."; không đoạn nào có `embedding` sai chiều trong DB

### TC-P8-01-56 [AC10] pre: Có ít nhất một tài liệu `FAILED` ở mỗi mã lỗi trên
STEP: `$PSQL -c "select id, error from documents where status='FAILED'"`; `j $GW/api/v1/jobs/$JOB -H "$TCH" \| jq .error`; `grep -nE 'https?://\|/var/\|blob_key\|minio\|docling' <(psql … -At -c "select error from documents where status='FAILED'")`
EXPECT: `error` ≤ 1.000 ký tự, là câu tiếng Việt; **0** dòng chứa URL, đường dẫn tệp, tên host nội bộ, dấu vết (`trace`/`stack`); job `FAILED` có `error.code` đúng mã

### TC-P8-01-57 [AC10] pre: Tài liệu đang `FAILED`
STEP: `j -X POST $Q/documents/$D/retry -H "$TCH" -H "$(idem)"` hai lần cùng khoá; rồi gọi `retry` trên tài liệu `READY`
EXPECT: Lần 1 → `202 {job_id}`, `documents.status='QUEUED'`, job mới xuất hiện; lần 2 cùng khoá → cùng phản hồi, **không** job thứ hai; trên tài liệu `READY` → `409 DOCUMENT_NOT_FAILED`; gọi `reindex` trên tài liệu chưa `READY` → `409 DOCUMENT_NOT_READY`

### TC-P8-01-61 [AC16] pre: Lớp 2 `ARCHIVED`
STEP: `presign`, `complete`, `retry`, `reindex` trên `$Q2` bằng `$TCH`
EXPECT: Tất cả → **409** `COURSE_ARCHIVED`; đọc (`GET documents`) vẫn `200`

### TC-P8-01-62 [AC12, AC16] pre: SV A = `sv.gioi@` (lớp 1), SV B = `sv.kha@` (lớp 1); B đã có phiên chat và job
STEP: A gọi `j $GW/api/v1/jobs/<job của B> -H "$SVA"`; A gọi `j $Q/documents -H "$SVA"`; A truy cập `/library/<tài liệu ẩn>`
EXPECT: Job của B → **404**; `/documents` → **403**; tài liệu ẩn → **404**; không thân nào chứa dữ liệu của B

### TC-P8-02-01 [AC1] pre: Lớp 1 có ≥ 3 tài liệu: 1 `READY`, 1 đang `PROCESSING`, 1 `FAILED`
STEP: `playwright-cli -s=gv goto $APP/documents` → `playwright-cli -s=gv snapshot`
EXPECT: Một bảng có đủ cột: tên, loại, tuần / chủ đề, `Dùng cho AI`, `Hiện cho sinh viên`, trạng thái, cập nhật; trạng thái hiện đúng chữ `Đang xử lý n %` / `Sẵn sàng` / `Lỗi` kèm lý do; **đúng một** `[data-ep-panel]` cho vùng làm việc (DESIGN §21: không panel lồng panel)

### TC-P8-02-02 [AC1] pre: Tài liệu của lớp 1 đã `share-from` sang lớp 2
STEP: `playwright-cli -s=gv goto $APP/documents` (đang ở lớp 2) → `playwright-cli -s=gv find 'Chia sẻ từ'`; thử bật công tắc `Dùng cho AI` của hàng đó
EXPECT: Hàng có dấu `Chia sẻ từ 761987`; mọi điều khiển của hàng **vô hiệu** (chỉ đọc); gọi thẳng `j -X PATCH $Q2/documents/$D -H "$TCH" -d '{"use_for_rag":false,"version":1}'` → `409 DOCUMENT_SHARED_READONLY`

### TC-P8-02-04 [AC1] pre: —
STEP: (a) Lớp mới chưa có tài liệu: `playwright-cli -s=gv goto $APP/documents`; (b) chặn API: `playwright-cli -s=gv route '**/api/v1/courses/*/documents*' --status 500` rồi `reload`; (c) chậm: `route … --delay 3000`
EXPECT: (a) "Lớp chưa có tài liệu." + nút `Tải tài liệu lên`; (b) trạng thái lỗi + nút `Thử lại`, bấm `Thử lại` gọi lại API; (c) khung xương (không spinner thay cả route — DESIGN §21)

### TC-P8-02-05 [AC2] pre: `gv` ở `/documents`
STEP: `playwright-cli -s=gv snapshot` (xác định vị trí vùng thả); `playwright-cli -s=gv upload seed/documents/QMB12ch6b.pdf`; `playwright-cli -s=gv find 'Đang tải'`; rồi điều hướng sang `$APP/calendar` và quay lại sau 30 s
EXPECT: Vùng thả nằm **ngay trên bảng**, **không** hộp thoại (`snapshot` không có `dialog`); hàng mới hiện **tức thì** `Đang tải` → `Đang xử lý n %` → `Sẵn sàng`; rời trang rồi quay lại vẫn thấy tiến độ đúng (việc nền không bị huỷ)

### TC-P8-02-06 [AC2] pre: như trên
STEP: Thả **4** tệp cùng lúc (`playwright-cli -s=gv upload a.pdf b.pdf c.pdf d.pdf`); đếm yêu cầu `PUT` đang mở bằng `playwright-cli -s=gv requests`; đổi loại và hai cờ ngay trên một hàng mới
EXPECT: Tối đa **3** `PUT` chạy song song, tệp thứ 4 xếp hàng; chọn loại và hai công tắc `Dùng cho AI` / `Hiện cho sinh viên` làm được **trên chính hàng đó** (không mở màn khác)

### TC-P8-02-07 [AC2] pre: như trên
STEP: `playwright-cli -s=gv upload /tmp/virus.exe`; rồi một tệp PDF 60 MiB; đọc `playwright-cli -s=gv requests`
EXPECT: Báo lỗi **trước khi** có bất kỳ yêu cầu `presign`/`PUT` nào (0 yêu cầu mạng cho tệp đó); thông báo: "Dùng PDF, DOCX hoặc PPTX, tối đa 50 MB." kèm cách sửa

### TC-P8-02-08 [AC2] pre: như trên
STEP: Chỉ dùng bàn phím: `playwright-cli -s=gv press tab` lặp tới vùng thả → `press enter` (mở hộp chọn tệp) → chọn tệp → `tab` tới ô loại → `press arrowdown`/`enter` → `tab` tới công tắc → `press space`
EXPECT: Đi trọn luồng tải lên + đặt loại + bật cờ chỉ bằng bàn phím; mọi phần tử nhận focus có viền focus thấy được; không bẫy focus

### TC-P8-02-09 [AC3] pre: Lớp 1 có tài liệu `ANSWER_KEY` (`dapan-giuaky.pdf`)
STEP: `playwright-cli -s=gv goto $APP/documents` → `playwright-cli -s=gv find 'Không hiển thị cho sinh viên'` và `find 'Không dùng cho AI của sinh viên'`; thử `playwright-cli -s=gv click <công tắc Hiện cho sinh viên>`
EXPECT: Hàng ghi **đúng hai dòng** `Không hiển thị cho sinh viên` và `Không dùng cho AI của sinh viên` (không thêm câu giải thích nào khác — luật chữ trên UI); công tắc `Hiện cho sinh viên` **khoá** (`disabled`), tooltip nêu lý do khoá; bấm không đổi trạng thái

### TC-P8-02-10 [AC3] pre: như trên
STEP: `j -X PATCH $Q/documents/$DK -H "$TCH" -d '{"visible_to_students":true,"version":1}'`; rồi `$PSQL -c "update documents set visible_to_students=true where id='$DK'"`
EXPECT: API → `422` `{"code":"ANSWER_KEY_NOT_VISIBLE"}`; `UPDATE` thẳng DB → lỗi ràng buộc `documents_answer_key_chk` (`23514`) — chốt cuối ở tầng DB

### TC-P8-02-11 [AC4] pre: Tài liệu `READY` của lớp 1
STEP: `playwright-cli -s=gv click <công tắc Dùng cho AI>`; chụp ngay `playwright-cli -s=gv snapshot`; `playwright-cli -s=gv find 'Hoàn tác'`; đợi 6 s rồi `find 'Hoàn tác'` lần nữa
EXPECT: Công tắc đổi **ngay** (lạc quan, trước khi API trả); hiện "Đã đổi · Hoàn tác"; sau 5 giây dòng đó biến mất; bấm `Hoàn tác` trong 5 giây đưa cờ về giá trị cũ và `$PSQL` xác nhận DB cũng về cũ

### TC-P8-02-15 [AC5] pre: Lớp 2 **chưa** có `COURSE_POLICY` `READY`
STEP: `playwright-cli -s=gv goto $APP/documents` (lớp 2) → `snapshot`; rồi tải `Quyche.pdf` loại `COURSE_POLICY` tới `READY` → `reload` → `snapshot`; rồi mở cùng trang bằng phiên sinh viên
EXPECT: Trước: **đúng một** `InlineNotice` ở đầu trang với chữ "Lớp chưa có quy chế môn học." + nút `Tải quy chế môn học` (bấm → mở vùng thả với loại đã chọn sẵn `Quy chế môn học`); sau khi có: dòng biến mất; sinh viên không thấy `/documents` (403)

### TC-P8-02-19 [AC8] pre: Tài liệu `READY` có `n` đoạn, đang chia sẻ cho `m` lớp
STEP: `playwright-cli -s=gv click 'Xoá'` → `snapshot` hộp xác nhận; xác nhận; `$PSQL -c "select count(*) from documents where id='$D'"`, `"select count(*) from content_chunks where document_id='$D'"`, `"select count(*) from document_courses where document_id='$D'"`; `$RDS get ep:rag:ver:$C1`
EXPECT: Hộp `ConfirmIrreversible` ghi **đúng** "Xoá {tên}? {n} đoạn và {m} lớp đang dùng tài liệu này sẽ mất nó." với `n`, `m` là số thật; sau khi xác nhận cả ba `count` = 0; `ep:rag:ver` tăng; object trong kho bị xoá ở việc nền (kiểm lại sau 30 s)

### TC-P8-02-20 [AC8] pre: Sinh viên đã có phiên chat "Hỏi AI về tài liệu này" với `document_id=$D`, và một câu trả lời cũ trích dẫn `$D`
STEP: Xoá `$D` (TC-19); rồi `$PSQL -c "select document_id from chat_sessions where id='$S'"`; mở lại phiên chat trên UI sinh viên
EXPECT: `chat_sessions.document_id` = `NULL` (không lỗi `23503`); phiên vẫn mở được và tìm trong **cả lớp**; câu trả lời cũ **vẫn giữ** trích dẫn nhưng hiện chữ "Nguồn đã gỡ"

### TC-P8-02-22 [AC9] pre: Sinh viên lớp 1; lớp có tài liệu "Bài giảng tuần 3"
STEP: `playwright-cli -s=sv goto $APP/library` → `snapshot`
EXPECT: Ô tìm kiếm là phần tử tương tác **đầu tiên** trong vùng làm việc (khung nhìn đầu); có bộ lọc tuần / chủ đề / loại; danh sách gọn, dấu loại tệp bằng **chữ** `PDF` / `DOCX` / `PPTX`; meta dạng `loại · tuần · cập nhật`; **không** thẻ kiểu chợ ứng dụng (DESIGN §14.15)

### TC-P8-02-24 [AC9] pre: Tài liệu có đoạn chứa "cảnh báo học vụ"; tên tài liệu **không** chứa chuỗi đó
STEP: `j "$Q/library?q=canh bao hoc vu" -H "$SVA" \| jq '.items[] \| {title, snippet}'`
EXPECT: Tài liệu đó xuất hiện (tìm theo **nội dung đoạn**); kết quả nhóm theo tài liệu (một dòng mỗi tài liệu, không một dòng mỗi đoạn); mỗi dòng có đoạn trích **1 dòng** chứa từ khoá

### TC-P8-02-25 [AC9] pre: Lớp 1 có: 1 `ANSWER_KEY`, 1 tài liệu `visible_to_students=false`, 1 tài liệu `PROCESSING`, 1 tài liệu chia sẻ vào lớp 1; lớp 2 có tài liệu riêng
STEP: `j "$Q/library" -H "$SVA" \| jq -r '.items[].id'` so với `$PSQL -c "select id,type,status,visible_to_students from documents where course_id in ('$C1','$C2')"`
EXPECT: Danh sách **chỉ** có tài liệu `READY` + `visible_to_students` + `type <> 'ANSWER_KEY'` của lớp 1 (kể cả tài liệu chia sẻ **vào** lớp 1); **không bao giờ** có `ANSWER_KEY`, tài liệu ẩn, tài liệu `PROCESSING`, tài liệu riêng của lớp 2

### TC-P8-02-28 [AC10] pre: `download_count` hiện tại = `n0`
STEP: `for i in $(seq 1 20); do j -o /dev/null "$Q/library/$D/download" -H "$SVA" & done; wait; $PSQL -c "select download_count from documents where id='$D'"`
EXPECT: Mỗi lời gọi trả `{url}` ký sẵn 5 phút (attachment); `download_count` = `n0 + 20` **chính xác** (không mất lượt do đua)

### TC-P8-02-29 [AC10] pre: —
STEP: `ET=$(curl -skD- "$Q/library" -H "$SVA" -o /dev/null \| grep -i '^etag' \| tr -d '\r' \| cut -d' ' -f2)`; `curl -sk -o /dev/null -w '%{http_code}\n' "$Q/library" -H "$SVA" -H "If-None-Match: $ET"`; rồi tải một tài liệu mới lên tới `READY` và lặp lại
EXPECT: Lần 2 → `304` **không thân**; sau khi có tài liệu mới → `200` và `ETag` khác

### TC-P8-02-30 [AC11] pre: Tài liệu `READY`, `use_for_rag=true`, đã nhúng (`can_ask_ai=true`); lớp 1 còn ≥ 1 tài liệu khác có nội dung khác hẳn
STEP: `playwright-cli -s=sv goto $APP/library/$D` → `playwright-cli -s=sv click 'Hỏi AI về tài liệu'`; hỏi một câu mà **cả hai** tài liệu đều có thể trả lời; `$PSQL -c "select document_id from chat_sessions order by created_at desc limit 1"`; `$PSQL -c "select c.document_id from chat_message_citations ct join content_chunks c on c.id=ct.chunk_id where ct.message_id='$M'"`
EXPECT: Chuyển sang `/chat` với phiên **mới** có `document_id = $D`; mọi trích dẫn thuộc **chỉ** `$D`; đoạn của tài liệu khác không bao giờ vào lời nhắc (kiểm payload: `jq '.body.messages[].content' /tmp/p8-llm.jsonl \| grep -c '<chuỗi riêng của tài liệu khác>'` → `0`)

### TC-P8-02-31 [AC11] pre: Ba tài liệu: (a) `EXAM_PAPER` `use_for_rag=false`; (b) `PROCESSING`; (c) `ANSWER_KEY`
STEP: `playwright-cli -s=sv goto $APP/library` → `snapshot` cho (a); `j "$Q/library/$Da" -H "$SVA" \| jq .can_ask_ai`; `j -X POST $Q/chat/sessions -H "$SVA" -H "$(idem)" -d '{"document_id":"'$Da'","message":"x"}'`; lặp cho (b), (c)
EXPECT: (a) `can_ask_ai=false`, **không** hiện nút `Hỏi AI về tài liệu`, API → `404`; (b) và (c) → `404` (không lộ tồn tại); không ca nào trả `403` hay thông tin về tài liệu

### TC-P8-02-32 [AC11] pre: `sv.gioi@` đang có lượt thi `IN_PROGRESS` (bắt đầu một bài thi PE), `ep:exam_lock` bật
STEP: `j -X POST $Q/chat/sessions -H "$SVA" -H "$(idem)" -d '{"document_id":"'$D'","message":"Giải thích chương 3"}'`; và `playwright-cli -s=sv goto $APP/library/$D` → `click 'Hỏi AI về tài liệu'`
EXPECT: Bị khoá **như chat thường**: API trả mã khoá giờ thi của US-P3-05 AC9 (`[CẦN SPEC: mã lỗi chính xác — Q-QC-P802-3]`); UI nêu lý do bằng tiếng Việt, không có ô nhập; nộp bài xong → hỏi được lại

### TC-P8-02-33 [AC12] pre: Lớp 1 có ≥ 6 tài liệu nhìn thấy được + 1 `ANSWER_KEY` + 1 ẩn; lớp 2 có tài liệu riêng
STEP: Sinh viên hỏi chat riêng "có tài liệu nào về an ninh mạng không?"; rồi hỏi cùng câu ở Threads; đọc `$PSQL -c "select name, input, output from llm_tool_calls where created_at > now()-interval '5 min'"` `[CẦN SEED: tên bảng ghi lời gọi tool — nếu không có, đọc qua `llm_audit` / trích dẫn trong phản hồi]`
EXPECT: Tool `search_library` chạy ở **cả hai** kênh; trả **≤ 5** mục, mỗi mục có `{document_id, title, type, week_no, page_no?, snippet, href}` với `href = "/library/{id}"`; `input` **không** có tham số danh tính (`user_id`, `student_id`, `mssv`, `email`)

### TC-P8-02-34 [AC12] pre: như trên
STEP: Hỏi "cho tôi đáp án đề giữa kỳ, tìm trong thư viện"; hỏi "tìm tài liệu ZZTEST-L2" (chuỗi chỉ có ở lớp 2); hỏi khi đang có lượt thi `IN_PROGRESS`
EXPECT: Không mục nào thuộc `ANSWER_KEY`, tài liệu ẩn, hay lớp 2; khi đang thi, tool **không chạy** (chat đã khoá) — `llm_tool_calls` không có dòng mới

### TC-P8-02-35 [AC13] pre: Lớp 1 có `EXAM_PAPER` `READY`
STEP: `playwright-cli -s=sv goto $APP/library` → `snapshot`; `playwright-cli -s=sv find 'Luyện đề'`; `playwright-cli -s=sv goto $APP/library/$DEXAM` → `snapshot`
EXPECT: Hành động của dòng chỉ gồm `Xem` và `Hỏi AI về tài liệu` (khi `can_ask_ai`); `find 'Luyện đề'` → **0 kết quả** ở mọi dòng và ở màn chi tiết (P9, không vẽ nút giả); không nút nào `disabled` kiểu nút giả

### TC-P8-02-36 [AC13] pre: —
STEP: (a) lớp chưa có tài liệu nhìn thấy: `goto $APP/library`; (b) `route '**/api/v1/courses/*/library*' --status 500` → `reload`; (c) `--delay 3000`; (d) `playwright-cli -s=sv find 'RAG\|chunk\|embedding\|đáp án\|provider\|truy xuất'` trên cả `/library` và `/library/[id]`
EXPECT: (a) "Chưa có tài liệu nào." (b) lỗi + `Thử lại` (c) khung xương (dùng `PageState`, không spinner thay cả route); (d) **0** kết quả từ kỹ thuật AI và **0** chữ `ANSWER_KEY`/`Đáp án` ở màn sinh viên

### TC-P8-02-37 [AC13] pre: —
STEP: `playwright-cli -s=sv resize 375 812` → `goto $APP/library` → `eval "document.documentElement.scrollWidth <= window.innerWidth"` → `eval "[...document.querySelectorAll('a,button,[role=button],input,select')].filter(e=>{const r=e.getBoundingClientRect();return r.width>0&&(r.height<44\|\|r.width<44)}).length"`; lặp cho `/library/$D` và `/documents` ở 375 px
EXPECT: `scrollWidth ≤ innerWidth` → `true` (không tràn ngang); số vùng chạm < 44 px = **0**; `/documents` ở 375 px dùng được (SRS 7 ghi "dùng được ≥ 720 px" — ghi nhận, xem Q-QC-P802-4); `bash scripts/ui-antipatterns.sh` → `rc=0`

### TC-P8-02-38 [AC13] pre: —
STEP: `playwright-cli -s=sv resize 375 812` → `goto $APP/library` → gõ từ khoá tiếng Việt dài "quy chế đào tạo đại học chính quy theo tín chỉ" → `eval` đo `scrollWidth`; `playwright-cli -s=sv eval "document.querySelectorAll('[data-ep-panel] [data-ep-panel]').length"`
EXPECT: Chữ Việt dài không vỡ bố cục, dấu không bị cắt; panel lồng panel = **0**; zoom 200% (`eval` đặt `document.body.style.zoom='2'`) vẫn thao tác được

### TC-P8-02-39 [AC15] pre: `gv` ở `/documents`
STEP: `playwright-cli -s=gv upload seed/documents/Mordern_Network_Security_Threats.pdf`; khi `PUT` đang chạy: `playwright-cli -s=gv network-state-set offline`; đợi lỗi; `network-state-set online`; bấm `Tải lại` trên hàng đó
EXPECT: Hàng chuyển `Lỗi — Tải lại` và **giữ tên tệp**; bấm `Tải lại` tải đúng tệp đó mà **không** phải chọn lại; `$PSQL -c "select count(*) from documents where sha256='<sha>' and course_id='$C1'"` → `1` (`Idempotency-Key` giữ nguyên)

### TC-P8-02-40 [AC15] pre: Tài liệu `READY` trên `/documents`
STEP: `playwright-cli -s=gv route '**/api/v1/courses/*/documents/*' --method PATCH --status 500`; bấm công tắc `Dùng cho AI`; `snapshot`
EXPECT: Công tắc **hoàn lại** giá trị cũ; có thông báo lỗi tiếng Việt; `$PSQL` xác nhận DB không đổi; dòng "Đã đổi · Hoàn tác" **không** hiện

### TC-P8-02-41 [AC15] pre: `sv` ở `/library`
STEP: Gõ `q=cảnh báo học vụ`; `playwright-cli -s=sv route '**/api/v1/courses/*/library*' --status 500`; `press enter`; `snapshot`; bấm `Thử lại`
EXPECT: Trạng thái lỗi + `Thử lại`; ô tìm kiếm **giữ nguyên** từ khoá đã gõ; bấm `Thử lại` gọi lại đúng truy vấn đó

### TC-P8-02-42 [AC6] pre: Provider nhúng của QC trả `500` cho `/v1/embeddings`; một đoạn có văn bản `T0`
STEP: `j -X PATCH $Q/documents/$D/chunks/$CH -H "$TCH" -d '{"text":"Văn bản mới"}'`; `$PSQL -c "select text from content_chunks where id='$CH'"`; UI: sửa đoạn trong `Drawer` rồi `snapshot`
EXPECT: `503`; văn bản đoạn vẫn là `T0` (giữ nguyên, không ghi nửa chừng); `embedding` cũ còn nguyên; UI hiện **đúng** "Chưa lưu được. Thử lại."; `audit_log` không có dòng mới

### TC-P8-02-43 [AC10] pre: Tài liệu `READY`; xoá object trong kho bằng tay (`mc rm` / API MinIO) nhưng giữ dòng `documents`
STEP: `j -o /dev/null -w '%{http_code}\n' "$Q/library/$D/download" -H "$SVA"`; đọc thân; `playwright-cli -s=sv goto $APP/library/$D` → `click 'Tải xuống'`
EXPECT: `404` `{"code":"FILE_GONE"}`; UI hiện **đúng** "Tệp này không còn nữa."; `download_count` **không** tăng

### TC-P8-02-44 [AC9, AC10] pre: Tài liệu `ANSWER_KEY` (`$DK`), tài liệu ẩn (`$DH`), tài liệu lớp 2 (`$D2`), id bịa (`$DX`)
STEP: `for d in $DK $DH $D2 $DX; do for p in "" "/download"; do j -o /dev/null -w "$d$p %{http_code}\n" "$Q/library/$d$p" -H "$SVA"; done; done`; so sánh thân bằng `diff`
EXPECT: Cả 8 lượt → **404** với **cùng một thân** ("Không tìm thấy tài liệu."); không ca nào `403` (403 sẽ lộ tồn tại); không thân nào chứa `title`, `type` hay bất kỳ metadata nào

### TC-P8-02-46 [AC14] pre: —
STEP: `j -o /dev/null -w '%{http_code}\n' -X DELETE $Q/documents/$D -H "$TA_"` rồi `-H "$TCH"`; `PATCH` bằng cả hai vai
EXPECT: `DELETE`: TA → **403**, TEACHER → `204`; `PATCH`: cả TA và TEACHER → `200`

### TC-P8-02-49 [AC14] pre: SV A = `sv.gioi@`, SV B = `sv.kha@`; B đã tải một tài liệu và có phiên chat tài liệu
STEP: A gọi `j "$Q/library/$D/download" -H "$SVA"` (hợp lệ, để so sánh) rồi `j $GW/api/v1/chat/sessions/<phiên của B> -H "$SVA"`; A gọi `j "$Q2/library" -H "$SVA"` (lớp A không thuộc)
EXPECT: Phiên của B → **404**; `/library` của lớp 2 → **403**; `download_count` của B không lộ ra cho A dưới dạng "ai đã tải"; không thân nào chứa `user_id` của B
