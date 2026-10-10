# Sprint 5 — báo cáo (PE Thi hằng tuần + US-PU-06)

Mục tiêu: trọn `docs/phases/PE.md`. Giảng viên soạn ngân hàng câu hỏi (trắc nghiệm + bài code C/C++ có bộ test), lên lịch bài thi; sinh viên làm bài có đồng hồ, tự lưu; máy chấm (Quiz Engine + sandbox `go-judge`); tự công bố khi bài đóng; liêm chính theo D56. Kèm US-PU-06: trả nợ LCP / TBT.

**Kết quả: 10/10 story PASS (có điều kiện) · cổng PE ĐẠT CÓ ĐIỀU KIỆN** sau 1 vòng sửa cổng. Nhánh `sprint/5-pe`.

| Story | QC (TC) | Kết quả |
| --- | --- | --- |
| US-PU-06 Khung trang vẽ từ máy chủ; LCP (lượt devtools) và TBT về `error` | `qc/report-US-PU-06.md` (20) | PASS có điều kiện, sau sửa |
| US-PE-01 Migration `00006_weekly_exam` (13 bảng, FK phức hợp), Quiz Engine, chấm điểm `decimal` | `qc/report-US-PE-01.md` (40) | PASS sau sửa |
| US-PE-02 Sandbox `go-judge`, hàng chấm kiểu outbox (thuê / thử lại), 15 ca tấn công | `qc/report-US-PE-02.md` (57) | PASS có điều kiện |
| US-PE-03 Ngân hàng câu hỏi: soạn, nhập test / zip, chạy lời giải mẫu, duyệt, gợi ý AI nháp | `qc/report-US-PE-03.md` (55) | PASS sau sửa |
| US-PE-04 Bài thi: tạo, lên lịch, gia hạn, xem trước, nav "Bài thi" | `qc/report-US-PE-04.md` (50) | PASS có điều kiện |
| US-PE-05 Làm bài trắc nghiệm: đồng hồ máy chủ, xáo trộn, tự lưu, một nơi ghi, tự nộp | `qc/report-US-PE-05.md` (60) | PASS có điều kiện |
| US-PE-06 Làm bài code: nháp, chạy thử test mẫu, nộp, lịch sử, SSE | `qc/report-US-PE-06.md` (47) | PASS có điều kiện |
| US-PE-07 Liêm chính: khoá chat, log rời tab / dán, so độ giống (winnowing) | `qc/report-US-PE-07.md` (48) | PASS có điều kiện, sau sửa |
| US-PE-08 Chấm xong → tự công bố, kết quả, sửa điểm, chấm lại, phúc khảo, `TestNoAnswerLeak` | `qc/report-US-PE-08.md` (68) | PASS có điều kiện |
| US-PE-09 Seed bài mẫu, bảng điểm tính tay, k6 `exam-submit`, `gate-pe.sh` | `qc/report-US-PE-09.md` (31) | PASS có điều kiện, sau sửa |

- Cổng: `qc/report-GATE-PE.md` (24 mục).
- CI xanh ở HEAD (`bc09dd2`): Go, Frontend, Judge attacks (amd64, seccomp bật).
- Spec: `docs/specs/FEAT-weekly-exam/` v1.9, `docs/specs/FEAT-ui-foundation/` v1.6.

"Có điều kiện" nghĩa là không có lỗi chức năng; còn mục chưa kiểm được trên máy QC hoặc chờ phase sau (xem Nợ).

## Số liệu
- **Test case:** 500 (story 476, cổng 24). Lỗi thật vòng 1: 7, đã sửa hết. (Sửa 2026-10-10: bản trước ghi 510 vì đếm cả dòng tiêu đề bảng `TC-id`.)
- **Tấn công:**
  - 15 ca A1–A15 trong `TestSandboxAttacks` cộng 20 chương trình tấn công QC tự viết;
  - fork bomb không tạo được tiến trình thứ hai (`procPeak = 1`);
  - không ra mạng, không đọc file hệ thống;
  - seccomp bật trên amd64 qua CI.
- **Chống rò đáp án:** `TestNoAnswerLeak` (20 × 6 tổ hợp) xanh; sinh viên chỉ thấy test ẩn ở dạng "đạt X trên Y".
- **k6 `judge_burst`:** 60 bài code nộp rải trong 5 phút, `judge_done` p95 ≈ 1,0 s.
- **k6 `mixed`:** chat chạy song song với hàng chấm không chậm đi; trung vị tỉ lệ p95 là 0,998, p95 khoảng 315 ms.
- **Lighthouse:**
  - LCP lượt devtools 1,4–1,9 s (ngưỡng 2,5 s);
  - TBT 6 route ≤ 200 ms (mô phỏng);
  - JS ≤ 256 KB.
