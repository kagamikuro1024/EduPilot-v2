# Báo cáo QC — US-PE-09 (seed bài thi mẫu, k6, cổng PE)
**Kết luận: PASS có điều kiện** (chấm lại ở `bc09dd2`, góp ý #18 / #19 ACCEPTED). TC-17 **PASS** khi đo theo #18 (trễ thật 300 ms, 3 cặp, trung vị): tỉ lệ p95 `mixed` / `chat` **1,008 / 0,998 / 0,989 → trung vị 0,998** (≤ 1,2); p95 `mixed` 313–317 ms (≤ 1,5 s); `Chạy thử` p95 1,03 / 1,53 / 2,05 s (≤ 5 s). Lần chấm đầu (FAIL) đo với `fake` không trễ (nhiễu) và **bị sai vì 429 / 404 được tính là thành công** trong k6 cũ (dev đã sửa script); toàn bộ số k6 cũ của QC (judge_burst 1.040 ms, autosave 67 ms, chat ~5 ms) **bỏ**. Điều kiện còn lại: TC-19 `GET …/results` 1.000 dòng + 240 bài; `lhci` có đăng nhập, 3G chậm (TC-26/27).

## Lỗi / lệch
- **B1 (đã đóng, TC-17).** Lần đầu FAIL (+23 / +26 / −1 %) vì đo với `fake` ~5 ms; PM #18 chuyển sang `FAKE_LLM_LATENCY=300-300` + ngưỡng ×1,2 và ≤ 1,5 s + trung vị 3 cặp. Chấm lại: xem Kết luận.
- **L0.** k6 cũ cho 4xx qua `expectedStatuses` (429 / 404 coi là OK) nên số đo chat / autosave của QC lượt đầu không dùng được; chạy lại bằng script mới (2xx / 3xx mới là thành công) trên stack thử có `testroutes` (`docker-compose.test.yml` + `test-seed`): chat trên stack dev thường trả 404 (`http_req_failed` 99,6 %), đúng với mô tả của dev.
- **L1 (TC-27).** `lhci` cho `/exams/<id>/take` không chạy được (cần đăng nhập); thay bằng quan sát trong trang (`PerformanceObserver`) ở 375 px: LCP **72 ms** (`/exams`) và **92 ms** (`/take`) không throttle. Chưa có LCP 3G chậm.
- **L2.** Lần đo đầu (trước khi PM chuyển repo) cũng vượt (+23 %); log `k6-pe09/mixed.log`.
- **L3.** Cổng `gate-pe.sh` chạy `TestSandboxAttacks` **không** có `-tags integration` (≈ 2 s) — QC chạy lại với tag: 15 ca `A1…` PASS (8,9 s) trong bộ integration.

## Kết quả
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| 01, 03 | PASS | `check-exam-seed.mjs bank` → `OK`; DB: 12 `MCQ_SINGLE` + 4 `MCQ_MULTI` + 4 `TRUE_FALSE` `APPROVED` `MANUAL`, 5 `AI_DRAFT` `PENDING`, 2 `CODE` `APPROVED` |
| 02 | PASS (một phần) | 20 câu có `explanation` (`check … bank`); trọng số từng ô SRS 4.11: `bank` kiểm; QC chưa đối chiếu tay từng ô |
| 04 | PASS | `grep -c '_test/' scripts/seed.mjs` = 0; không INSERT / psql (chỉ chú thích) |
| 05, 06, 07 | PASS | 2 bài (`MCQ` 10 câu / `CODE` 2 bài × 5 test; khung 2 phút), sau ~3 phút cả hai `PUBLISHED`; 24 lượt MCQ + 6 lượt code |
| 08 | PASS | QC tự chấm 24 lượt MCQ bằng `fractions` từ `exam_answers` + `answer_key` (`p509-score.py`): **0 lệch**, 4 lượt `MCQ_MULTI` điểm một phần; `check-exam-seed.mjs scores` → `matched 30/30` (mô hình độc lập của dev; 6 lượt code theo verdict thật) |
| 09 | PASS | `multi_scoring = PARTIAL`; có lượt chọn một phần |
| 10 | PASS | `check-exam-seed.mjs demo` → `OK`; UI: ảnh `shots/pe09` (xem TC-31) |
| 11 | PASS (một phần) | `@real` không chạy; QC tự đăng nhập bằng `playwright-cli` (GV, SV 375 / 1440) — `shots/pe09` |
| 12, 13 | PASS | seed lần hai **12 s**; đếm trước / sau y hệt (2 bài thi, 27 câu, 30 lượt, 60 users, 61 enrollments, 11 bản nộp; Mailpit 59 → 59); lần đầu 76 s (≤ 5 phút) |
| 14 | PASS | `APP_ENV=production node scripts/seed.mjs` → `rc=1`, "Seed bị chặn ở production." |
| 15 | PASS | k6 `judge_burst` (script mới): `judge_done_ms` p95 **2.030 ms** (< 60.000), `judge_ie` 0, `http_req_failed` 0 % (315 yêu cầu) |
| 16 | PASS | k6 `autosave` (script mới, chỉ 2xx): p95 **35,7 ms** (< 150), max 291 ms, 0 lỗi / 27.066 yêu cầu |
| 17 | PASS (chấm lại, #18) | 3 cặp `chat` → `mixed` (stack thử `testroutes`, `FAKE_LLM_LATENCY=300-300`, `LLM_MAX_CONCURRENCY=100`, ~49.000 yêu cầu / lượt, 0 lỗi): chat p95 314,1 / 317,0 / 316,4 ms; mixed p95 316,7 / 316,5 / 313,0 ms → tỉ lệ **1,008 / 0,998 / 0,989**, **trung vị 0,998** (`scripts/ttft-ratio.mjs` rc=0); p95 `mixed` ≤ 317 ms (≤ 1,5 s); `run_ms` p95 1,03 / 1,53 / 2,05 s (≤ 5 s); `judge_done` p95 1,0–2,0 s, `judge_ie` 0. Log: `k6-pe09-b/` |
| 18 | PASS | `benchmarks/reports/pe-exam-submit-*.json` có ba kịch bản (dev); QC lưu `.log` |
| 19 | PASS | `Bắt đầu làm bài`: 27 lượt mới @17/s: p95 **12 ms**; 510 lần làm tiếp @17/s: p95 **12 ms** (≤ 300); `docker stats judge` rảnh **54 MiB** (≤ 100). `GET …/results` 1.000 dòng và thông lượng 240 bài: KHÔNG KIỂM ĐƯỢC (stack seed chỉ 60 tài khoản) |
| 20 | PASS | `exam.spec.ts` + toàn bộ Playwright: **443 pass, 0 fail** (không `@real`, `visual`); các ca dev phủ 12 ý AC7 |
| 21 | PASS (một phần) | ca (3) hai thiết bị, (4) mất mạng 30 s, (8) gửi phúc khảo, (11) xem điểm chạy tay bằng `playwright-cli` — xem TC-31; ca hết giờ khi đang gõ đã đo bằng API ở PE-06 |
| 22 | PASS | `bash scripts/gate-pe.sh` → 10 bước PASS, `GATE PE: PASS`, `rc=0` (`/tmp/qc5g/gate-ok.log`: vet → test race exam/judge/quiz → sandbox → scoring → leak → contract → sqlc → csv → antipatterns → Playwright 83 pass) |
| 23 | PASS | gieo `TestQCInjectedFailure` (`t.Fatal`) trong `internal/exam`: dừng ở "go test -race exam + judge + quiz" `FAIL`, **`rc=1`**, `GATE PE: FAIL`, các bước sau không chạy; xoá → `rc=0` |
| 24 | PASS | `grep -c 'GATE PE' handoff` = 4 |
| 25 | PASS | `<PageState>` trên `/questions`, `/exams`, `/exams/[id]`, `/exams/[id]/results`, `/take` (ảnh `01-05`, `17`); rỗng "Chưa có bài thi nào" + "Tạo bài thi"; `ui-antipatterns.sh` rc=0 |
| 26 | PASS (một phần) | ngắt mạng 30 s ở `/take` 375 px: dải "Mất mạng — bài vẫn được giữ trên máy bạn" → có mạng "Đã lưu lúc 19:57:22", 3 đáp án còn đủ, DB 2 câu (ảnh `09`, `10`); 3G chậm 400 kbps chưa đo |
| 27 | PASS (một phần, L1) | không thêm thư viện editor (`codemirror|monaco|ace|prism` = 0 trong `package.json`); LCP trong trang 72 / 92 ms; JS ≤ 250 KB / route: chưa đo |
| 28 | PASS | không chuỗi kỹ thuật hiển thị trong `features/exam`; UI: không `sandbox`, `verdict`, `RAG`… ở các màn chụp |
| 29 | PASS | `ui-antipatterns.sh` rc=0; `lint-selftest` trong cổng |
| 30 | PASS | `weekly_exam` trong `PROGRESS.md` = 2; đủ 6 nợ PE (máy chấm tách máy, precompiled header, checker tuỳ chỉnh, xoá `RUN`/`exam_events`, sổ điểm P6, chat riêng P3 + amd64 seccomp, `BlobReader`); `docs/sprints/5/report.md` do QC viết ở bước kế |
| 31 | PASS | **F19 chạy tay bằng `playwright-cli`** (ảnh `shots/pe09/01…24`): GV xem `/questions`, `/exams`, chi tiết; SV `sv.gioi` điện thoại 375 px (màn bắt đầu có câu minh bạch, làm câu đúng / sai + nhiều đáp án, **offline 30 s → online tự lưu**); máy tính 1440: "Bài đang mở ở nơi khác" → `Làm tiếp ở đây`, soạn mã C, `Chạy thử` "Đúng 2/2 test mẫu", `Nộp lời giải` "Lần nộp tính điểm", `Nộp bài` hộp xác nhận "Bạn đã trả lời 3/4 câu. Còn 1 câu chưa trả lời…"; đóng bài lúc 13:02:46 → **`PUBLISHED` lúc 13:02:50,8 (+4,5 s)**; SV thấy `9,00 / 10,00` + giải thích + "Test ẩn: đạt 2 trên 2"; SV gửi `Gửi yêu cầu xem lại`; GV tab "Xem lại điểm" chọn `Sửa điểm`, nhập `9,5` (dấu phẩy nhận 9.50), phản hồi → DB `ADJUSTED 9.00 → 9.50`; SV tải lại thấy `9,50 / 10,00` + "Điểm đã được giảng viên điều chỉnh." + phản hồi. Nhánh lỗi: bài chưa mở / TA không lên lịch / chat khoá / hết giờ / phúc khảo hết hạn: đã chấm ở PE-04…08 bằng API |

## Việc sau
- **QC:** TC-19 `GET …/results` 1.000 dòng + 240 bài (cần stack có nhiều tài khoản), `lhci` có đăng nhập, 3G chậm.
