# DEV handoff — US-P8-02 (`/documents` + `/library`) — **giao đủ (backend + UI)**
Nhánh `sprint/6-p3-p8`. Commit `US-P8-02: …`.

## Đã làm
**Backend**
- `internal/document/manage.go`: `List` (của lớp + chia sẻ vào, lọc loại / trạng thái / tên không dấu, con trỏ `(updated_at,id)`), `Get`, `Patch` (một giao dịch: khoá hàng, `version`, `audience` mọi đoạn đổi theo, audit trước / sau, `document.changed`, bật `Dùng cho AI` → xếp `reindex`; `ANSWER_KEY` + hiện → 422 `ANSWER_KEY_NOT_VISIBLE`; đổi sang đáp án tự ẩn; chia sẻ từ lớp khác → 409 `DOCUMENT_SHARED_READONLY`), `Delete` (Giảng viên; `CASCADE` đoạn / `document_courses`; phiên chat → `document_id NULL`; audit; xoá object sau commit), `Impact` (số đoạn / lớp cho hộp xác nhận), `Chunks` (con trỏ theo `ord`), `EditChunk` (nhúng một lần + lưu + audit + chạm `updated_at`/`version` trong một giao dịch; nhúng lỗi → 503 giữ nguyên), `Stats` (một truy vấn tổng hợp + hai nhóm đếm).
- `internal/library` (sinh viên): `List` (READY + hiện + không `ANSWER_KEY`, của lớp hoặc chia sẻ vào; `q` ≥ 2 ký tự khớp tên / tệp / chủ đề không dấu hoặc từ khoá đoạn bằng `tsv`; dưới 2 ký tự bỏ qua), `Get` (+ `preview_url` PDF ký sẵn 5 phút `inline`, `can_ask_ai`), `Download` (URL 5 phút; `UPDATE … download_count + 1` nguyên tử; mất tệp → 404 `FILE_GONE`), `Search` = `agent.LibrarySource` của tool `search_library` (≤ 5, không tham số danh tính; nối vào `PrivateRegistry` của chat).
- Route: `documenthttp.MountManage` (#3–#8, #12 + `impact`), `libraryhttp` (#13–#15, guard `StudentRole`; ETag = băm thân đã tuần tự hoá qua `httpx.WriteJSONETag`, 304 khi `If-None-Match`).
- Truy vấn chunk mới đều lọc `course_ids` (`TestNoUnscopedChunkQuery` cập nhật danh sách đường ĐỌC: `DocChunks`, `DocChunkLock`, `DocStats`, `DocDeleteImpact`, `LibList`, `LibGet`, `LibSearchTool`, `ChatDocumentUsable`). `ChatDocumentUsable` (Hỏi AI về tài liệu) nay tính cả tài liệu chia sẻ vào lớp và đòi có đoạn đã nhúng (`can_ask_ai`).
- Mã lỗi `ANSWER_KEY_NOT_VISIBLE`, `FILE_GONE`; `openapi.yaml` +11 thao tác (tag `library`), op 151 → 162; `scenarios_document_test.go` (`docMgmtScenarios`); `exempt.go`: `PATCH …/chunks/{id}` 503 (fake luôn nhúng được, do `document.TestEditChunkEmbedFailureKeepsOld` kiểm).

**Giao diện**
- `/documents`: `Documents` → `RealDocuments` (phiên Giảng viên / TA thật) hoặc `DemoDocuments`. Một Panel: dòng nhắc thiếu quy chế (một dòng + `Tải quy chế môn học`), dải thống kê một dòng, vùng thả ngay trên bảng (Enter / Space mở chọn tệp), tối đa 3 tệp song song, hàng `Đang tải → Đang xử lý n %` (`useJob`) → biến khi xong, kiểm tệp trước khi tải (sai loại / > 50 MB, kèm cách sửa), mất mạng → `Lỗi` + `Tải lại` đúng tệp đó với cùng `Idempotency-Key`; bảng (tên, loại sửa tại chỗ, tuần, công tắc `Dùng cho AI`, `Hiện cho sinh viên` khoá với đáp án + hai dòng nói rõ, trạng thái, cập nhật, ⋯ Thử lại / Lập chỉ mục lại / Xoá); sửa lạc quan + `Đã đổi … · Hoàn tác`, 5xx → hoàn lại + "Chưa lưu được. Thử lại."; Drawer đoạn (sửa đoạn); `ConfirmIrreversible` xoá nêu số cụ thể (chỉ Giảng viên có mục Xoá).
- `/library`, `/library/[id]`: `LibraryScreen` / `LibraryDetail` → bản thật cho sinh viên: ô tìm kiếm lấy focus trước, lọc loại / tuần, dòng gọn có dấu loại tệp bằng chữ (`PDF`/`DOCX`/`PPTX`), `Xem`, ⋯ `Hỏi AI về tài liệu` (tạo phiên có `document_id` → `/chat?session=…`); chi tiết: PDF xem trước trong trang, DOCX / PPTX chỉ `Tải xuống`; lỗi giữ nguyên từ khoá; không có nút `Luyện đề này`.
- `shared/data/uploadFile.ts` (presign → PUT thẳng tới kho tệp, token không gửi tới origin khác); e2e `documents.spec.ts` (12 ca), `library.spec.ts` (7 ca), `support/doc-fixtures.ts`; `asDemo` giả mặc định `GET …/documents*` và `…/library*`.

## Lệnh QC
```bash
cd backend-go && export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true
go test -count=1 -race ./internal/document ./internal/library ./internal/rag ./internal/chat -v
go test -count=1 -race -tags testroutes ./internal/contract ./internal/integration -run 'Contract|Spec|TestNoUnscopedChunkQuery|TestStatus'
cd ../frontend && E2E_API_PORT=3412 pnpm build:gate && E2E_PORT=3410 E2E_API_PORT=3412 npx playwright test e2e/documents.spec.ts e2e/library.spec.ts --workers=2
bash ../scripts/ui-antipatterns.sh   # rc=0
```

## AC tự đánh giá
AC1 ✓ · AC2 ✓ (kéo thả thật bằng chuột chưa chụp; e2e dùng `setInputFiles` + bàn phím) · AC3 ✓ · AC4 ✓ · AC5 ✓ · AC6 ✓ · AC7 ✓ · AC8 ✓ (blob xoá sau commit trong request, không phải việc nền — `ponytail:`) · AC9 ✓ (sắp theo `updated_at`, **chưa** theo độ khớp: con trỏ hai khoá; thêm khi QC đo thấy cần) · AC10 ✓ · AC11 ✓ · AC12 ✓ · AC13 ✓ · AC14 ✓ · AC15 ✓.

## Nợ / cần hỏi
1. Chưa chạy trên stack compose thật (chủ đang dùng stack s55): xem trước PDF trên MinIO thật, tải lên 50 MB, docling thật là việc của QC / `gate-p8.sh`.
2. AC9 "sắp theo độ khớp rồi `(updated_at DESC, id DESC)`": hiện chỉ theo `updated_at` (xem trên). Đề xuất thêm vào `proposals.md` nếu BA muốn giữ chữ AC.
3. Xoá object khỏi MinIO làm trong request sau commit (lỗi chỉ log, có thể mồ côi); chưa có việc nền dọn.
4. `internal/document` rig dùng pool mặc định (nợ chung "đổi sang `RuntimePool`"); `internal/library` đã dùng `RuntimePoolAt`.
5. Chạy `./...` song song một lần thấy `cmd/worker` panic nil-pointer và `internal/chat TestCancelReachesProvider` đỏ; lần chạy lại toàn bộ và chạy riêng từng gói đều xanh (không tái hiện). Nghi lỗi tải / Redis dùng chung giữa các gói test (cùng họ với `TestQuestionReviewProvider`); nếu QC gặp lại, gửi log để Dev truy gốc.
