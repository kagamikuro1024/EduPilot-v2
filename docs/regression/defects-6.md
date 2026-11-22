### DEF-R6-01 — [F giữ, «nợ PR» (PROGRESS.md:73: NOSUPERUSER NOBYPASSRLS) — PM 2026-11-21: không giao sửa lúc này] Vai DB của ứng dụng là siêu quyền
- Mức: Thấp
- Sprint gốc: 6 · TC gốc: TC-P3-01-49 · Hồi quy do sprint sau?: KHÔNG (có sẵn — chính là BUG-1 trong `docs/sprints/6/qc/report-US-P3-01.md`)
- Môi trường: ep-r, be0c0d96; gateway nối Postgres bằng vai `edupilot`
- Bước tái hiện: 1. `docker exec ep-r-postgres-1 psql -U edupilot -d edupilot -Atc "select rolsuper,rolbypassrls from pg_roles where rolname='edupilot'"` → `t|t`.
- Kỳ vọng: vai ứng dụng không `rolsuper`; Thực tế: `rolsuper=t`. Trigger append-only (`pii_events`, `audit_log`) vẫn chặn UPDATE/DELETE với 42501 nên không mất bất biến.
- Bằng chứng: lệnh trên; `docs/regression/smoke-6.md` · Nghi ngờ mã: cấu hình compose (user Postgres = user ứng dụng), không phải mã Go
- Chạy lại lần 2: ổn định

### DEF-R6-02 — Bộ nhớ đệm roster PII không bị vô hiệu khi sinh viên vào lớp (duyệt hoặc vào bằng mã)
- Mức: Cao (tên SV mới vào lớp không được che trong Threads/chat tới 1 giờ ⇒ lộ họ tên người khác ở kênh công khai)
- Sprint gốc: 6 · TC gốc: TC-P3-02-21 · Hồi quy do sprint sau?: KHÔNG RÕ (có vẻ có sẵn: `members.go` ở db7c187b đã chỉ phát `course.member_changed` ở mời ra / hoàn tác; duyệt phát `course.join_decided`, vào tự động không phát gì; worker chỉ gắn `roster.Invalidate` vào `course.member_changed`, `cmd/worker/registry.go:129`)
- Môi trường: ep-r, be0c0d96; GV `teacher`, SV; lớp E6C1 (`class_code=E6C1`)
- Bước tái hiện: 1. Bật `require_approval` (PUT `/courses/{id}/join-settings`). 2. `redis-cli DEL ep:roster:<course>`; SV A `POST …/threads/precheck` để dựng cache (không có SV B). 3. SV B `POST /courses/join` (PENDING) rồi GV `POST /courses/{id}/members/{B}/approve` (200). 4. `redis-cli EXISTS ep:roster:<course>` sau 10 s → vẫn `1`; `precheck` với «Đỗ Bảo Hải …» (tên SV B) → `allowed:true`. Tương tự khi lớp không cần duyệt (vào mã). Mời ra thì cache xoá sau 0,5 s (đúng).
- Kỳ vọng (SRS P3-02 AC3 / TC-P3-02-21): cache roster xoá ≤ 5 s sau khi thành viên đổi, tên SV mới bị chặn; Thực tế: cache sống đến hết TTL 1 giờ.
- Bằng chứng: lệnh trên (chạy hai lần, cùng kết quả); outbox chỉ có `course.member_changed` cho thao tác mời ra · Nghi ngờ mã: `backend-go/internal/course/members.go:222` (approve chỉ ghi `TopicJoinDecided`), `backend-go/internal/course/join.go:185-269` (không ghi sự kiện khi ACTIVE ngay), `cmd/worker/registry.go:129`
- Chạy lại lần 2: ổn định
- **ĐÃ SỬA (afbd433e), RERUN 2026-11-16 ĐẠT:** vào bằng mã (không duyệt) → outbox `course.member_changed` +1, `ep:roster` xoá ≤ 3 s; duyệt 1,0 s; mời ra 0,5 s. TC-P3-02-21 đạt. (Lượt trước còn đỏ vì gateway ep-r chạy ảnh cũ do cửa sổ chaos khôi phục nhầm ảnh `edupilot-gateway`.)
### DEF-R6-C-01 — Dòng khoá ở /threads dùng chữ của chat thay vì "Đăng bài tạm khóa…"
- Mức: Thấp
- TC: TC-P3-06-42
- Hồi quy do sprint sau?: KHÔNG (có từ commit 3f1e376a, US-P3-06)
- Bước tái hiện: 1. SV có lượt thi IN_PROGRESS. 2. Mở /threads?course=<lớp>, bấm "Đặt câu hỏi". 3. Đọc dòng báo trạng thái trên ô soạn.
- Kỳ vọng / Thực tế: SRS FEAT-private-chat-pii 472 "Đăng bài tạm khóa trong lúc bạn làm bài thi. Dùng lại được sau {HH:mm}." / hiện "Chat tạm khóa trong lúc bạn làm bài thi. Dùng lại được sau 00:08." (e2e threads.spec.ts:180 cũng khẳng định chữ chat).
- Nghi ngờ mã: frontend/src/features/threads/RealThreads.tsx:186

