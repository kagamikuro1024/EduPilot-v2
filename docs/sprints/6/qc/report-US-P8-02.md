# QC report — US-P8-02 (`/documents` + `/library`)  · Kết luận: PASS phần đã kiểm (TC cần embedding thật / k6 / UI sâu → ghi dưới)

Handoff: `docs/sprints/6/handoff/dev-US-P8-02.md` (HEAD `603b2b7`). Bộ TC: `tc-US-P8-02.md` (49 TC; spec v1.3/1.4).
**Môi trường:** gateway + worker `-tags testroutes` + frontend `next build/start :3410` build từ HEAD, DB riêng `qc_p801`, docling thật; tài liệu thật `QMB12ch6b.pdf`, `Quyche.pdf`, PDF canary (`ANSWER_KEY`), 1 tài liệu ẩn, 1 `EXAM_PAPER`, 22 tài liệu "Bulk" (`use_for_rag=false`) cho phân trang; `playwright-cli` 375 px (`--browser=chromium`). Không đụng stack s55.

## Cổng đã chạy
| Lệnh | Kết quả |
| --- | --- |
| `go test -count=1 -race ./internal/document ./internal/library ./internal/rag ./internal/chat` | PASS — 115 test (lượt đầu chạy song song với docling: `internal/chat` `TestChatSSENotBuffered` 394 ms > 300 ms, do tải; chạy lại khi rảnh xanh) |
| `go test -race -tags testroutes ./internal/contract ./internal/integration -run 'Contract|Spec|TestNoUnscopedChunkQuery|TestStatus'` | PASS |
| `bash scripts/ui-antipatterns.sh` | PASS (rc=0) |

