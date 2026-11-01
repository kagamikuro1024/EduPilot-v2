# Prompt cho `qc`: smoke hộp đen trên VPS (OPS-VPS)

URL `https://v2202609419695521098.happysrv.de`, tài khoản seed (`docs/sprints/ops-vps/handoff/dev-deploy.md`). Bằng `playwright-cli` (1440 và 375 px), đi các luồng chính của `docs/DEMO_SCRIPT.md` bước 1 → 7 trên stack thật với LLM thật:
- đăng nhập 4 vai;
- chat riêng + Threads;
- nhờ giảng viên → `/inbox` trả lời;
- điểm danh;
- sổ điểm + giải trình;
- bài tập: nộp trong app (tải tệp thật qua URL ký sẵn) → chấm → duyệt → công bố → sinh viên thấy điểm;
- bài thi C++.

Kiểm chéo:
- HTTPS hợp lệ;
- không cổng nội bộ nào mở ra ngoài (`nmap -Pn -p 1-65535 152.53.64.175` hoặc ít nhất 5432, 6379, 9000, 9001, 8025, 8080, 3000, 20128);
- không có nhãn "Bản mô phỏng";
- không lỗi console;
- LCP / TTFB thật từ Việt Nam: ghi số, không chấm FAIL.

Ghi `docs/sprints/ops-vps/qc/report-smoke.md` (PASS / FAIL từng luồng, ảnh vào `docs/sprints/ops-vps/qc/shots/`). Báo PM ≤ 4 dòng.
