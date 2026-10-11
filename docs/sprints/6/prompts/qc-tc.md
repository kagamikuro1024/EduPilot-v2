# Prompt cho `qc`: viết test case sprint 6 (hộp đen)

Worktree QC `/Users/kuro/Documents/TA_Agent_qc5`. Trước khi bắt đầu: `git fetch && git switch --detach origin/sprint/6-p3-p8`.

## Đọc
- `docs/team/QC.md`.
- `docs/sprints/6/plan.md`.
- Hai spec **APPROVED** (`FEAT-private-chat-pii`, `FEAT-docs-calendar`): đọc US, SRS, QUESTIONS và `TL-REVIEW.md` mục "Quyết định PM".
- `docs/sprints/6/proposals.md`.
- `docs/phases/P3.md`, `docs/phases/P8.md`: phần cổng nghiệm thu và "Bạn tự kiểm".
- `docs/FLOWS.md` F3, F6, F13.
- **Không đọc code của dev.**

## Việc
Viết test case trước khi có code:
- `docs/sprints/6/qc/tc-US-P3-01…08.md`, `tc-US-P8-01…03.md`;
- `gate-P3.md`, `gate-P8.md`.

Mỗi test case có các cột: `TC-id → AC → tiền điều kiện → bước/lệnh → kết quả mong đợi`. Không có dòng tiêu đề `TC-id` lẫn vào số đếm.

Phải có:
- **Tấn công dữ liệu cá nhân**, kiểu bộ E1 thu nhỏ:
  - tên có dấu / không dấu / đảo thứ tự, MSSV, email, SĐT, CCCD;
  - placeholder bị cắt giữa chừng;
  - hỏi hộ người khác;
  - prompt injection đòi tool cá nhân ở Threads.
- **Quét payload gửi provider giả:** 0 MSSV / họ tên roster.
- **Stream:** tải lại giữa chừng, nút Dừng, mạng 3G chậm, `OVERLOADED`.
- **Khoá giờ thi:** chat riêng và đăng thread mới bị khoá; thread cũ vẫn đọc được.
- **RAG:** `ANSWER_KEY` không bao giờ được truy xuất; `audience`; ingest lặp lại cùng `content_hash`; tệp scan tiếng Việt.
- **Lịch:**
  - ICS mở được bằng client lịch (lệnh `curl` + kiểm `BEGIN:VCALENDAR`);
  - token thô không lưu trong DB;
  - xoay token thì link cũ hết hiệu lực;
  - nhắc không gửi trùng.
- **Tên người đăng thread công khai với cả lớp** (chủ dự án chốt).
- **375 px** cho `/chat`, `/threads`, `/library`, `/calendar`.

Nghiệm thu tay bằng `playwright-cli` (global). Không chạy `playwright-cli install` trong repo.

Xong thì commit `sprint 6: QC test case P3/P8`, push, báo PM ≤ 5 dòng (số TC từng story). Câu hỏi về AC ghi `Q-QC-…` trong file TC, PM chuyển cho BA.