### DEF-R6-U1 — (ĐÃ SỬA 812ab974; chạy lại P ngày 2026-11-21 trên ep-r 425396a7) Chữ «Tìm hiểu» của chat riêng khác SRS
- Mức: Thấp · TC: TC-P3-05-35 · Sprint gốc: 6 · Hồi quy do sprint sau?: KHÔNG RÕ
- Bước tái hiện: SV mở /chat, mở «Tìm hiểu» về việc che thông tin.
- Kỳ vọng (SRS FEAT-private-chat-pii / TC): «Tên, mã số sinh viên, email và số điện thoại được thay bằng ký hiệu trước khi gửi cho AI, rồi hiện lại cho bạn.» · Thực tế: «Tên, MSSV, email, số điện thoại được thay bằng ký hiệu trước khi gửi cho AI và khôi phục khi hiển thị cho bạn.»
- Nghi ngờ mã: frontend/src/features/chat/RealChat.tsx:416
### DEF-R6-U2 — (ĐÃ SỬA 812ab974; chạy lại P ngày 2026-11-21 trên ep-r 425396a7) Tiêu đề hộp thoại PII ở Threads khác SRS
- Mức: Thấp · TC: TC-P3-06-09 · Kỳ vọng «Bài viết có thông tin cá nhân» · Thực tế «Bài này có thông tin cá nhân» · Nghi ngờ mã: frontend/src/features/threads/PIIChannelDialog.tsx:33
### DEF-R6-U3 — (ĐÃ SỬA 812ab974; chạy lại P ngày 2026-11-21 trên ep-r 425396a7) Nút đăng lại khi mất mạng ở Threads khác SRS
- Mức: Thấp · TC: TC-P3-06-34 · Kỳ vọng nút «Đăng lại» · Thực tế «Gửi lại»
### DEF-R6-U4 — (ĐÓNG: P-11-80, spec đổi — «Chưa có câu hỏi nào», không nút; TC-P3-06-37 chấm lại P ngày 2026-11-21) Trạng thái rỗng /threads thiếu câu mời
- Mức: Thấp · TC: TC-P3-06-37 · Kỳ vọng «Chưa có câu hỏi nào. Đặt câu hỏi đầu tiên.» · Thực tế «Chưa có câu hỏi nào»
### DEF-R6-U5 — (ĐÃ SỬA 812ab974; chạy lại P ngày 2026-11-21 trên ep-r 425396a7) Chữ lỗi MASK_FAILED của chat khác SRS
- Mức: Thấp · TC: TC-P3-02-50 · Kỳ vọng (SRS dòng 114) «Chưa gửi được tin nhắn. Thử lại.» · Thực tế «AI đang gián đoạn. Thử lại sau.» (backend-go/internal/chat/buf.go errorText nhánh mặc định). Fail-closed đúng (go test TestMaskFailsClosed, TestMaskNoCourseFailsClosed); không bắn được thật vì PRIVACY_MASK_TIMEOUT_MS=1 không làm tin ngắn quá hạn.
### DEF-R6-U6 — (ĐÃ SỬA 812ab974; chạy lại P ngày 2026-11-21 trên ep-r 425396a7) Chữ CHAT_UNAVAILABLE ở UI khác SRS
- Mức: Thấp · TC: TC-P3-05-26 · API đạt (503 CHAT_UNAVAILABLE khi tra khoá lỗi, bảng exam_attempts đổi tên tạm). Kỳ vọng «Chat chưa dùng được lúc này. Thử lại sau ít phút.» · Thực tế frontend/src/shared/i18n/vi.ts:68 «Chat tạm thời chưa dùng được. Thử lại sau.»
### DEF-R6-D1 — (ĐÃ SỬA ec157a06 + 210f72d6; chạy lại P 2026-11-21 @50fe1a97: API removed:true, UI «Nguồn đã gỡ» ở 1440 và 375) Trích dẫn cũ vẫn hiện tên tài liệu đã bị xoá, không có «Nguồn đã gỡ»
- Mức: Trung bình · TC: TC-P3-05-42 (phần b); ảnh hưởng nghi ngờ TC-P8-02-20 · Sprint gốc: 6 · Hồi quy: KHÔNG RÕ
- Bước tái hiện: 1. SV hỏi, nhận câu trả lời có trích dẫn tài liệu X. 2. GV xoá X. 3. SV mở lại phiên chat.
- Kỳ vọng (FEAT-docs-calendar SRS dòng 105, US-P3-05 AC15b): trích dẫn hiện «Nguồn đã gỡ» · Thực tế: vẫn hiện tên X; chuỗi «Nguồn đã gỡ» không tồn tại trong backend-go/internal/chat/*.go hay RealChat.tsx.
### DEF-R6-P1 — (ĐÓNG: P-11-79, spec đổi — mẩu <200 rune cuối trang hợp lệ; TC-P8-01-18 chấm lại P) Bộ chia đoạn không gộp mẩu < 200 rune qua ranh giới trang
- Mức: Thấp · TC: TC-P8-01-18 · Kỳ vọng (SRS 142 / US-P8-01 AC6): mẩu < 200 rune gộp vào mẩu kề nếu tổng ≤ 800 · Thực tế: PDF 23 trang slide QMB12ch6b.pdf → 4 đoạn < 200 rune không phải đoạn cuối (ord 0: 152, 1: 39, 5: 117, 6: 88; 117+88=205 ≤ 800 gộp được). Các tính chất khác đạt (ord liên tục, ≤800, duy nhất, token_count NULL, trùng (document_id,ord) → 23505).
### DEF-R6-P2 — (ĐÃ SỬA 506ae3a0; chạy lại P ngày 2026-11-21 trên ep-r 425396a7) Chia sẻ tài liệu không tăng ep:rag:ver của lớp đích
- Mức: Trung bình · TC: TC-P8-01-37 · Kỳ vọng (US-P8-01 AC14): chia sẻ → outbox document.changed → ep:rag:ver tăng ≤ 5 s, cache câu trả lời cũ không trúng · Thực tế: tải tài liệu / PATCH cờ / sửa đoạn / xoá đều tăng 1 trong ≤ 5 s; POST /courses/{đích}/share-from (tài liệu mới vào document_courses, course_ids của lớp đích) giữ nguyên 26→26 trong 24 s.
- Nghi ngờ mã: backend-go/internal/course/share.go không ghi outbox document.changed (chỉ document/manage.go và ingest/process.go ghi).
### DEF-R6-U7 — (dev2 1f9738d4 — chạy lại P @859a90db) Hàng ANSWER_KEY ở /documents không ghi hai dòng cố định
- Mức: Thấp · TC: TC-P8-02-09 · Kỳ vọng (US-P8-02 AC3, SRS dòng 381): hàng ghi đúng «Không hiển thị cho sinh viên» và «Không dùng cho AI của sinh viên» · Thực tế: hàng có công tắc «Dùng cho AI: bật» (bật/tắt được) và «Hiện cho sinh viên: tắt» (khoá, chỉ có tooltip); hai dòng chữ chỉ ở khung chi tiết. Nghi ngờ mã: frontend/src/features/documents/RealDocuments.tsx cột `ai`/`vis` (dòng ~120-128).
### DEF-R6-U8 — (dev2 1f9738d4 — chạy lại P @859a90db) Chữ lỗi chọn tệp sai loại/quá cỡ khác SRS
- Mức: Thấp · TC: TC-P8-02-07 · Kỳ vọng «Dùng PDF, DOCX hoặc PPTX, tối đa 50 MB.» · Thực tế .exe «<tên>: chỉ nhận PDF, DOCX hoặc PPTX. Hãy chọn tệp khác.», 60 MiB «<tên>: quá 50 MB. Hãy nén hoặc tách tệp.» (frontend/src/shared/data/uploadFile.ts:10-11). Chặn trước khi gọi mạng đạt (0 yêu cầu presign/PUT).
### DEF-R6-U9 — (dev2 5fbb9c8f — chạy lại P @859a90db (hiện khi lớp có chủ đề)) Thư viện của SV thiếu bộ lọc «Chủ đề»
- Mức: Thấp · TC: TC-P8-02-22 · Kỳ vọng (US-P8-02 AC9, SRS 377): lọc theo tuần / chủ đề / loại · Thực tế: chỉ «Loại» và «Tuần» (API đã có `category`).
### DEF-R6-U10 — (dev2 6e362a9b — chạy lại P @859a90db (1 lớp)) Hộp xác nhận xoá tài liệu: «{m} lớp» bằng 0 với tài liệu chưa chia sẻ
- Mức: Thấp (cần BA xác nhận ngữ nghĩa) · TC: TC-P8-02-19 · Thực tế «Xoá ui06j2? 1 đoạn và 0 lớp đang dùng tài liệu này sẽ mất nó.»; GET /documents/{id}/impact trả `courses` = số lớp được chia sẻ tới (không tính lớp gốc). Phần còn lại của TC đạt.
### DEF-R6-P3 — (dev2 701f71e8 — chạy lại @859a90db: vẫn F Thấp (cụm nhiều từ không chứa nguyên cụm)) snippet của /library?q= không chứa từ khoá
- Mức: Thấp · TC: TC-P8-02-24 · Thực tế: snippet = 160 ký tự đầu của đoạn khớp đầu tiên theo `ord` (OR các cặp âm tiết), ví dụ phần mở đầu văn bản không chứa «cảnh báo học vụ» · Nghi ngờ mã: backend-go/internal/store/queries/library.sql:7-9 (và dòng 40).
### DEF-R6-P4 — (dev2 545bbdd4 — chạy lại P @859a90db) search_library không được nối vào agent Threads
- Mức: Trung bình · TC: TC-P8-02-33 · Kỳ vọng (SRS 213, US-P8-02 AC12): tool chạy ở cả hai kênh · Thực tế: chat riêng đạt (5 mục ≤5, href /library/{id}); Threads: thread hỏi «có tài liệu nào về an ninh mạng không?» → SKIPPED/LOW_SCORE, 0 lời gọi tìm thư viện; `agent.NewPublicRegistry` (internal/agent/tools.go:100) không có nơi gọi nào ngoài test.

- LT-đã-xoá TC-P3-01-05: proposals #13 (số 00009 bỏ trống, calendar là 00011)
- LT-đã-xoá TC-P3-03-31: GET /admin/llm/usage mở cho TEACHER: openapi «ADMIN, TEACHER» (sprint 9/10)
- LT-đã-xoá TC-P3-04-45: US-P4-07 2026-10-14: khủng hoảng hai tầng, câu mẫu đổi
- LT-đã-xoá TC-P3-05-19: FEAT-escalation-notify US-P4-01 AC7 (sprint 7, APPROVED): nút «Nhờ giảng viên hỗ trợ» nay hiện dưới mọi câu trả lời DONE; quyết định ẩn 2026-10-10 chỉ tới sprint 7
- LT-đã-xoá TC-P3-05-44: như TC-P3-05-19 (US-P4-01 AC7 thay quyết định ẩn nút 2026-10-10)

### DEF-R6-T1 — [N: PM 2026-11-21 — đo lại ở cửa sổ SEED cuối khi các stack khác dừng; còn > 4 phút thì F gửi dev] Seed bước 11 (tài liệu docling thật) vượt 4 phút trên DB trống
- Mức: Thấp (đo trên máy dùng chung; PM quyết nới ngưỡng hay chạy lại trên máy rảnh) · TC: TC-P8-03-51 (AC17) · Sprint gốc: 6
- Kỳ vọng: seed tăng thêm ≤ 4 phút (docling thật). Thực tế: bước 11 = 241 s, 268 s, 264 s ở ba lần seed DB trống (evidence-6/k6/seed/seed-t0-clean.json, seed4/); phần chức năng (docs=8 READY=5 answer_key=1 shared=5 reembedded=0 events=2/1 idempotent=ok, lần hai không thêm dòng) đạt.

### DEF-R6-D2 — (ĐÃ SỬA 210f72d6; chạy lại 2026-11-21 @50fe1a97: GET messages 200 cả với trích dẫn tài liệu đã gỡ) GET /chat/sessions/{id}/messages 500 với mọi phiên có trích dẫn (hồi quy do ec157a06, DEF-R6-D1)
- Mức: Cao · TC: TC-P3-05-42 (b), TC-P8-02-20 · Sprint gốc: 6 (sửa D1) · Hồi quy: có
- Tái hiện (ep-r 859a90db): SV hỏi chat có trích dẫn tài liệu; GET /api/v1/chat/sessions/{sid}/messages → 500 INTERNAL. Log gateway: «chat: tài liệu của trích dẫn: failed to encode args[0]: unable to encode []uuid.UUID{…} into text format for unknown type (OID 0): cannot find encode plan».
- Nghi ngờ mã: backend-go/internal/chat/sessions.go:264 `select id from documents where id = any($1)` với `ids []uuid.UUID`; db.go dùng QueryExecModeExec (PgBouncer transaction mode) nên không có OID để mã hoá mảng uuid; test testcontainers nối thẳng PG nên xanh. Sửa: `any($1::uuid[])` + tham số có kiểu (pgtype.FlatArray / []string) hoặc sqlc; thêm test chạy ở QueryExecModeExec.

### DEF-R6-U11 — [giao dev2] Danh sách /documents không tự cập nhật khi tệp được tải lên từ nơi khác
- Mức: Thấp · TC: TC-P8-01-17 (AC5) · Sprint gốc: 6
- Tái hiện: Staff mở /documents (không tải lại); tải tệp lên bằng API/curl ở nơi khác; sau 70 s hàng mới KHÔNG xuất hiện; không có điều hướng, không có cập nhật. Nghi ngờ mã: RealDocuments.tsx chỉ refresh() khi chính trang thao tác; documentsApi.ts không có refetchInterval/useSSE. Chỉ hàng tải lên từ chính trang đổi trạng thái Đang tải → Đang xử lý n % → Sẵn sàng (TC-P8-02-05 đạt). Kỳ vọng AC5: Staff mở /documents thấy hàng mới + trạng thái đổi không tải lại.

### DEF-R6-U12 — [PM 2026-11-21: BA làm proposal (SRS docs-calendar dòng 97 nói cả hai); giữ F tới khi PM chốt] Docling tắt: tài liệu ra FAILED sau ≤ 2 phút, TC/AC17 và SRS dòng 97 nói QUEUED
- Mức: Thấp (xung đột spec: AC10 «thử lại ≤ 3 lần rồi FAILED EXTRACT_UNAVAILABLE» và AC17 / SRS dòng 97 «tài liệu QUEUED khi dịch vụ tắt») · TC: TC-P8-01-43 (AC17) · Sprint gốc: 6
- Tái hiện: docker stop docling (ep-r); tải PDF mới: complete → 202; sau 2 phút documents.status = FAILED (error «Chưa đọc được tài liệu lúc này. Thử lại sau.», jobs.error.code EXTRACT_UNAVAILABLE), không phải QUEUED; bật lại docling thì tài liệu không tự chạy tới READY (vẫn FAILED; TC kỳ vọng tự chạy tiếp). Cần BA/PM chốt: giữ QUEUED khi dịch vụ tắt (đợi dịch vụ) hay FAILED sau 3 lần thử.
