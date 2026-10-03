# Prompt cho `research` — Chấm code C/C++ tự động cho bài thi hằng tuần

Worktree `/Users/kuro/Documents/TA_Agent_v2-s4`, nhánh `sprint/4-p2`; commit `docs/research/**` ở đó.

## Bối cảnh
Thầy hướng dẫn muốn thêm **bài kiểm tra lấy điểm hằng tuần** ngay trên hệ thống, hai loại: **trắc nghiệm** và **lập trình C/C++** (chấm tự động bằng bộ test ẩn, trả điểm). Tải: T1 = 1.000 SV / 20 lớp (`docs/SYSTEM_DESIGN.md`); kịch bản xấu nhất: 1 lớp 30–60 SV nộp dồn trong 5 phút cuối, nhiều lớp thi cùng khung giờ. D46 chốt chỉ Go + `docling-serve` → chấm code cần một sandbox, có thể phải mở lại D46.

## Câu hỏi (mỗi câu một mục trong cùng file `docs/research/2026-10-03-code-judge.md`)
1. **Sandbox biên dịch + chạy C/C++ không tin cậy**: so sánh `criyle/go-judge` (Go), Judge0 (self-host), `ioi/isolate`, nsjail, gVisor (`runsc`), Piston. Tiêu chí: cách ly (seccomp, cgroups v2, không mạng, giới hạn CPU/RAM/thời gian/output/số tiến trình/fork bomb), chạy được trong Docker Compose trên **colima macOS (dev)** và Linux VPS (demo), cần `--privileged` không, giấy phép, độ trưởng thành, API, thông lượng (bài/phút trên 4 CPU).
2. **Tích hợp với kiến trúc hiện có**: worker Go nhận bài qua Redis Streams (luật 12: 202 + job + SSE tiến độ), gọi sandbox qua HTTP; không lưu file lên đĩa gateway (luật 10) — mã nguồn trong DB/blob; idempotent khi chấm lại.
3. **Mô hình chấm**: bộ test ẩn/mẫu, so khớp đầu ra (chính xác / bỏ khoảng trắng / checker tuỳ chỉnh), điểm theo % test đạt, giới hạn thời gian/bộ nhớ theo bài, phân loại kết quả (CE, WA, TLE, MLE, RE, AC), trình biên dịch + cờ (gcc/g++ phiên bản, `-O2 -std=c11/c++17`).
4. **Liêm chính tối thiểu khả thi**: phát hiện code giống nhau giữa bài nộp (MOSS, JPlag, hay tự viết so khớp token/AST — có thư viện Go?), ghi log rời tab/dán, khoá chat AI trong giờ thi (P9 L2b đã có `quiz_lock`).
5. **PoC**: dựng phương án đề xuất trong `/tmp` (hoặc `docs/research/poc/code-judge/`) trên colima, chấm thử 3 bài: AC, TLE (vòng lặp vô hạn), fork bomb / đọc file hệ thống → phải bị chặn. Ghi lệnh + kết quả thật + thời gian chấm.

Kết luận đầu file: phương án chọn, có phải mở lại D46 không (thêm 1 container sandbox), rủi ro, việc cần chủ dự án quyết. Không mở subagent. Commit `research: chấm code C/C++`, push, tóm tắt ≤ 10 dòng, dừng.
