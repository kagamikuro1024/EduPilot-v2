# Handoff — US-S9-SEED (seed sprint 9 + cổng P7)

## Đã làm
- **`scripts/seed.mjs` bước 19** (chỉ HTTP API thật + `cmd/mock-imap`): chờ ≤ 60 s `mock-imap` / `mock-graph` (hết hạn → dừng, in tên dịch vụ); thẻ IMAP + Teams của lớp 1 (ADMIN, trỏ `mock-imap:1143` / `mock-graph`); **Bài tập 03** (`create-bt03.json` của BA, hạn = bây giờ + 75 s) + **Bài tập 04** (kênh IN_APP + TEAMS, id lớp / bài lấy từ `testdata/mock-graph/meta.json`), cả hai công bố bằng TEACHER; 12 sinh viên nộp trong app (`submit/presign` + PUT + `submit`, sv12 hai phiên bản, Idempotency-Key cố định); ZIP 5 tệp qua `uploads/presign` + `submissions/upload` (tên tệp chứa MSSV, riêng sv30 không có); 8 thư qua `POST /__mock/append` (INTERNALDATE = hạn + giờ trong `seed-plan.csv`, `Authentication-Results: mx.edupilot.test; dmarc=pass header.from={domain}`) rồi `sync-now`; Sinh viên B nộp **sau hạn** (chờ tới hạn); cuối cùng `gateway admin seed-age --assignment BT03 --hours 96` đưa hạn + mọi bài nộp / thư về quá khứ NGUYÊN khoảng cách (cửa nộp muộn 2 ngày cũng đóng → sinh viên không nộp là 0 theo `MISSING_SCORE`, AC4). Không chạy chấm. Dữ liệu đọc từ đúng tệp của BA (`scripts/assignment-seed-data.mjs`: `seed-plan.csv`, `expected_grading.json`, `create-bt03.json`, `meta.json`).
- **`cmd/mock-imap`** (`imapserver` + `imapmemserver`, go-imap v2): IMAP `:1143`, cổng điều khiển `:8091` (`POST /__mock/append` đặt INTERNALDATE / header, cùng `message_id` thêm lần hai → không tạo thư; `GET /__mock/health`). `Dockerfile` target `mock-imap`; `docker-compose.local.yml` service `mock-imap` (healthcheck `-healthcheck`) + cổng `127.0.0.1:8091` / mock-graph `127.0.0.1:8090` cho seed; `IMAP_TRUSTED_AUTHSERV_ID=mx.edupilot.test` mặc định ở compose local và `.env.example`.
- **`gateway admin seed-age --assignment <mã> --hours N`** (`internal/seedtool.AgeAssignment`, chỉ dev / test): lùi `due_at` + `received_at` / `submitted_at` bài nộp + `received_at` thư; chạy lại không lùi thêm (bài có hạn trước `now − N/2` thì bỏ qua). Test `TestSeedAgeAssignment`.
- **`seed/demo/fake_llm.json`**: thêm 12 bản ghi chấm (`bt03-s{1,2,3,5,6,7}-p{1,2}`; khoá = nội dung: thứ tự tiêu chí của lượt + câu đầu bài làm — hợp đồng demo là khoá theo nội dung, không theo băm: sửa lời nhắc không làm hỏng bản ghi), 1 bản ghi `SuggestRubric`, 1 bản ghi quy chế bản 2 (`quy-che-lop1-v2`, khoá `Điểm quá trình gồm ba thành phần`, đặt TRƯỚC bản 1 vì cả hai đều chứa "40%"). `seed/demo/quy-che-lop1-v2.pdf` dựng bằng `node scripts/make-policy-pdf.mjs v2` (chữ nguyên của BA; hai PDF cũ không bị ghi đè).
- **`scripts/check-assignments-seed.mjs`** (`bt03`, `counts`, `grade`, `demo5`, `teams`, `v2`; `v2` ở `check-assignments-v2.mjs`) và **`scripts/gate-p7.sh`** (đúng lệnh của `P7.md` + `-race` + các mode trên, một dòng PASS/FAIL mỗi lệnh, nhắc "Bạn tự kiểm").
- Go tests (`internal/integration/seed_assignments_test.go`, dựng thế giới từ đúng tệp của BA như `seed_gradebook_test.go`): `TestSeedImapAuthHeaders`, `TestSeedRosterJoinedVia`, `TestSeedGradeAllMatchesExpected` (22 bài, 42 lời gọi, sv24 0 lời gọi), `TestSeedTeamsHappyCounts`, `TestSeedUnmatchedBlocksZero`, `TestSeedV2AfterResolvingUnmatched` (30/30 đúng `expected_final_grades_v2.csv`), `TestSeedAssignmentsIsolation`, `TestSeedAssignmentsAdminForbidden`, `TestSeedCanaryNotToStudents`; `internal/grading`: `TestExtractLop1V2MatchesExpected`, `TestDemoModeRecordingsCoverPolicies` (đếm theo loại, proposals #15); `cmd/mock-imap/server`: `TestMockImapAuthHeaders`, `TestMockImapAppendIdempotentAndInternalDate`.
- e2e `assignment-lifecycle.spec.ts -g 'demo step 5'` (B "Đang chấm · Nộp muộn 1 ngày" không điểm → giảng viên sửa tiêu chí 2 → Duyệt bài → Công bố → B thấy 7,5 cùng nhận xét và đoạn trích). Sửa nhỏ ở `StudentAssignment.tsx`: trạng thái `Đang chấm` kèm `· Nộp muộn n ngày` (spec AC5: "đã nộp, nhãn nộp muộn, đang chấm").

## Lệch / cần biết
- **Hạn "đã qua" bằng `seed-age`**: API không có cách ghi thời gian quá khứ cho bài nộp trong app; ngoại lệ `gateway admin seed-*` đã được phép (đầu `seed.mjs`). Nếu seed gãy trước bước lùi, chạy lại sẽ tự lùi ở cuối.
- `Kiểm:` ghi `-tags integration`; repo không dùng tag đó (chạy `-tags testroutes` hoặc không tag).
- Tên `fake_llm.json` "khoá theo băm đầu vào" trong AC2 → thực tế khoá theo nội dung (cơ chế `match` có sẵn của provider giả từ sprint 8).
- Chạy lại `seed.mjs` SAU khi đã chạy `check-assignments-seed.mjs v2` không hỗ trợ (công thức bản 2 đã xác nhận làm đổi tín hiệu rủi ro CRM; bước 16 báo lệch). `gate-p7.sh` chạy `counts` TRƯỚC `grade`/`v2`.
- Đổi `fake_llm.json` → khởi động lại `gateway` + `worker` (đọc tệp lúc khởi động).

## Kết quả chạy (stack `pnpm dev` xoá volume, `node scripts/seed.mjs`, 2026-11-01)
| Kiểm | Kết quả |
| --- | --- |
| `seed.mjs` lần đầu | rc 0 (~6 phút gồm chờ hạn 75 s); lần hai 20 s, không tạo đôi |
| AC9: tắt `mock-imap` rồi `seed.mjs` | rc 1, `Dịch vụ mock-imap chưa sẵn sàng sau 60 s (http://localhost:8091/__mock/health)`; bật lại → rc 0 |
| `NODE_ENV=production node scripts/seed.mjs` | rc 1, `Seed bị chặn` |
| `check-assignments-seed.mjs bt03` | `OK 22/22 khớp ngay + 3 chưa khớp + 6 không nộp; IN_APP 13 · IMAP 8 (kèm sv29 + thư lạ) · UPLOAD 5 (kèm sv30)` |
| `check-assignments-seed.mjs counts` | `{"assignments":2,"matched":22,"unmatched":3,"inApp":13,"inbound":8,"documents":6,…}` không đổi sau seed lần hai |
| `check-assignments-seed.mjs grade` | `22/22 khớp expected_grading.json` |
| `check-assignments-seed.mjs demo5` | `B 7.50` |
| `check-assignments-seed.mjs teams` | `24 khớp + 1 chưa khớp` |
| `check-assignments-seed.mjs v2` | `30/30 OK (bản 2)` (INCOMPLETE cho 6 sinh viên không nộp + `GRADES_INCOMPLETE` trước khi gán; gán sv29 / sv30, bỏ qua thư lạ, công bố, xác nhận) |
| `go test … TestSeed* TestExtractLop1V2MatchesExpected` | `ok` (9 + 1) |
| `playwright assignment-lifecycle.spec.ts -g 'demo step 5'` | xanh |
| k6 `grading-vs-chat.js` (22 bài) | xem `dev-US-P7-07.md` mục Vòng 2: first_event p95 28 ms, TTFT +0,05 %, 0 % lỗi |
| `gate-p6.sh` (stack seed sạch, TRƯỚC `v2`) | xem mục dưới |
