# Prompt cho `dev`: deploy EduPilot lên VPS (OPS-VPS)

Đọc `docs/sprints/ops-vps/plan.md` trước. Trong worktree dev: `git fetch && git switch -c ops/vps-deploy origin/main`; commit + push `ops/vps-deploy`. Trên VPS clone repo công khai vào `/opt/edupilot` (pull nhánh `ops/vps-deploy`).

## Mục tiêu
`https://v2202609419695521098.happysrv.de` chạy đủ stack như `pnpm dev` (gateway, worker, frontend, Postgres + pgvector, PgBouncer, Redis, MinIO, Caddy, judge, docling profile `ingest`, mailpit, mock-imap, mock-graph). Seed demo chạy được (`scripts/seed.mjs` bằng API thật). LLM chat thật qua 9router.

## Cách làm (chọn lối ít mã nhất)
- **Một tệp override** cho VPS, ví dụ `deploy/vps.compose.yml` dùng chồng lên `docker-compose.local.yml`, cùng một `deploy/README.md` ngắn (lệnh dựng / cập nhật / seed / xem log). Không viết lại compose; không thêm service mới.
- **Secret:** chỉ ở `/opt/edupilot/.env` trên VPS (`chmod 600`), sinh ngẫu nhiên (Postgres, Redis, MinIO, JWT, khoá mã hoá…). Khoá 9router đọc trực tiếp từ sqlite vào tệp đó, **không echo**. Repo chỉ có `.env.example` / mẫu, không giá trị thật.
- **Mạng:** chỉ Caddy mở 80/443 ra ngoài; Postgres, Redis, MinIO, gateway, mailpit **không publish cổng công khai** (bind `127.0.0.1` nếu cần, xem qua SSH tunnel). Caddy tự cấp HTTPS cho hostname trên; nếu tường lửa netcup chặn 80/443 thì báo PM ngay (chủ dự án mở trong SCP).
- **URL ký sẵn MinIO** phải mở được từ trình duyệt ngoài (đặt `BLOB_PUBLIC_ENDPOINT` qua Caddy). Thử tải tài liệu và nộp bài thật.
- **LLM:** `LLM_PROVIDER`, `DEMO_MODE`, `LLM_DEMO_FILE` **tắt** chế độ giả cho chat. Khai provider `openai_compatible` base URL `http://9router:20128/v1`, model `ag/gemini-3.8-flash` cho mọi task sinh chữ (CHAT, CLASSIFY, UTILITY, GRADING, INSIGHT…), embedding giữ provider giả hiện có. Gắn `gateway` và `worker` vào network ngoài `9router_default`. Cấu hình bằng cách hệ thống đã có (env / seed / `/settings/llm`), ghi rõ trong `deploy/README.md`. `LLM_DEFAULT_RPM` vừa phải (vd. 20).
- **Judge trên amd64 có seccomp** (nợ PE (7)): lần đầu chạy thật; bỏ `-no-seccomp`. Chấm thử một bài C++ của seed; lỗi thì sửa / ghi rõ.
- **Build trên VPS** (12 vCPU / 32 GB đủ). Image docling lớn: build một lần.

## Kiểm (smoke, ghi kết quả thật vào handoff)
1. `curl -I https://v2202609419695521098.happysrv.de` 200 + chứng chỉ hợp lệ; trang login mở được.
2. Seed chạy rc 0 trên DB trống; đăng nhập `teacher@edupilot.local` / `sv.kha@edupilot.local`.
3. Chat riêng sinh viên hỏi một câu về tài liệu lớp → câu trả lời **từ Gemini thật** (`llm_audit` có dòng provider 9router, model `gemini-3.8-flash`, che danh tính > 0 khi câu có tên).
4. Trích công thức điểm từ quy chế lớp 2 bằng LLM thật → bản nháp ra (đúng / sai không quan trọng ở bước này; Tech Lead đo).
5. "Chấm tất cả" BT03 bằng LLM thật → có nháp (đo chất lượng là việc Tech Lead).
6. Bài thi C++ chấm được trên judge amd64.
7. `docker stats` lúc rảnh và lúc chấm; ghi RAM / CPU.

Ghi `docs/sprints/ops-vps/handoff/dev-deploy.md`: lệnh dựng, những gì đổi, kết quả 7 bước, giới hạn. Không mở subagent. Lỗi thật trong mã khi chạy với LLM thật (vd. JSON bọc ```json, placeholder không khôi phục) → sửa nhỏ trên nhánh này kèm test, ghi handoff; thay đổi lớn → báo PM. Xong báo PM ≤ 5 dòng.
