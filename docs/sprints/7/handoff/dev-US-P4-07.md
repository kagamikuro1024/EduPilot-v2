# DEV handoff — US-P4-07 (khủng hoảng: bộ luật hai tầng, câu mẫu ân cần, Threads không lưu, giờ thi không khoá, cấu hình hỗ trợ)
Nhánh: `sprint/7-p4-p5`. Commit: `US-P4-07: …` (một commit).

## Đã làm
- **Bộ luật hai tầng** (`internal/agent/intent.go`, thay `reCrisis` của sprint 6): tầng 1 (không cần ngôi) = 18 cụm trên chữ bỏ dấu + `tự tử` / `tự hại` / `tự làm đau` trên chữ CÓ DẤU (biên từ `(?:^|[^\p{L}])…(?:$|[^\p{L}])`); tầng 2 (chỉ khi có ngôi thứ nhất đứng ngay trước, kèm trợ từ tuỳ chọn) = `tự tử | tự hại | tự làm đau | chết đi | cắt tay | treo cổ | kết thúc tất cả | chán sống` (có dấu và bỏ dấu). `tu tu` / `tu hai` / `tu lam dau` **chuyển** sang tầng 2 ⇒ "Em đọc từ từ", "Join từ hai bảng" không còn là `CRISIS` (lỗi sprint 6). Khớp trên `strings.ToLower` + `auth.Fold`, không thêm thư viện. **Lệch SRS một chữ:** SRS nói "NFC bằng thư viện chuẩn hoá mà `auth.Fold` dùng", nhưng `auth.Fold` không dùng thư viện nào (bảng tự viết) và `golang.org/x/text` chỉ là phụ thuộc gián tiếp; để khỏi thêm thư viện, chuỗi NFD (dấu kết hợp) không so được dấu nên rơi về luật cũ của sprint 6 (`tu tu` / `tu hai` / `tu lam dau` khớp không cần ngôi): thà bắt nhầm còn hơn bỏ sót. Có test.
- **Câu mẫu** (`agent.CrisisText(support)`): `ReplyCrisis` + `SUPPORT_RESOURCES_VI` (rỗng → `DefaultSupport`), dùng chung `Agent.crisisReply` và Threads. Chữ giữ nguyên văn bản chốt SRS 4.9.3.
- **Threads**: `thread.Service.gate` (chạy cho `Create`, `Comment`, `EditThread`, `EditPost`) kiểm `DetectIntent(title+"\n"+body) == IntentCrisis` TRƯỚC tường lửa PII ⇒ `422 CRISIS_SUPPORT`, `message` = câu mẫu, `details.actions=["OPEN_PRIVATE_CHAT"]`; không `forum_*`, không `pii_events`, không outbox / job (không AI), không thông báo. `precheck` không đổi, không bao giờ trả mã này.
- **Chat riêng**: `chat.precheck(ctx, a, text)` — tin khủng hoảng bỏ qua khoá giờ thi và không đếm `RecordChatBlocked` (vẫn qua giới hạn tốc độ). Tin khủng hoảng **không** đặt tiêu đề phiên (quét canary phát hiện `chat_sessions.title` giữ nguyên chữ tin ⇒ vi phạm AC3; tin thường sau đó đặt tiêu đề như cũ).
- **Cấu hình**: `SUPPORT_RESOURCES_VI` (`config.SupportResourcesVI`, ≤ 300 ký tự theo rune, dài hơn → `Load` lỗi nêu tên biến); nối vào `Agent.SupportResources` (trước đó KHÔNG được nối ở gateway — chưa từng đọc biến) và `thread.Service.Support`; `docker-compose.local.yml` đặt giá trị mô phỏng "Phòng Công tác sinh viên, phòng 101 nhà A1, giờ hành chính".
- **Frontend**: `CrisisSupportDialog` (cùng tệp `PIIChannelDialog.tsx` nên qua danh sách Dialog của `ui-antipatterns`), `usePostGate` bắt `CRISIS_SUPPORT`; lối chính `Mở chat riêng` (chỉ sinh viên; dùng `from-draft` sẵn có, nháp không lưu ở máy chủ), lối phụ `Quay lại sửa` giữ nguyên chữ; trợ giảng / giảng viên chỉ có `Quay lại sửa`. Dùng chung cho sửa bài (`EditForm`).

