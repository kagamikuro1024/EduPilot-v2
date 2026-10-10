# QC report — US-P8-01 (ingest nền + `internal/rag`)  · Kết luận: PASS phần đã kiểm (chấm lại sau fix `6b61556`, 2026-10-11); TC cần UI/PATCH/chat chuyển theo Q1

Handoff: `docs/sprints/6/handoff/dev-US-P8-01.md` (commit `287e8bb`). Bộ TC: `tc-US-P8-01.md` (62 TC; spec đã lên v1.3 — TC chưa cập nhật, đối chiếu v1.3 khi khác).
**Môi trường:** stack đang chạy của dev là bản sprint 5.5 (chưa có route tài liệu), nên QC chạy gateway + worker build từ `287e8bb` (`-tags testroutes`) làm tiến trình cục bộ (`:18080`), trỏ vào Postgres / Redis / MinIO dùng chung của stack dev nhưng **DB riêng `qc_p801`, Redis db 9, bucket `qc-p801`**; docling thật = container `qc-docling` (image `edupilot-docling`, 3 GiB, `DOCLING_SERVE_MAX_NUM_PAGES=400`). Seed `scripts/seed.mjs` chạy tới bước 10/10 (dừng ở `JUDGE_UNAVAILABLE`, không liên quan). Provider LLM là `fake` trong tiến trình → **không quét được payload** (xem TC-21). Script: `scripts/p801-lib.mjs`.

## Cổng đã chạy
| Lệnh | Kết quả |
| --- | --- |
| `sqlc generate && sqlc diff` | PASS |
| `go test -count=1 -race ./internal/ingest ./internal/rag ./internal/document ./internal/contract ./internal/store` | **FAIL lượt đầu** (131 pass, 1 fail: `TestIngestLeaseRenewed`, `ingest_test.go:564`, xảy ra khi chạy song song 5 gói); chạy riêng 4/4 lượt xanh → test nhạy thời gian, ghi BUG-5 |
| `TestAnswerKeyNeverRetrieved`, `TestRetrievalStillFullAfterForbiddenNeighbors`, `TestNoPublicGradingSearch` | PASS |
| `TestVnBigramQuery`, `TestSchemaChunkTSV`, `TestTSVIndexUsed` | PASS |
| goose up → down → up trên DB trống | PASS (`00010` lên `version: 10`; tsv `GENERATED … to_tsvector('simple', vn_fold(NORMALIZE(text, NFC)))`, GIN `content_chunks_tsv_gin`). `00009` chưa tồn tại (US-P8-03) nên TC-45 mục "đứng sau 00009" chưa kiểm được |