- **Khối lượng:** 115 commit; `backend-go` + `frontend` +47.371 dòng.
- **Góp ý:** #1–#19, PM quyết cả 19. Có 1 lần PM đổi đề xuất (#17c) và 1 lần PM tự đề xuất (#18).
- **Tech Lead:**
  - review spec với 11 ý (1 chặn);
  - trả lời 4 câu kỹ thuật (TL-1…4).

## Lỗi thật tìm ra và đã sửa
- **Làm tròn điểm hai lần:** khi `max_score` không nguyên, 166 trên 20.000 ca lệch so với tính bằng `big.Rat` (PE-01). Đã sửa để chỉ làm tròn một lần ở cuối, có test vi sai.
- **Ký tự NUL:** chuỗi chứa ký tự NUL làm Postgres trả 500 (PE-03). Đã chặn ở bộ đọc JSON chung, trả 422.
- **Ngưỡng nghi chép bài:** lớp nhỏ có bài chép mà không bị gắn cờ (PE-07, #17c). Đã thêm trần `SIMILARITY_CAP` 0,9.
- **Hàng chấm (TL review, trước khi viết migration):** cơ chế cũ có thể chấm một bài hai lần và đánh nhầm bài thành `ERROR` (TLR-1). Đã đổi sang thuê việc kiểu outbox với các cột `lease_until`, `next_attempt_at`, `fail_count`.
- **Header cache:** 6 route trả `s-maxage` (PU-06).
- **TBT trang `/gradebook`:** có hai đỉnh, vượt ngưỡng (PU-06).
- **CI chập chờn:**
  - test đo thời gian chạy trên runner dùng chung (TL-3, nay chạy riêng, tuần tự);
  - A8 ra `MLE` trên amd64 (TL-4 / #19: kiểm bất biến `procPeak = 1` thay vì nhãn verdict).

## Quyết định trong sprint
- **D58:** `go-judge` v1.13.0 chạy chung máy, mạng Docker riêng. Một consumer judge duy nhất (`JUDGE_CONSUMER`). Trần `memory_limit_mb` 512.
- **#14:** LCP chấm ở lượt Lighthouse devtools. `font-display: block` được dùng, có điều kiện.
- **Chủ dự án chốt trong sprint:**
  - bài code nộp nhiều lần, lần cuối tính điểm;
  - sau công bố, test ẩn chỉ hiện số đạt / tổng;
  - nhiều đáp án chấm điểm từng phần;
  - chấm lại sau công bố được phép làm giảm điểm (có lý do);
  - TA được duyệt câu hỏi.
- **Hạ tầng:**
  - Repo chuyển sang **`EduPilot-v2` (public)**, lịch sử đã làm sạch secret của Project III. Lý do: CI của repo private hết phút miễn phí.
  - CI có `concurrency` và bỏ qua commit chỉ sửa tài liệu.
- **Đội:** research kiêm Tech Lead, thẩm định spec của BA. QC dùng `playwright-cli` để nghiệm thu.

## Nợ
- Tách máy chạy sandbox khỏi máy gateway (PR, D58).
- Precompiled header cho `<bits/stdc++.h>`.
- Checker tuỳ chỉnh (testlib).
- Nối điểm bài thi vào sổ điểm (P6).
- Chat riêng tôn trọng khoá giờ thi (P3).
- Retention cho `RUN` / `exam_events` (PR).
- QC chưa chạy: `audit.mjs` / `sweep.mjs` lượt này; bảng kết quả 1.000 dòng + 240 bài; Lighthouse có đăng nhập ở mạng 3G chậm; điện thoại thật.
- Phép đo token độ giống của QC lệch 0,039 so với hệ thống (định nghĩa token khác nhau).

## Chủ dự án tự kiểm
- `SEED_ON_EMPTY_DB=true pnpm dev`. Đăng nhập giảng viên, vào `/exams`: có 2 bài mẫu. Lên lịch một bài ngắn.
- Đăng nhập sinh viên, làm bài trắc nghiệm (thử tắt mạng giữa chừng, bật lại) và bài code C++. Trong giờ thi, chat AI bị khoá.
- Sau giờ đóng: điểm tự công bố; sinh viên thấy đáp án trắc nghiệm, còn test ẩn chỉ thấy "đạt X trên Y".
- Giảng viên xem `/exams/[id]/results` và trang nghi giống nhau.