## TC
| TC | Kết quả | Chứng cứ |
| --- | --- | --- |
| 01 | PASS (một phần) | `/documents` (Giảng viên, lớp 761987): một bảng; mỗi hàng loại / công tắc `Dùng cho AI`, `Hiện cho sinh viên`, trạng thái; chưa soát từng cột |
| 02, 04, 08, 36(b,c), 39, 41 | KHÔNG KIỂM ĐƯỢC | tài liệu chia sẻ, trạng thái rỗng / lỗi 500 / chậm, offline giữa lúc tải, bàn phím đầy đủ: chưa chạy (e2e của dev có) |
| 03 | PASS | 27 tài liệu / 3 trang `limit=10`: 27 hàng, 27 khác nhau = DB; `limit=101` → `422`; lọc `type=ANSWER_KEY` 1, `status=READY` 24 |
| 05, 10(UI) | PASS | UI tải tệp: `presign` + `complete` tới gateway (JSON nhỏ), `PUT` thân tệp tới host MinIO `localhost:9000` (không qua gateway); hàng chuyển `Đang xử lý` → `Sẵn sàng` không tải lại trang |
| 06, 07 | KHÔNG KIỂM ĐƯỢC | 4 tệp song song / tệp `.exe` / 60 MiB chưa chạy trên UI |
| 09 | KHÔNG KIỂM ĐƯỢC | chưa chụp dòng "Không hiển thị cho sinh viên" |
| 10 | PASS | `PATCH visible_to_students:true` trên `ANSWER_KEY` → `422 ANSWER_KEY_NOT_VISIBLE`; ràng buộc DB `documents` có `CHECK (type <> 'ANSWER_KEY' OR visible_to_students=false)` |
| 11 | KHÔNG KIỂM ĐƯỢC | hoàn tác lạc quan 5 s trên UI chưa chạy |
| 12 | PASS | `PATCH` với `version` hiện tại `200` (`version` 2); gửi lại `version` cũ → `409 VERSION_CONFLICT` |
| 13 | PASS | SV tìm "dự báo": 1 kết quả → `PATCH visible_to_students=false` → 0 kết quả, `audience` mọi đoạn `STAFF`; bật lại → 1 |
| 14 | PASS | mỗi `PATCH` có `audit_log document.patch` (before/after); tài liệu chia sẻ: chưa thử |
| 15 | PASS (API) | `stats.has_course_policy=true` khi có `COURSE_POLICY`; dòng nhắc UI chưa chụp |
| 16 | PASS | `GET …/chunks` trả `{id, ord, page_no, heading, text, audience}`; `PATCH chunk` `200`, `text` đổi, `embedding` còn, audit `document.chunk.edit` |
| 17 | PASS | `text:""` và 4.001 ký tự → `422` |
| 18 | PASS | `GET /documents/stats`: `total 27`, `by_type`, `by_status` (READY 24 / PROCESSING 1 / QUEUED 2), `chunks 77`, `embedded_chunks 44`, `pages`, `bytes`, `has_course_policy`, `last_upload_at` |
| 19, 20, 21, 46 | PASS | `impact` `{chunks:14, courses:0}`; `DELETE` bởi TA `403`, bởi TEACHER `204`; dòng `documents` 0, đoạn 0, audit `document.delete`; phiên chat có `document_id` của tài liệu đó → `NULL`; `/library` không còn; `PATCH` bởi TA `200` (TA là Staff) |
| 22 | PASS | `/library` 375 px: ô "Tìm tài liệu" (`AES, chữ ký số, quy chế…`) đang `active` đầu tiên; lọc `Loại`, `Tuần` |
| 23 | PASS | `q=bai giang tuan 3`, `q=BÀI GIẢNG`, `q=du bao` → 1 kết quả mỗi cái; `q=a` (< 2 ký tự) bị bỏ qua (23 mục, không `422`); các mục có `id,title,type,file_kind,category,week_no,updated_at,snippet,can_ask_ai` |
| 24 | PASS | tìm theo nội dung đoạn `q="of plumbing"` → ra tài liệu đúng (tên không chứa chuỗi), kèm `snippet` |
| 25 | PASS | SV thấy 23: **không** có `ANSWER_KEY`, **không** có tài liệu `visible=false`; Bulk (READY) hiện |
| 26 | PASS | 23 mục qua 3 trang `limit=10` không lặp / sót; `limit=101` → `422` |
| 27 | PASS | chi tiết có `file_kind=PDF`, `size_bytes`, `page_count`, `preview_url`; GET `preview_url`: `206`, `Content-Type: application/pdf`, `Content-Disposition: inline` |
| 28 | PASS | 20 `download` song song → `200` cả 20, `download_count` 0 → **20** (nguyên tử); URL tải `200` |
| 29 | PASS | `ETag: W/"…"`; `If-None-Match` → `304` |
| 30 | PASS | phiên có `document_id`: hỏi bằng đúng đoạn của tài liệu đó → trích dẫn thuộc đúng tài liệu; hỏi bằng đoạn của tài liệu khác → "chưa tìm thấy" (không lẫn tài liệu khác) |
| 31 | PASS | `EXAM_PAPER` `use_for_rag=false`: `/library/{id}` `200`, `can_ask_ai:false`; tạo phiên `document_id` đó → `404/400` (không hỏi được) |
| 32 | KHÔNG KIỂM ĐƯỢC | chưa ghép khoá giờ thi + `document_id` (khoá giờ thi đã PASS ở P3-05) |
| 33, 34 | PASS | chat "Tìm tài liệu về dự báo" → intent `LIBRARY_SEARCH`, khối `library_results` ≤ 5 mục kèm `href /library/{id}`; "tìm tài liệu ZZTEST-L2" (chỉ có ở lớp 2) → "Hệ thống chưa có dữ liệu tài liệu của bạn."; "tìm slide đáp án đề giữa kỳ" chỉ trả tài liệu Bulk (LECTURE hiển thị) chứa chữ canary, **không** trả tài liệu `ANSWER_KEY` |
| 35 | PASS (một phần) | `/library` không có "Luyện đề này" (nút "Luyện đề" duy nhất là mục điều hướng) |
| 37, 38 | PASS (một phần) | `/library` 375 px: `scrollWidth − clientWidth = 0`; mọi điều khiển ≥ 44 px trừ link ẩn "Bỏ qua điều hướng"; ô tìm lấy focus; chưa thử từ khoá rất dài |
| 40 | KHÔNG KIỂM ĐƯỢC | `PATCH` 500 giả trên UI |
| 42 | KHÔNG KIỂM ĐƯỢC | chưa ép nhúng lỗi (fake luôn nhúng được; dev có `TestEditChunkEmbedFailureKeepsOld`) |
| 43 | PASS | xoá object khỏi kho (xoá thư mục object trong `/data/qc-p801`) → `download` → `404 FILE_GONE` "Tệp này không còn nữa." |
| 44 | PASS | tài liệu `ANSWER_KEY`, ẩn, id bịa: `GET /library/{id}` và `/download` đều `404`; SV ngoài lớp (sv31): `/library` `403`, `/library/{id}` `404` |
| 45 | PASS | SV gọi `/documents`, `/documents/stats`, `/documents/{id}/chunks` → `403`; `/documents/{id}` `404`; ADMIN `/documents` `403` |
| 47 | PASS | `/library`: SV `200`; TA, TEACHER, ADMIN `403` |
| 48, 49 | KHÔNG KIỂM ĐƯỢC | chưa dựng SV `PENDING`/`REMOVED`; chưa thử SV khác dùng phiên/tải của người khác |

## AC
AC1 (một phần), AC3, AC4, AC5 (API), AC6, AC7, AC8, AC10–AC14 PASS; AC9 PASS về chức năng; AC2, AC15 (kéo thả thật, mạng xấu, hoàn tác) chưa kiểm được trên UI.

## Lỗi / ghi chú
- **Ghi chú 1 (Thấp, TC-09 / AC9 chữ):** dev ghi sắp xếp `/library` theo `updated_at` chưa theo độ khớp (AC9 yêu cầu độ khớp trước); với kho nhỏ chưa thấy khác biệt. Đề nghị BA/PM quyết giữ chữ AC hay sửa (dev đã nêu).
- **Ghi chú 2 (Thấp):** trên `/documents` 375 px tên tài liệu là link cao 32 px (< 44 px); màn Staff không nằm trong luật 375 px, nên không tính lỗi.
- **Ghi chú 3:** vỏ trang vẫn hiện tên "Nguyễn Minh Trung" / "Bản mô phỏng" với phiên thật (đã nêu ở P3-05).
- **Ghi chú 4:** xoá object khỏi MinIO làm ngay trong request sau commit (dev đã khai) — lỗi xoá chỉ log, có thể mồ côi.

## Đề nghị
PASS phần kiểm được; chuyển các TC UI sâu (02, 04, 06–09, 11, 39–41), 32, 42, 48, 49 sang lượt gate P8 / `gate-P8.md`. Câu AC9 (xếp theo độ khớp) xin PM/BA quyết.