## TC đã chạy
| TC | Kết quả | Chứng cứ |
| --- | --- | --- |
| 01 | PASS | `200`, đủ trường, `expires_in` 600; `ep:upload:<id>` TTL 900 |
| 02 | PASS | `.exe`, `application/zip`, `.txt`+pdf → `422 FILE_TYPE_NOT_ALLOWED`, không để khoá |
| 03 | PASS | 52428801 / 0 / −1 → `422 FILE_TOO_LARGE` (0 và −1 cũng báo "quá lớn" — thông điệp lệch, Thấp) |
| 04 | PASS | sha ngắn, sha có `G`, tên 256 ký tự, `../`, `\` → `422 VALIDATION_FAILED` |
| 05 | PASS (hành vi) | 10×`200`, lần 11 `429 RATE_LIMITED retry_after 5`. Giá trị đếm Redis = **11** (TC viết 10; SRS chỉ nói INCR, 10/phút) → Q-QC |
| 06 | PASS | lớp lưu trữ → `409 COURSE_ARCHIVED` |
| 07 | PASS (một phần) | PUT đúng → `200`; PUT sai kích thước → `503` của MinIO (không `200`; TC mong 4xx, MinIO trả 503 khi thân ngắn); hết hạn 10 phút chưa chạy |
| 08, 09 | PASS | `202 {job_id, document}`; `QUEUED`, `use_for_rag=t`, `visible=t`, `sha256` khớp, kind `document.ingest`; 3 lần cùng khoá → 3 thân giống hệt, `count=1` |
| 11 | PASS | `409 DOCUMENT_DUPLICATE` `details.existing_id` đúng; 1 dòng `documents` |
| 12 | PASS | `409 DOCUMENT_DUPLICATE_ELSEWHERE` `details.class_code=761987`, `document_id` |
| 14 | PASS | `202` + `409 DOCUMENT_DUPLICATE`; `count=1` |
| 15 | PASS | `QMB12ch6b.pdf` → `READY`, `page_count=24`, 14 đoạn (≥ 8); tổng ~2 p 43 s từ lúc tải tới `READY` gồm hàng đợi + docling khởi động lạnh |
| 16 | PASS (lệch tên) | `job.progress` đơn điệu `5 → 11…26 → 50 → 95 → 100` (đọc 11–26, chia đoạn 50, nhúng 95); sự kiện cuối là `job.progress status=SUCCEEDED, progress=100, result.document_id` — **không có** sự kiện riêng `job.result` như TC viết; SRS chỉ định nghĩa `job.progress` → TC viết sai, không phải lỗi dev |
| 18 | PASS (theo v1.3 #12) | `ord` 0…13 liên tục, `max=798`, 14 đoạn, 14 khác nhau, `token_count` NULL; đoạn 0 dài 192 rune không gộp được với đoạn kề (192+796 > 800) → hợp lệ theo trần-thắng-sàn |
| 19 | PASS | `Quyche.pdf` (bản scan, OCR `vie`): 5 trang, 15 đoạn, 8 đoạn có "học vụ"; `page_no` không giảm |
| 21 | PASS (một phần) | 1536 chiều; `llm_audit EMBEDDING / BATCH`, 1 lời gọi mỗi tài liệu 14 đoạn. **Không kiểm được** kích thước lô ≤ 100 và chuỗi `heading+"\n"+text` (không có đường đọc payload) |
| 22 | PASS | `use_for_rag=false`: 14 đoạn, 0 `embedding`, không thêm lời gọi nhúng |
| 23 | PASS | `ANSWER_KEY`→`GRADING` (và `visible_to_students` mặc định `false`), `visible=false`→`STAFF`, còn lại `ALL` |
| 25 | KHÔNG KIỂM ĐƯỢC | tin QC `XADD ep:ingest job_id … document_id …` bị consumer bỏ ("tin hỏng, bỏ", đã ACK, không crash); tên trường thật không có trong spec |
| 37 | PASS (một phần) | `ep:rag:ver:<C1>` tăng theo mỗi lần `READY` (đo 0→5 sau 4 tài liệu); chưa đo 5 s cho PATCH/xoá/chia sẻ (P8-02) |
| 39 | PASS (một phần) | `reindex` tài liệu `READY` → `202`, job `SUCCEEDED`, 14 đoạn đều có `embedding`, số `POST /v1/convert` ở docling **không tăng** (7→7). Chưa tắt docling |
| 40 | PASS (một phần) | `reindex` cả lớp `202`; TA → `403`; xem BUG-3 |
| 43 | **FAIL** | docling dừng → tài liệu `FAILED EXTRACT_UNAVAILABLE` sau ~20 s; TC và SRS 3.3 ("tài liệu `QUEUED` khi dịch vụ tắt") mong `QUEUED` |
| 46 | PASS (một phần) | `vn_bigram_query('canh bao hoc vu')` = `'canh' <-> 'bao' | 'bao' <-> 'hoc' | 'hoc' <-> 'vu'`; một âm tiết → chính nó; chuỗi tiêm `''') | & ! (x) :* a` → `'x' <-> 'a'` (không lỗi cú pháp). NFD/NFC chưa thử |
| 48 | **FAIL** | `FILE_TYPE_MISMATCH` đúng mã, 0 dòng `documents`; `message` = "Nội dung tệp không đúng với loại tệp." ≠ SRS 3.3 "Tệp không đọc được. Dùng PDF, DOCX hoặc PPTX." |
| 49 | PASS | không PUT / PUT sai kích thước → `422 UPLOAD_INCOMPLETE` (PUT sai: MinIO `403`), không tạo `documents`/`jobs` |
| 50 | PASS (2/3 ca) | `upload_id` bịa và `upload_id` của TA dùng bởi TEACHER → `404 UPLOAD_NOT_FOUND` cùng thân; ca hết TTL chưa chạy |
| 51 | **FAIL** | cùng nguyên nhân TC-43: không thử lại 3 lần theo 5 s / 30 s / 2 phút; `FAILED` ngay |
| 52 | PASS | PDF 2 trang trắng → `FAILED` sau 9 s, `NO_TEXT` (log), message "Không đọc được chữ trong tệp (có thể là bản quét). Hãy tải bản có chữ." đúng SRS |
| 53 | **FAIL** (Thấp) | `401` trang → `FAILED` sau 3 s, code `TOO_MANY_PAGES` đúng (log + job `error.code` của job; lưu ý lần đầu QC quên `DOCLING_SERVE_MAX_NUM_PAGES=400`, không phải lỗi dev); `message` = "Tài liệu quá dài (tối đa 400 trang). Hãy tách nhỏ rồi tải lại." ≠ SRS "Tệp dài hơn 400 trang. Hãy tách nhỏ." |
| 54 | PASS | PUT tệp khác cùng kích thước → `FAILED HASH_MISMATCH`, "Tệp bị lỗi khi tải lên. Tải lại." |
| 56 | PASS (một phần) | `error` ≤ 1000, tiếng Việt, không URL / đường dẫn / host / dấu vết ở 3 mã đã gặp |
| 57 | **FAIL** | `retry` trên `READY` → `409 DOCUMENT_NOT_FAILED` ✓; `reindex` chưa-READY chưa thử; **gửi `retry` hai lần cùng `Idempotency-Key`: lần 1 `202`, lần 2 `409` (không phải cùng phản hồi)** |
| Phân quyền | PASS | SV / ADMIN presign → `403 FORBIDDEN`; chưa đăng nhập `401`; TA presign `200` (đúng ma trận SRS: TA là Staff); TA reindex cả lớp `403`; reindex tài liệu lớp khác `404` |

**KHÔNG KIỂM ĐƯỢC (tính FAIL theo luật QC; đề nghị chuyển theo Q1):** 10, 17 (cần UI `/documents`, US-P8-02); 24, 32, 41 (cần `PATCH /documents`, US-P8-02); 28, 29–31, 33–36, 38 (cần chat P3-05 / `search_library` / `/library` P8-02; mức `rag` đã có `TestAnswerKeyNeverRetrieved` PASS); 13 (cần TEACHER chỉ dạy lớp 2: seed không có); 20 (chia sẻ `share-from` chưa thử), 26, 27, 42, 44, 47 (tải nặng / k6 / kill tiến trình — chưa chạy), 55 (cần provider trả 512 chiều), 21 một phần.

## AC
| AC | Kết quả |
| --- | --- |
| AC1 presign | PASS |
| AC2 complete + idempotent | PASS |
| AC3 kiểm đầu vào | FAIL (Thấp: thông điệp `FILE_TYPE_MISMATCH` lệch SRS) |
| AC4 trùng | PASS |
| AC5 trích → chia → nhúng, SSE | PASS |
| AC6 đoạn | PASS |
| AC7 nhúng | PASS một phần (lô ≤ 100 chưa kiểm được) |
| AC8 audience | PASS (PATCH ở P8-02) |
| AC9 idempotent / một-PROCESSING / thuê | CHƯA KIỂM (TC-25, 26, 27) |
| AC10 lỗi + thử lại | **FAIL** (TC-43, 51, 53 thông điệp, 57) |
| AC11–AC14 truy xuất | `TestAnswerKeyNeverRetrieved` & liên quan PASS; API chưa có |
| AC15 reindex | PASS một phần |
| AC17 docling | FAIL (TC-43/51); RAM `qc-docling` đỉnh 1,81 GiB < 3 GiB |
| AC18 migration | PASS |
| AC19 k6 | KHÔNG KIỂM ĐƯỢC |

## Lỗi
- **BUG-1 (Cao)** — Docling tắt/chết giữa lúc đọc → tài liệu `FAILED EXTRACT_UNAVAILABLE` ngay (~20 s), không thử lại 3 lần (5 s / 30 s / 2 phút) và không ở lại `QUEUED`. Tái hiện: `docker stop qc-docling`, tải tệp, quan sát `documents.status` mỗi 10 s: `PROCESSING` (t+10) → `FAILED` (t+20); log worker: một dòng `ingest thất bại code=EXTRACT_UNAVAILABLE error="docling không dùng được: gửi tệp"`. Kỳ vọng: SRS 3.3 / AC10 / AC17. Nghi: `internal/ingest` phân loại lỗi kết nối (connection refused) là không tạm thời. Hệ quả thực tế: mỗi lần docling khởi động lại làm hỏng mọi tài liệu đang nạp.
- **BUG-2 (Trung bình)** — `POST /documents/{id}/retry` bỏ qua `Idempotency-Key`: cùng khoá lần hai trả `409` thay vì phản hồi cũ (luật 14; TC-57). Tương tự `POST /documents/reindex` cùng khoá tạo **hai job** (`document.reindex_all` ×2). SRS chỉ gắn `[K]` cho `complete`, nên cần BA/PM quyết có yêu cầu `[K]` cho `retry` / `reindex` hay không.
- **BUG-3 (Thấp)** — Thông điệp lỗi lệch SRS 3.3: `FILE_TYPE_MISMATCH`, `TOO_MANY_PAGES` (xem TC-48, 53). Ba thông điệp còn lại (`NO_TEXT`, `HASH_MISMATCH`, `EXTRACT_UNAVAILABLE`) khớp.
- **BUG-4 (Thấp)** — `size_bytes` 0 / −1 trả `FILE_TOO_LARGE` "Tệp quá lớn" (thông điệp sai nghĩa; TC chấp nhận mã).
- **BUG-5 (Thấp)** — `TestIngestLeaseRenewed` đỏ khi chạy song song 5 gói (`ingest_test.go:564`, ngưỡng 150 ms), xanh khi chạy riêng.
- Ghi chú: đường `ep:ingest` chịu được tin hỏng (bỏ + ACK); `delete from documents` thủ công không dọn object MinIO (ngoài phạm vi).

## Kiểm phản mẫu / luật mở rộng / phân quyền
Không `fetch` trần phía frontend (story không đụng UI). Luật 12: việc nặng chạy nền, `202` + job, tiến độ SSE ✓. Luật 10: upload qua presign, gateway không giữ tệp ✓ (không đo bộ nhớ). Phân quyền: bảng trên.

## Đề nghị
FAIL tới khi sửa BUG-1 (chặn merge), BUG-2 (chờ quyết định `[K]`), BUG-3. Chuyển các TC cần UI / PATCH / chat sang report P8-02 / P3-05 (Q1). Câu hỏi: Q-QC-P801-9 (TC-05 giá trị đếm 11), Q-QC-P801-10 (TC-16 tên sự kiện cuối `job.progress`, không `job.result`).

## Chấm lại sau fix `6b61556` (2026-10-11)
Môi trường như trên (gateway/worker build từ HEAD, DB `qc_p801`, docling thật `qc-docling` có `MAX_NUM_PAGES=400`).

| Lỗi | Kết quả | Chứng cứ |
| --- | --- | --- |
| BUG-1 (docling tắt → `FAILED`; TC-43, 51) | **PASS** | `docker stop qc-docling`, tải tệp: tài liệu ở `QUEUED` suốt 143 s (job `RUNNING`, thử lại), không `FAILED`; `docker start` → `READY` sau 184 s tổng, 14 đoạn, một bộ đoạn |
| BUG-2 (Idempotency-Key `retry`/`reindex`; TC-57, 40) | **PASS** | `retry` cùng khoá: `202`/`202`, cùng `job_id`; `reindex` cả lớp và `reindex` tài liệu cùng khoá: cùng `job_id`; không gửi khoá vẫn `202` |
| BUG-3 (thông điệp; TC-48, 53) | **PASS** | `FILE_TYPE_MISMATCH`: "Tệp không đọc được. Dùng PDF, DOCX hoặc PPTX."; 401 trang: "Tệp dài hơn 400 trang. Hãy tách nhỏ." (khớp SRS 3.3) |
| BUG-5 (`TestIngestLeaseRenewed` nhạy thời gian) | **PASS** | 3 lượt `go test -race` 5 gói song song xanh, test chạy riêng xanh |
| Cổng | **PASS** | `go test -count=1 -race ./internal/{ingest,rag,document,contract,store}` xanh; `sqlc diff` rc=0 |

BUG-4 (thông điệp `size_bytes` 0 / −1) chưa sửa (Thấp, không chặn).

**Kết luận story:** các TC kiểm được ở tầng API đều PASS; AC10, AC17 nay PASS. TC chưa kiểm được (UI `/documents`, `PATCH`, chat / `search_library`, k6, kill tiến trình, provider 512 chiều: 10, 13, 17, 20, 24, 26–36, 38, 41, 42, 44, 47, 55) chuyển sang `report-US-P8-02` / `report-US-P3-05` theo Q1.
