# QC test case — US-NFR-01 Môi trường đo chạy lặp lại được trên Mac
Nguồn: US-NFR-01 + SRS §4 FR-1–3, §6. Loại: xem `tc-00-chung.md`.

| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-N01-01 | AC1 | repo ở commit handoff | `test -f benchmarks/README.md && grep -n "Đo trên Mac" benchmarks/README.md` | Tệp có, mục «Đo trên Mac» có | FILE |
| TC-N01-02 | AC1 | – | `grep -Ein 'VPS|/srv/work/research|\bssh\b|9router' benchmarks/README.md benchmarks/real-llm/README.md` | 0 dòng (không nhắc VPS, `/srv/work/research`, `ssh`, 9router; `real-llm/README.md` cũng nên sạch theo AC2) | SCAN |
| TC-N01-03 | AC1 | Mac sạch: Colima chạy, chưa có stack `ep-qc` | Làm **đúng từng bước** trong README «Đo trên Mac» (không dùng kiến thức ngoài): dựng stack `-p ep-qc`, cổng lệch, chạy một lệnh `smoke.js` | Dựng được stack; `smoke.js` chạy xong, có JSON kết quả; mọi lệnh trong README chạy được nguyên văn (lệnh nào lỗi → ghi dòng README + lỗi; README thiếu bước → FAIL) | RUN |
| TC-N01-04 | AC1 | TC-N01-03 | `docker ps --format '{{.Names}} {{.Ports}}'` | Mọi cổng của `ep-qc` bind `127.0.0.1` (hoặc cổng lệch theo README), không trùng cổng stack demo; container tên `ep-qc-*` | RUN |
| TC-N01-05 | AC2 | – | `sed -n` toàn `benchmarks/real-llm/README.md`; `grep -n -i "không chạy trên Mac\|Chạy trên VPS"` | Dòng cũ «Chạy trên VPS … không chạy trên Mac» **không còn**; có quy trình Mac: stack `-p ep-research`, cổng lệch, `down -v` khi xong | FILE |
| TC-N01-06 | AC2 | – | `grep -nE 'EP_HOST|9router|sqlite|sqlite3|better-sqlite|https://\$?\{?EP_HOST' deploy/vps-llm.mjs`; `grep -n 'process.env' deploy/vps-llm.mjs` | Không đọc khoá 9router / sqlite; không `https://EP_HOST` cứng; có đọc env `BASE`, `MODEL`, `FALLBACK`, `PHASE`, `GEMINI_API_KEY` | SCAN |
| TC-N01-07 | AC2 | Stack `-p ep-qc` (cổng lệch), admin đăng nhập được | `BASE=<api ep-qc> PHASE=live MODEL=gemini-test-a FALLBACK=gemini-test-b GEMINI_API_KEY=AIzaFAKE0000000000000000000000000000000 node deploy/vps-llm.mjs` (khoá **giả**, stack kiểm; không gọi mạng ngoài nếu script không cần) rồi `GET /admin/llm/providers`, `GET /admin/llm/routes` bằng admin | Có provider `gemini-chat` (`type: gemini`) với `model`/dự phòng như env, `rpm_limit` 5, và `gemini-embed` (`type: gemini`, `gemini-embedding-001`, `dims` 1536, `rpm_limit` 60); route sinh chữ trỏ `gemini-chat`; **khoá không in ở stdout/stderr và API trả dạng che** | RUN |
| TC-N01-08 | AC2 | – | Chạy lại với `PHASE=seed` | Embedding thật (`gemini-embed`), chat giả (theo mô tả «PHASE=seed (embedding thật, chat giả)»); route chat là `fake` | RUN |
| TC-N01-09 | AC2 | Không có `GEMINI_API_KEY`, `BASE` đúng | `BASE=… PHASE=live node deploy/vps-llm.mjs` | Dừng ngay, **một dòng** lỗi, không in khoá, mã ≠ 0, không tạo provider | RUN |
| TC-N01-10 | AC2 | – | `git check-ignore -v .env.local`; `git ls-files \| grep -c '\.env\.local'` | `.env.local` bị ignore; 0 tệp `.env.local` được theo dõi | SCAN |
| TC-N01-11 | AC3 | Một tệp kết quả bất kỳ do script đo của sprint ghi (ví dụ `smoke.js` ở TC-N01-03) | `jq '.meta' <file>`; `node benchmarks/check-meta.mjs <file>`; echo $? | `meta` có đủ 7 nhóm trường (xem `tc-00-chung.md`); `check-meta` mã 0 | RUN |
| TC-N01-12 | AC3 | Bản sao `/tmp/m/*.json` của tệp ở TC-N01-11, mỗi bản **thiếu một** trường (`commit`, `thời điểm`, `máy`, `số lần chạy`, `cấu hình`, `cấu hình.model`, `cấu hình.RPM`, `k6_cpus`, `demo_stack`) | `node benchmarks/check-meta.mjs /tmp/m/<x>.json` từng bản | Mỗi bản mã ≠ 0 và nêu **tên trường thiếu** (không báo đạt khi thiếu); thêm bản `demo_stack:"unknown"` → ≠ 0 | RUN |
| TC-N01-13 | AC3 | `benchmarks/reports/**/*.json` hiện có | `for f in $(find benchmarks/reports -name '*.json'); do node benchmarks/check-meta.mjs $f; done` | Mọi tệp mã 0; tệp nào thiếu `meta` = phép đo không hợp lệ (ghi tên) | FILE |
| TC-N01-14 | AC3 | – | Với mỗi lệnh đo trong README (k6, `real_llm.py`, seed T1, baseline, chaos): chạy phần đủ nhẹ hoặc đọc mẫu đầu ra | Đầu ra JSON có `meta` ở **mọi** lệnh (không riêng k6) | FILE |
| TC-N01-15 | AC4 | Stack demo đang chạy hoặc không (QC không động vào) | `BASE=http://127.0.0.1:3000 <lệnh setup của README>`; tương tự `grade-run`, `node scripts/seed-t1.mjs` với `BASE`/`DATABASE_URL` trỏ cổng demo | Mỗi lệnh **từ chối**, in **một dòng** lý do, mã ≠ 0; không gửi request ghi (kiểm `docker logs edupilot-gateway-1 --since 1m`: 0 `POST` mới do lệnh; chỉ đọc log demo) | RUN |
| TC-N01-16 | AC4 | `-p ep-qc` | Cùng lệnh với `BASE` = cổng `ep-qc` | Chạy bình thường (mã 0 hoặc tiến hành) | RUN |
| TC-N01-17 | AC4 | – | Biến thể đánh lừa: `BASE=http://localhost:3000`, `http://[::1]:3000`, `http://127.0.0.1:3000/`, tên host `host.docker.internal:3000`, project mặc định bằng `COMPOSE_PROJECT_NAME=edupilot` | Cả năm bị từ chối (cùng cơ chế kiểm theo cổng / tên project); biến thể nào lọt = FAIL (ghi) | RUN |
| TC-N01-18 | AC5 | repo | `bash scripts/canary-scan.sh; echo $?`; `grep -rEn 'AIza[0-9A-Za-z_-]{20,}|sk-[A-Za-z0-9]{16,}' benchmarks docs/research` ; `grep -rn 'Bearer ' benchmarks docs/research` | `canary-scan` mã 0, 0 kết quả; không khoá thật; dòng `Bearer ` chỉ được phép là **mẫu mã** (`Bearer ${token}`, `Bearer <token>`), không phải token (P-95-7 Q1: `Bearer ${…}` / `Bearer <…>` là mẫu được phép, cấm token thật) | SCAN |
| TC-N01-19 | AC5 | – | `git log --all -p -S'AIza' -- . \| grep -c 'AIza[0-9A-Za-z_-]\{20,\}'` (lịch sử nhánh) | 0 | SCAN |

## Phân quyền / bí mật
TC-N01-07, 09, 10, 18, 19 (khoá chỉ ở `.env.local` / env / màn Cấu hình LLM).

## Lịch sử sửa TC
- 2026-11-14 — PM trả lời P-95-7 (P-95-7 Q1…23 chấp nhận theo đề xuất QC): TC-N01-18 ghi rõ Q1 (không đổi hành vi kiểm).
