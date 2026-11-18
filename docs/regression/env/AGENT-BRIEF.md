# Brief cho agent smoke hồi quy (qc2) — đọc hết trước khi làm

## Mục tiêu
Chạy lại MỌI test case cũ của sprint được giao trên mã mới nhất (sprint/11-p10-eval) trên stack QC `ep-r` cho tới khi N = 0 (không chọn mẫu). Kết quả mỗi ca: P / F / LT (lỗi thời: spec đã đổi hợp lệ ở sprint sau, nêu quyết định) / K(a|b|c) ((a) thiết bị ta không có, (b) cần người thật, (c) ngoài phạm vi luận văn) / N (chưa chạy). Một dòng bằng chứng mỗi ca. Lỗi do môi trường (timeout, tải máy, rate limit, fixture cũ, lỗi kịch bản QC) PHẢI chạy lại tới khi P hoặc F; không phải K, không phải F. Bạn là QC: trượt thật là phát hiện sản phẩm.

## Stack ep-r (đã dựng, KHÔNG dựng lại / khởi động lại)
- Worktree `~/Documents/TA_Agent_qc2` (detached, đã merge origin/sprint/11-p10-eval). Không đụng `~/Documents/TA_Agent`. Không đụng nhánh `sprint/11-p10-eval`. KHÔNG sửa mã sản phẩm (`backend-go/`, `frontend/src/`) hay test hiện có. Spec QC mới chỉ ở `frontend/e2e/qc2-*` hoặc `docs/regression/evidence-<sprint>/`.
- `source docs/regression/env/ep-r-lib.sh` → B, PW, tok <user>, psqlr, rlclear. HTTPS `https://localhost:1143` (API `/api/v1`), FE :3700, Mailpit :8725, MinIO :9700, Postgres container `ep-r-postgres-1` (user/db `edupilot`), Redis `ep-r-redis-1`. Mật khẩu seed `Edupilot#Seed-2026`; tài khoản: admin, teacher, ta, sv.gioi, sv.kha, sv.nguyco (@edupilot.local), gv.s8f. Lớp seed 761987/761988 DÙNG CHUNG: không sửa; tạo lớp riêng (SQL/API) cho ca ghi dữ liệu.
- Go test: `DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true go test ./internal/<gói> …` (chạy THEO TỪNG GÓI; `./...` toàn cây quá tải). Thêm `-tags integration` / `-tags testroutes` theo ca.
- Playwright thật: dùng `frontend/node_modules/@playwright/test`; cấu hình `frontend/playwright.qc2g.config.ts` (`QC2G_DIR=<thư mục spec> QC_REAL=1 NODE_TLS_REJECT_UNAUTHORIZED=0 npx playwright test -c playwright.qc2g.config.ts <spec>`); thư viện đăng nhập/điều hướng mẫu ở `frontend/e2e/qc2-r8/lib.ts`. Ảnh mốc visual: image `mcr.microsoft.com/playwright:v1.63.0-noble`.
- Lần chạy trước của sprint đó: `docs/regression/smoke-<N>.md`, `evidence-<N>/` (script, state, results), định nghĩa TC ở `docs/sprints/<N>/qc/`. Tái dùng script cũ, sửa nếu lỗi kịch bản. Bảng smoke sinh bằng script trong evidence (make_report.py / build*.py / smk34-merge.py, bản ghi cuối của mỗi TC thắng): thêm bản ghi rồi chạy lại script; không xoá kết quả cũ.

## LUẬT AN TOÀN (đã gây sự cố thật)
1. Chuỗi shell/tsend chỉ dùng nháy ĐƠN; không backtick, không dollar-paren trong chuỗi nháy kép.
2. CẤM `pnpm dev`, `pnpm dev:down`, mọi lệnh trên compose project `edupilot`, `docker compose down -v`, xoá volume; chỉ project `ep-r` (hoặc project tạm của chính bạn tên `smk…`, dọn xong). Không `pkill`/`killall` theo tên; chỉ kill PID của mình. Không dừng container `edupilot-test-*` (testcontainers dùng chung).
3. Auth bị giới hạn 5/giờ/IP: trước ca đăng nhập nhiều chạy `rlclear` (hoặc `docker exec ep-r-redis-1 sh -c "redis-cli --scan --pattern 'ep:rl:auth*' | xargs -r redis-cli del"`).
4. 9router (LLM thật) dùng chung 20 yêu cầu/phút: ca gọi LLM thật chạy TUẦN TỰ; ca về đường ống dùng stub OpenAI-compatible cục bộ (`evidence-8/scripts/s8r-stub.py`, `evidence-3/scripts/qstub.py`). KHÔNG gọi embedding Gemini (hết quota; route EMBEDDING giả/rỗng).
5. Thay đổi toàn cục (route LLM/ngân sách, cờ, đổi env hoặc dừng/khởi động lại gateway-worker-redis-postgres) CHỈ khi giữ khoá `mkdir /tmp/qc2-lock-global` (thử lại mỗi 15 s, giữ ≤ 10 phút), ghi 'HH:MM:SS <tên>: LOCK — …' / 'UNLOCK — …' vào `/tmp/qc2-global-changes.log`, hoàn lại mặc định, rồi `rmdir` đúng khoá của mình.
6. Tải: tối đa 2 việc nặng của bạn cùng lúc; kết quả 5xx/504/timeout do tải → chờ rồi chạy lại. Khi PM báo cửa sổ đo (LCP/E6: 'dừng Playwright/go test/build nặng') → dừng việc nặng, chỉ viết tài liệu.
7. Không sửa `.env.local`. Không ghi secret/PII vào repo.
8. KHÔNG git commit/push (agent chính commit). Không git add/checkout/reset/stash.
9. Không tạo agent con.

## Đầu ra
- Mỗi ca một dòng trong `docs/regression/smoke-<N>.md` + header đếm khớp; mỗi F một mục trong `docs/regression/defects-<N>.md`: mã ca, triệu chứng 1 dòng, bước tái hiện, mức (Cao/Trung bình/Thấp), sprint gốc, có phải hồi quy do sprint sau không (đọc git log/blame, chỉ đọc).
- Sau mỗi nhóm ca: thêm 1 dòng vào `/tmp/qc2-progress.tsv` KHÔNG; thay vào đó cập nhật dòng sprint của mình trong tệp đó bằng sed/python theo định dạng cột: `sprint N⇥tổng⇥đạt⇥trượt⇥(tổng−đạt−trượt−N)⇥chữ ngắn việc đang làm`.
- Lỗi sản phẩm MỚI → ngay: `tsend pm 'qc2-<tên>: DEF <mã ca> <mức> <triệu chứng> — bằng chứng <đường dẫn>'` (nháy đơn).
- Tin cuối: P/F/LT/K/N từng sprint, danh sách F (mã + triệu chứng + mức), tệp đã đổi, ca còn N + lý do.
