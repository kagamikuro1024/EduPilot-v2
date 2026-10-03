# QC test case — US-PE-09 (seed bài thi mẫu, k6 `exam-submit.js`, E2E `exam.spec.ts`, script cổng PE, cổng UX, bàn giao)
Nguồn: `docs/specs/FEAT-weekly-exam/US.md` US-PE-09 AC1–AC10 + `SRS.md` 4.11 (seed), 8.3 (SLO), 9 (kiểm thử), `docs/phases/PE.md` ("Cổng nghiệm thu", "Bạn tự kiểm"), `FLOWS.md` F19, `SYSTEM_DESIGN.md` §5, `UX.md` mục 6. Hộp đen; stack như `tc-US-PE-02.md` (đầy đủ: Postgres, Redis, Mailpit, 2 gateway, worker, judge, Caddy hoặc cổng thẳng). QC **tự tính tay** điểm của 30 lượt mẫu bằng Python `fractions` (`scripts/p509-expected.py`) **trước khi** mở `seed/expected_exam_scores.csv` của dev để đối chiếu; QC tự viết kịch bản k6 riêng (`scripts/p509-k6/*.js`) bên cạnh kịch bản của dev.

Tiền điều kiện chung: stack local trống (`down -v` → `up`); `SEED_DEFAULT_PASSWORD`; `node scripts/seed.mjs`. Công cụ: **S** shell, **D** SQL, **A** Chrome, **G** `go test`, **K** k6, **P** Python.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-PE09-01 | AC1 (ngân hàng) | stack trống | **S** `node scripts/seed.mjs`; **D** `select type,review_status,origin,count(*) from question_bank where course_id='$C1' group by 1,2,3 order by 1,2,3` | 20 câu `APPROVED` (12 `MCQ_SINGLE`, 4 `MCQ_MULTI`, 4 `TRUE_FALSE`) `MANUAL`; 5 câu `AI_DRAFT` `PENDING`; 2 bài `CODE` `APPROVED`; **lớp 2: 0 dòng** |
| TC-PE09-02 | AC1 (chi tiết) | – | **D** 5 chủ đề An ninh mạng; độ khó đủ 3 mức; mỗi câu có `explanation`; bài code: 2 mẫu + 3 ẩn, trọng số `1/1/2/2/2` và `1/1/1/1/1`; `reference_verified_version = tests_version`; một bài `c11`+`cpp17`, một bài `cpp17`; đề tiếng Việt | khớp bảng SRS 4.11 từng ô; `explanation` không rỗng ở 20/20 câu |
| TC-PE09-03 | AC1 | – | **S** `node scripts/check-exam-seed.mjs bank` | `OK` |
| TC-PE09-04 | AC2 (qua API thật) | – | **S** `grep -n '_test/' scripts/seed.mjs \| wc -l`; `grep -nE 'INSERT|psql|pg\.Client|postgres://' scripts/seed.mjs`; quan sát log HTTP gateway lúc seed (QC bật `curl`-trace hoặc xem access log) | `0`; không ghi DB trực tiếp; seed gọi đúng chuỗi: tạo câu → verify → duyệt → tạo bài → chọn câu → lên lịch → SV bắt đầu / lưu / nộp (đúng luồng F19) |
| TC-PE09-05 | AC2 (hai bài thi, cửa sổ ngắn) | – | **D** `select title,kind,status,duration_minutes from exams where course_id='$C1'`; cấu hình `EXAM_MIN_DURATION_MINUTES`, `EXAM_MIN_LEAD_SECONDS` ở compose; `grep` các biến này ở file production | 2 bài: "Kiểm tra tuần 9 — Mật mã" (`MCQ`, 10 câu, 45 phút) và "Kiểm tra tuần 9 — Lập trình" (`CODE`, 2 bài × 5 test, 60 phút); biến cửa sổ ngắn **chỉ** ở `docker-compose.local.yml` / `docker-compose.test-seed.yml`, **không** ở production (`docker-compose.prod*`/VPS) |
| TC-PE09-06 | AC2 (đóng / công bố ≤ 3 phút) | seed xong | **D** poll `status` mỗi 5 s sau khi `seed.mjs` thoát | ≤ **3 phút** sau seed cả hai bài `PUBLISHED`; `seed.mjs` **không** chờ đóng bài (thoát trước) |
| TC-PE09-07 | AC2 (lượt mẫu) | – | **D** số `exam_attempts` của bài trắc nghiệm = 24, bài code = 6; SV A 10/10, B 8/10, C không có lượt, D không thuộc lớp; 6 bài code với kết quả khác nhau (AC đủ, một phần, `CE`, `TLE`, `WA`) + 1 cặp giống nhau (đổi tên biến) | khớp; mẫu đáp án **cố định** (chạy seed hai lần ở DB mới cho cùng điểm) |
| TC-PE09-08 | AC3 (**điểm khớp tay**) | bài `PUBLISHED` | **P** QC tự dựng bảng điểm 30 lượt từ **mẫu đáp án** của seed (đọc `scripts/seed-data/*.json` hoặc đáp án thực tế trong `exam_answers`) bằng `fractions` theo công thức SRS 4.6; so với `auto_score` và với `seed/expected_exam_scores.csv`; **S** `node scripts/check-exam-seed.mjs scores` | điểm máy khớp **từng dòng** cả hai bảng; `expected_verdict` khớp verdict từng bài code; in `matched 30/30`, thoát `0`; (nếu bảng của dev và của QC lệch nhau → **FAIL**, ghi dòng lệch) |
| TC-PE09-09 | AC3 (mặc định `PARTIAL`) | – | **D** `multi_scoring` của bài trắc nghiệm | `PARTIAL`; câu `MCQ_MULTI` trong mẫu có ít nhất 1 lượt chọn một phần (chứng minh điểm một phần có mặt) |
| TC-PE09-10 | AC4 (trạng thái trình diễn) | seed xong | **S** `node scripts/check-exam-seed.mjs demo`; QC tự gọi API với từng tài khoản: SV A, SV C, SV D, GV, TA, SV có `CE` | **A**: điểm cao nhất + đáp án; **C**: "Bạn không làm bài này"; **D**: không bài thi nào; **GV**: bảng điểm, phân bố, `Câu sai nhiều` có ≥ 1 câu đúng ≤ 40 %, tab `Nghi giống nhau` có **đúng 1** cặp `flagged` (cặp gieo) + việc `EXAM_SIMILARITY` ở "Hôm nay"; **TA**: bảng điểm, **không** `Tín hiệu`; SV có `CE` thấy nhãn "Lỗi biên dịch" ở kết quả của chính họ |
| TC-PE09-11 | AC4 (giao diện) | – | **A** đăng nhập từng tài khoản trên Chrome thật, chụp `/exams`, `/exams/[id]/take`, `/exams/[id]/results`, "Hôm nay"; `$PW exam.spec.ts -g 'seed demo state' --grep @real` | ảnh vào `shots/` của report; `rc=0` |
| TC-PE09-12 | AC5 (idempotent) | seed đã chạy | **S** `time node scripts/seed.mjs` lần hai; đếm bảng trước / sau; thư Mailpit; tài khoản, lớp | `Seed xong (không đổi)`; `exams` của lớp 1 = **2**, câu / lượt không tăng; số tài khoản / lớp / thư Mailpit của P2 **không đổi** |
| TC-PE09-13 | AC5 (thời gian) | DB trống | **S** `time` toàn bộ (P2 + PE) tính từ `readyz` | ≤ **5 phút**; ghi số đo |
| TC-PE09-14 | AC5 (production) | – | **S** `APP_ENV=production node scripts/seed.mjs; echo rc=$?` | `rc=1` **trước** mọi lời gọi (không request nào tới gateway) |
| TC-PE09-15 | AC6 (k6 `judge_burst`) | `JUDGE_PARALLELISM=2`, judge thật | **K** `k6 run benchmarks/load/exam-submit.js --env SCENARIO=judge_burst`; **và** kịch bản QC riêng (60 SV × 1 bài `<bits/stdc++.h>` 10 test, rải 5 phút, theo dõi `DONE` từ DB) | `judge_done_p95 < 60000`, `judge_ie == 0`, `http_req_failed < 0,5 %`; số QC tự đo (từ nộp tới `DONE` bằng `code_submissions.created_at`/`updated_at`) **thắng** nếu lệch; ghi cấu hình máy |
| TC-PE09-16 | AC6 (`autosave`) | 300 phiên | **K** `--env SCENARIO=autosave` và kịch bản QC riêng (300 VU lưu mỗi 2 s trong 3 phút) | `autosave_p95 < 150` ms, lỗi < 0,5 %; `5xx = 0` |
| TC-PE09-17 | AC6 (`mixed`: không làm chat chậm) | 50 luồng chat `fake` 5–15 s | **K** `--env SCENARIO=mixed` và QC đo TTFT chat **riêng** vs **cùng** `judge_burst` (tự đo từ `llm_audit.queue_wait_ms` / thời gian phản hồi) | TTFT INTERACTIVE không chậm hơn **+20 %**; `Chạy thử` p95 ≤ 5 s; judge và LLM **không chung làn** |
| TC-PE09-18 | AC6 (báo cáo) | – | **S** `ls benchmarks/reports/pe-exam-submit-*.json`; `jq` thử | có tệp, có số `p95` ba kịch bản |
| TC-PE09-19 | AC6 (SLO bổ sung SRS 8.3) | – | **K/S** `Bắt đầu làm bài` p95 ≤ 300 ms (1.000 SV / phút ≈ 17/s); `GET …/results` 1.000 dòng p95 ≤ 300 ms; `docker stats judge` lúc rảnh ≤ 100 MiB; thông lượng 240 bài / 5 phút hết ≤ 8 phút | đạt từng số (ghi đo thật; vượt → **FAIL**, không nới) |
| TC-PE09-20 | AC7 (E2E) | – | **S** `pnpm -C frontend exec playwright test exam.spec.ts` (không `@real`), rồi `--grep @real` trên stack seed; QC **đếm ca** và đối chiếu 12 ý của AC7 (1…12) | ≥ 10 ca phủ đủ 12 ý (soạn + zip + verify + duyệt; dựng bài + lỗi lên lịch + TA không lên lịch; trắc nghiệm 375 px + nộp có số; mất mạng 30 s; hết giờ tự nộp; hai tab; code chạy thử / nộp / CE / nộp lại; hết giờ khi đang gõ; `chat-gate`; sau đóng thấy điểm + đáp án không test ẩn; rò đáp án quét mạng bằng canary; phúc khảo); pass 100 % |
| TC-PE09-21 | AC7 (độc lập) | – | **A** QC chạy **tay** ca (3), (4), (8), (11) trên Chrome thật (không dùng spec của dev) | cùng kết quả như spec; ghi ảnh / số |
| TC-PE09-22 | AC8 (`gate-pe.sh`) | stack | **S** `bash scripts/gate-pe.sh; echo rc=$?` | thứ tự `go vet` → `go test -race ./internal/exam/... ./internal/judge/... ./internal/quiz/...` → `TestSandboxAttacks` → `TestScoringDecimal` → `TestNoAnswerLeak` → `go test ./internal/contract/...` → `sqlc diff` → Playwright `exam.spec.ts` → `ui-antipatterns.sh` → k6 ba kịch bản; bảng `PASS`/`FAIL` + thời gian; `rc=0` và dòng cuối `GATE PE: PASS` |
| TC-PE09-23 | AC8 (dừng ở lỗi đầu) | – | **S** gieo lỗi tạm (làm một test `exam` đỏ) rồi chạy `gate-pe.sh`; hoàn tác | dừng ở lỗi đầu, `rc≠0`, bảng ghi `FAIL`; hoàn tác → `rc=0` |
| TC-PE09-24 | AC8 (bàn giao) | – | **S** `grep -c 'GATE PE' docs/sprints/5/handoff/dev-US-PE-09.md` | ≥ 1 (kết quả dán vào handoff) |
| TC-PE09-25 | AC9 (cổng UX — mọi màn) | 5 route | **A** với `/questions`, `/exams`, `/exams/[id]`, `/exams/[id]/results`, `/exams/[id]/take`: `<PageState>` đủ tải / rỗng / lỗi; `grep -rnE 'fetch\(|alert\(|confirm\(' frontend/src/features/exam* \| grep -v shared` ; bàn phím đi hết luồng chính; axe; 4 mốc bề rộng (1440, 1024, 720/719, 375/390); `AUDIT_SRC`, `TOUCH_SRC` | đủ ba trạng thái; 0 `fetch`/spinner/confirm riêng; luồng chính chỉ bàn phím; 0 `serious`/`critical`; **375 px không tràn ngang** ở `/take` (trắc nghiệm) và màn kết quả SV |
| TC-PE09-26 | AC9 (3G + ngắt) | – | **A** CDP throttle 3G chậm (400 kbps, RTT 400) + ngắt giữa bài 30 s ở `/take` | không mất dữ liệu (xem PE05-40); trang vẫn dùng được |
| TC-PE09-27 | AC9 (hiệu năng) | build thật | **S** `lhci autorun --collect.url=…/exams/<id>/take` (mobile); `next build` bảng JS mỗi route | LCP mobile ≤ **2,5 s** (khung nhìn đầu render sẵn); JS mỗi route ≤ **250 KB gzip**; không thêm thư viện editor (`package.json` không thêm codemirror/monaco…) — **lưu ý** LCP toàn dự án đang "PASS có điều kiện" (#26): QC ghi số và dùng quy tắc chung đã chốt |
| TC-PE09-28 | AC9 (từ kỹ thuật) | – | **S** `grep -rnE 'sandbox\|verdict\|go-judge\|provider\|trace' frontend/src/features/exam* \| grep -v '//'`; **A** quét `innerText` mọi màn SV | không chuỗi hiển thị; 0 từ kỹ thuật ở màn SV |
| TC-PE09-29 | AC9 | – | **S** `bash scripts/ui-antipatterns.sh; echo rc=$?`; `--selftest` | `rc=0`; 19 / 19 |
| TC-PE09-30 | AC10 (bàn giao) | – | **S** `grep -c 'weekly_exam' docs/PROGRESS.md`; `test -f docs/sprints/5/report.md`; đọc mục nợ: sandbox tách máy (PR), precompiled header, checker tuỳ chỉnh, xoá `RUN` cũ + `exam_events` theo hạn (PR), nối sổ điểm (P6), chat tôn trọng khoá (P3) | ≥ 1; tệp có; đủ 6 nợ |
| TC-PE09-31 | tổng (F19 trọn) | stack seed | **S** theo `tc-US-PE-08` TC-PE08-67 | F19 đi từ đầu đến cuối, gồm các nhánh lỗi liệt kê trong `FLOWS.md` F19 |

## Nhánh lỗi / biên đã phủ
| Tình huống | TC |
| --- | --- |
| Seed ghi thẳng DB / dùng route `_test` / chạy ở production | 04, 14 |
| Seed chạy lần hai làm trùng dữ liệu hoặc phá P2 | 12 |
| Bảng điểm mong đợi của dev sai (cùng người viết và so) | 08 (QC tự tính độc lập) |
| Chấm dồn làm chat chậm | 17 |
| SLO đo bằng số của công cụ khác số thật | 15, 16 |
| Cổng PE xanh giả (không dừng ở lỗi) | 23 |
| Màn làm bài vỡ ở 375 px / 3G | 25, 26 |

## Câu hỏi cho BA / PM
- **Q-QC-PE09-1** — AC3 do dev tự tính `expected_exam_scores.csv`; QC tự tính độc lập (TC-PE09-08) từ mẫu đáp án thực tế của seed. Nếu hai bảng lệch, QC coi là FAIL và nêu dòng lệch. Đúng? — *chờ BA*.
- **Q-QC-PE09-2** — AC9 LCP ≤ 2,5 s: dự án đang áp dụng quyết định #26 ("PASS có điều kiện", LCP > 2,5 s trên máy dev). Với `/exams/[id]/take` QC chấm theo cùng quy tắc (ghi số, không FAIL riêng)? — *chờ PM*.
- **Q-QC-PE09-3** — AC6 `mixed` "TTFT INTERACTIVE +20 %": QC đo TTFT bằng thời gian tới byte đầu của `_test/llm/chat` (không streaming); chấp nhận? — *chờ BA*.

## Lịch sử sửa TC
- 2026-10-03 — viết lần đầu theo US.md v1.1 (FEAT-weekly-exam, APPROVED).

Tổng: 31 TC.
