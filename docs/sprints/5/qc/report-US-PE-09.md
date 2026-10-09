# Báo cáo QC — US-PE-09 (seed bài thi mẫu, k6, cổng PE) — **DỞ DANG, chưa có kết luận**
Dừng theo lệnh PM (chủ dự án vắng). Bản chấm `8d83087`. Stack thật `EP_PORT_OFFSET=100` + seed tự động (74 s); `.env.local` của QC đặt `EXAM_MIN_DURATION_MINUTES=1`, `EXAM_MIN_LEAD_SECONDS=5`, `EXAM_SAVE_RATE_PER_MIN=2000`, `LLM_DEFAULT_RPM/TPM` cao (như `docker-compose.test-seed.yml`). Đã tắt stack, `docker volume prune -f`.

## Đã đo (bằng chứng thô: `k6-pe09/*.log`, `scripts/p509-score.py`)
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| 01, 03 | PASS | `check-exam-seed.mjs bank` → `OK`; DB: 12 `MCQ_SINGLE` + 4 `MCQ_MULTI` + 4 `TRUE_FALSE` `APPROVED` `MANUAL`, 5 `AI_DRAFT` `PENDING`, 2 `CODE` `APPROVED` |
| 04 | PASS | `grep -c '_test/' scripts/seed.mjs` = 0; không INSERT / psql (chỉ chú thích) |
| 05, 06, 07 | PASS (một phần) | 2 bài (`MCQ` / `CODE`, 2 phút), sau ~3 phút `PUBLISHED`; 24 lượt trắc nghiệm + 6 lượt code |
| 08 | PASS | QC tự chấm 24 lượt MCQ bằng `fractions` từ `exam_answers` + `answer_key` (`p509-score.py`): **0 lệch**, 4 lượt có điểm `MCQ_MULTI` một phần; `check-exam-seed.mjs scores` → `matched 30/30` (mô hình của dev, đã chạy; code 6 lượt theo mô hình dev) |
| 09 | PASS | `multi_scoring = PARTIAL`, có lượt chọn một phần |
| 10 | PASS (API) | `check-exam-seed.mjs demo` → `OK` (UI chưa chụp) |
| 12, 13 | PASS | seed lần hai **12 s**; đếm trước / sau y hệt (2 bài thi, 27 câu, 30 lượt, 60 users, 61 enrollments, 11 bản nộp; Mailpit 59 → 59); lần đầu 74 s (≤ 5 phút) |
| 14 | PASS | `APP_ENV=production node scripts/seed.mjs` → `rc=1`, "Seed bị chặn ở production." |
| 15 | PASS | k6 `judge_burst`: `judge_done_ms` p95 **1.040 ms** (< 60.000), `judge_ie` 0, `http_req_failed` 0 % (269 request); kịch bản QC riêng 60 SV: **chưa chạy** |
| 16 | PASS | k6 `autosave`: p95 **67 ms** (< 150), max 220, 0 lỗi / 27.066 request; kịch bản QC 300 VU riêng: chưa chạy |
| 17 | **FAIL** (lần đo 1) | k6 `mixed` (`CHAT_BASE_P95` = 10,12 ms từ `chat`): TTFT chat p95 **12,444 ms** > ngưỡng 12,144 ms (cơ sở +20 %) → `chat_ttft_ms` threshold crossed, `rc=99`; `run_ms` p95 2.624 ms (≤ 5.000 đạt), `judge_done` p95 2.133 ms, `judge_ie` 0. Chênh 0,3 ms trên cơ sở ~10 ms (provider `fake`, ~5 ms): có thể là nhiễu — đang đo lặp thì bị dừng (chưa có số lần 2, 3). Không nới ngưỡng. |
| 18 | PASS (một phần) | báo cáo `benchmarks/reports/pe-exam-submit-*.json` có tệp cho 3 kịch bản (dev); QC dùng `.log` đính kèm |
| 19 | PASS (một phần) | `docker stats judge` lúc rảnh **54 MiB** (≤ 100); `Bắt đầu làm bài` p95, `GET …/results` 1.000 dòng, thông lượng 240 bài: chưa đo |
| 24 | PASS | `grep -c 'GATE PE' handoff` = 4 |
| 27, 28 | PASS (một phần) | không thêm thư viện editor (`codemirror|monaco|ace|prism` trong `package.json` = 0); không chuỗi kỹ thuật hiển thị trong `features/exam`; `lhci` chưa chạy |
| 30 | PASS | `grep -c weekly_exam docs/PROGRESS.md` = 2; nợ PE liệt kê đủ 6 mục + thêm; **`docs/sprints/5/report.md` chưa có** (QC viết ở cuối sprint) |

## Chưa chạy (làm tiếp tối nay)
- TC-02 (trọng số / `reference_verified_version` từng ô SRS 4.11), TC-11 (UI Chrome thật + ảnh), TC-17 **lặp lại** (chat ×2, mixed ×2 để xem nhiễu), TC-19 (SLO bổ sung), TC-20, 21 (ca 3, 4, 8, 11 tay), TC-22, 23 (`gate-pe.sh` + gieo lỗi — chạy khi stack tắt), TC-25–29 (cổng UX: PageState, 3G + ngắt, `lhci`, JS ≤ 250 KB), TC-31 (F19 trọn vẹn bằng `playwright-cli` + ảnh, gom cả UI các story trước).
- Sau đó: `gate-PE.md` → `report-GATE-PE.md`, rồi `docs/sprints/5/report.md`.

## Nghi vấn
- **TC-17** (xem trên): nếu lặp lại vẫn > +20 % thì FAIL thật, cần dev xem hàng đợi `INTERACTIVE` dưới tải judge.
