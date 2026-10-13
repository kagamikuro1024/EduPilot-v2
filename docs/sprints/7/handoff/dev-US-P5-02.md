# DEV handoff — US-P5-02 (điểm danh: lưới, PUT một giao dịch, phát biểu, hàng đợi ngoại tuyến, nối dây tool AI)
Nhánh: `sprint/7-p4-p5`. Commit: `US-P5-02: …` (một commit). CI: xem cuối tệp.

## Đã làm (theo thứ tự lát dọc)
- **Store / sqlc** (`queries/crm.sql`, `crm.sql.go` sinh): lưới (sinh viên `ACTIVE` sắp theo tên không dấu + dòng điểm danh + `speak_count`), khoá buổi, ghi hàng theo lô, đếm theo trạng thái cho tool AI, sự kiện tham gia (thêm / danh sách / xoá).
- **`internal/crm`** (mới): `GetGrid`, `PutGrid` (một giao dịch: hàng giống hệt = bỏ qua, `version + 1` hàng đổi, `audit_log` không tên cùng giao dịch, outbox `crm.attendance_saved`, `complete` đặt `attendance_done_at` / điền `PRESENT` cho hàng thiếu, `edit_after_complete`, xung đột → `409 VERSION_CONFLICT` kèm `details.conflicts[{student_id,mine,current}]`, tất cả hoặc không), `AddParticipation` (idempotent theo `client_event_id`), `ListParticipation` (con trỏ), `DeleteParticipation` (người tạo hoặc TEACHER), nguồn tool `Attendance` / `Participation` (chỉ đọc dòng của chính người đó, không ghi chú, không tên người điểm).
- **HTTP** (`internal/httpapi/crmhttp`): 5 route (SRS 6.2), guard `Staff`; `routes.go` dựng `crmService(d)`; `chatwire.go` truyền vào `agent.DefaultPrivateRegistry`; worker đăng ký `crm.attendance_saved` (`cmd/worker/registry.go`, không thêm vào `today.Topics()` vì `staff_test.go` ghim 24).
- **Cấu hình**: `ATTENDANCE_MAX_ROWS` (1.000), `PARTICIPATION_DEFAULT_POINTS` (0,25). Mã lỗi mới `STUDENT_NOT_IN_COURSE` (422).
- **OpenAPI**: 5 thao tác + schema; `contract_test.go` 189 → 194 (#11), bảng `want` bảo mật, kịch bản, miễn `PUT …/attendance` 503 ở `contract/exempt.go` (cùng lý do các route ghi khác: lỗi DB giả lập có test riêng `TestPutGridDBDownAtomic`).
- **Frontend**:
  - `shared/lib/saveQueue.ts` thành generic (`SaveQueue<V>`, kiểu mặc định giữ nguyên cho bài thi) + `discard(itemId)`; `shared/lib/unsavedWork.ts` (đếm việc chưa lưu, `wipeDrafts`); `OfflineBanner` nêu "Chưa có mạng · N thay đổi chờ lưu" khi có việc chờ.
  - `shared/domain/AttendanceGrid.tsx`: máy tính = bảng radio, `1`–`4` đặt rồi xuống hàng, `↑↓`, `Shift+1…4` giữ hàng, `P` phát biểu; điện thoại = danh sách hàng chỉ chữ trạng thái ("Vắng" đỏ), bấm hàng mở bốn lựa chọn ≥ 44 px (đổi một người = 2 chạm).
  - `features/attendance`: `attendanceApi.ts`, `useAttendance.ts` (hàng đợi `SaveQueue` khoá `ep:draft:<userId>:attendance:<course>:<session>`; mark gửi sau 2 s, phát biểu gửi ngay; xung đột dừng hàng đợi, chọn từng dòng `Giữ của tôi` / `Lấy của {tên}` rồi `restart`; `Lưu điểm danh` = `PUT complete:true` cho **cả lưới** một lời gọi), `RealAttendance.tsx`, `AttendanceScreen.tsx`. **Xoá** màn mô phỏng `AttendanceView.tsx` + CSS (không còn `@/mock` trong `features/attendance`).
  - Đăng xuất: `AuthProvider.logout` xoá khoá `attendance:` của người đó; `AppShell` hỏi `ConfirmIrreversible` ("Còn điểm danh chưa lưu · N thay đổi…") khi còn việc chờ.
  - e2e: `attendance.spec.ts` (17 ca), `support/att-fixtures.ts` (gateway giả có trạng thái), mặc định `asDemo` cho `…/sessions` và `…/attendance`; `exam-save-queue.spec.ts` thêm ca generic / `discard` / `fatal` / `restart`.

## Quyết định cần nêu
1. **`Lưu điểm danh` gửi cả lưới** (mọi sinh viên, kèm version đã biết), không chỉ hàng đổi: AC12 "đúng một lời gọi ghi cả lưới"; hàng không đổi bị bỏ qua và không xung đột (AC3), nên không tốn gì thêm. Tự lưu giữa chừng chỉ gửi hàng đã đổi.
2. **Phát biểu đi qua cùng hàng đợi** (mất mạng không mất; `client_event_id` khử trùng) nhưng gửi ngay, không chờ 2 s. Hoàn tác: chưa gửi → bỏ khỏi hàng đợi; đã gửi → `DELETE` (gửi lại cùng `client_event_id` trả sự kiện cũ nếu chưa biết mã).
3. **Số đếm phát biểu hiển thị** = số lúc mở màn + thay đổi của máy này (không tự tải lại lưới); lưới chỉ đọc một lần khi mở một buổi. Người khác ghi thêm sẽ thấy khi mở lại.
4. **Test AC14 nằm ở `internal/crm`**, không ở `internal/agent`: `crm` import `agent` nên gói `agent` không import ngược được. `go test ./internal/agent -run TestToolMyAttendance` sẽ không chạy gì; dùng lệnh ở mục dưới. Tên khớp US: `TestToolMyAttendance`, `TestToolMyParticipation`, `TestToolAttendanceNoteNotExposed`, `TestAgentSourcesReadOnlyOwnRows`, `TestChatAttendanceMatchesDB` (qua `PrivateRegistry.Run`, không qua LLM).
5. `TestPersonalToolsHaveNoIdentityParam` (agent) xanh không đổi.

## Lệnh QC chạy
```bash
cd backend-go && export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true
go test -count=1 -race -tags testroutes ./internal/crm -v           # lưới, PUT, xung đột, hoàn tất, validate, tham gia, RBAC, nguồn tool
go test -count=1 -race -tags testroutes ./internal/contract/... ./internal/httpapi/... ./internal/agent
sqlc generate && sqlc diff
cd ../frontend && E2E_API_PORT=3412 pnpm build:gate
E2E_PORT=3410 E2E_API_PORT=3412 pnpm exec playwright test attendance.spec.ts exam-save-queue.spec.ts --project=desktop
bash ../scripts/ui-antipatterns.sh                                   # rc=0
```

## AC tự đánh giá
AC1 ✓ · AC2 ✓ (`TestPutGrid30Under500ms` xanh) · AC3 ✓ (e2e hai giảng viên là gateway giả có `409` thật theo hợp đồng; hai ngữ cảnh trình duyệt trên stack thật để QC) · AC4 ✓ · AC5 ✓ · AC6 ✓ · AC7 ✓ · AC8 ✓ · AC9 ✓ · AC10 ✓ · AC11 ✓ · AC12 ✓ (kịch bản ≤ 15 s; bấm giờ tay trên điện thoại để QC / PO) · AC13 ✓ (chưa chọn cắt) · AC14 ✓ · AC15 ✓ · AC16 ✓ (`grep -rn "@/mock" frontend/src/features/attendance` = 0; `ATTENDANCE_SEED` còn dùng bởi màn khác nên chưa xoá — xoá sau US-P5-04 / P5-06).

## Cần PM / nợ
1. e2e `visual.spec` (13 ca) đỏ khi chạy ngoài ảnh Docker Playwright (phông máy khác); không đụng tới màn nào ở đây. CI là nguồn đúng.
2. `TestSchemaAttendance` (US-P5-01, `internal/store/crm_schema_test.go:64`) đỏ MỘT lần trong ~7 lượt `go test ./...` cục bộ (không lặp lại khi chạy riêng 3 lần; nên không rõ nguyên nhân). Gói `store` không đổi ở story này. Chạy với `-count>1` thì đỏ chắc chắn (`23505` thay vì `23503`: fixture dùng chung DB, không lặp lại được) — không phải lỗi của CI (`-count=1`). Ghi để theo dõi.
3. `docs/ARCHITECTURE.md` bảng env: thêm `ATTENDANCE_MAX_ROWS`, `PARTICIPATION_DEFAULT_POINTS` (BA).

## Vòng 2 (QC `report-US-P5-02.md` TC-38 / AC14: câu ghép chỉ trả nửa sau)
- **Gốc**: `PersonalIntent` / `DetectIntent` trả MỘT intent theo thứ tự ưu tiên SRS 4.5; câu "Em đã nghỉ mấy buổi, được cộng bao nhiêu điểm phát biểu rồi ạ?" khớp cả `rePart` và `reAtt`, thắng `PERSONAL_PARTICIPATION` nên `get_my_attendance` không chạy.
- **Sửa** (`internal/agent`): `Classification.Extra` = các ý cá nhân còn lại của cùng câu (`extraPersonal`, chỉ trong điểm danh / điểm cộng / điểm; điểm danh = `vắng|nghỉ|điểm danh|chuyên cần` + tự xưng; điểm = `điểm|kết quả|…` sau khi bỏ cụm "điểm danh / điểm cộng / điểm phát biểu / chuyên cần" để "điểm phát biểu" không tự thành ý "điểm"). Phân loại vẫn MỘT lần (D47). `Agent.personal` chạy tool của MỌI ý (thứ tự khối cố định điểm danh → điểm cộng → điểm), gộp dữ kiện (khoá `<nhãn>.<trường>`), một ý thiếu dữ liệu thì thêm dữ kiện "chưa có dữ liệu: …", rồi ĐÚNG MỘT lần sinh chữ; thiếu dữ liệu cả hai → câu mẫu, 0 lời gọi LLM. Câu một ý chạy đúng đường cũ (`a.tool`), không đổi hành vi.
- **Test mới** (`internal/agent/compound_test.go`, không sửa test cũ): `TestCompoundPersonalQuestion` (hai tool đều chạy, hai khối `attendance` + `participation`, 1 lần sinh, danh tính từ trusted context), `TestCompoundIntents` (bảng ý chính / phụ, câu một ý không có `Extra`), `TestCompoundNoData`. Toàn bộ `internal/agent`, `internal/crm`, `internal/chat` xanh.
- QC kiểm lại TC-38 bằng chính câu của AC với sinh viên B (vắng 2, 2 phát biểu): phản hồi phải nêu cả "2 buổi" và "0,50 điểm".