## Test (không sửa test cũ nào)
- `internal/agent/crisis_test.go` + `testdata/crisis_20.json` / `crisis_neg_17.json` (SRS 9.5, toàn bộ bịa): `TestCrisis20Recall` (20/20), `TestCrisisNegatives17`, `TestCrisisAmbiguousNeedsFirstPerson`, `TestCrisisBeatsOtherIntents`, `TestCrisisHyperboleStillSupport`, `TestCrisisReplyTone`, `TestCrisisReplyNoAdvice`, `TestCrisisReplyNoJargon`, `TestCrisisReplyUsesSupportResources`, `TestCrisisReplyDefaultSupport`. `TestCrisisCannedReply`, `TestCrisisNoLLMNoNotify`, `TestQCCrisisSet` (sprint 6) xanh nguyên.
- `internal/platform/config/support_test.go`: `TestSupportResourcesConfigLimit`.
- `internal/chat/crisis_test.go`: `TestCrisisNoSideEffects` (gồm ý `TestCrisisNeverTicketed` và `TestCrisisCanaryOnlyInMessages`: quét MỌI bảng trừ `chat_messages` + log tìm `CANARY-CRISIS-5X`), `TestCrisisAllowedDuringExamLock`, `TestStaffCannotReadCrisisMessages`.
- `internal/thread/crisis_test.go`: `TestThreadCrisisNotStored`, `TestThreadCrisisNoAIAnswer`, `TestCommentCrisisNotStored` (kèm sửa), `TestPrecheckNeverReturnsCrisisSupport`.
- e2e: `frontend/e2e/threads-crisis.spec.ts` — ca `crisis dialog keeps draft` + mở chat riêng + vai trợ giảng. **Lệch US:** US ghi `threads.spec.ts -g 'crisis dialog keeps draft'`; vì không được sửa spec cũ khi chưa có proposal nên ca nằm ở tệp mới; lệnh `-g 'crisis dialog keeps draft'` chạy cả thư mục vẫn tìm thấy.

## Lệnh QC chạy
```bash
cd backend-go && export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true
go test -count=1 -race -tags testroutes ./internal/agent -run 'Crisis' -v
go test -count=1 -race -tags testroutes ./internal/chat ./internal/thread ./internal/platform/config -run 'Crisis|StaffCannotReadCrisis|Precheck|SupportResources'
grep -n SUPPORT_RESOURCES_VI ../docker-compose.local.yml
cd ../frontend && E2E_API_PORT=3412 pnpm build:gate
E2E_PORT=3410 E2E_API_PORT=3412 pnpm exec playwright test threads-crisis.spec.ts --project=desktop
bash ../scripts/ui-antipatterns.sh                                   # rc=0
```

## AC tự đánh giá
AC1 ✓ (20/20, 0/17 dương tính giả) · AC2 ✓ · AC3 ✓ (phát hiện và sửa tiêu đề phiên) · AC4 ✓ · AC5 ✓ · AC6 ✓ (`TestSupportResourcesConfigLimit`; đã nối biến vào gateway lần đầu) · AC7 mã ✓ (`TestCrisisHyperboleStillSupport`), phần tài liệu `docs/thesis-notes/` + `QUESTIONS.md` `Q-crisis-hyperbole` là việc BA (SRS 4.9 đã có đủ ba ý) · AC8 ✓.

## Cần PM / nợ
1. `docs/ARCHITECTURE.md` bảng env: thêm `SUPPORT_RESOURCES_VI` (BA); `docs/thesis-notes/` + `Q-crisis-hyperbole` (BA).
2. Lệch SRS về "thư viện chuẩn hoá" (xem trên) — BA sửa chữ SRS 4.9.1 hoặc PM cho thêm `golang.org/x/text` làm phụ thuộc trực tiếp.
3. Bình luận sinh viên gõ NFD dấu kết hợp và dùng cụm tầng 2 có dấu (vd. "em tự tử" ở dạng NFD) chỉ bắt được qua luật nới cho `tu tu` / `tu hai` / `tu lam dau`; cụm tầng 2 khác ("em cắt tay") ở dạng NFD vẫn khớp qua nhánh bỏ dấu. Không có giới hạn nào khác đã biết.
