# QC sprint 10 (FEAT-practice, P9) — quy ước chung của bộ test case

Viết ở **pha 1** (2026-11-14) từ `docs/specs/FEAT-practice/{US,SRS,QUESTIONS}.md` v1.0 APPROVED. Không đọc mã, handoff hay test của dev. Chưa chạy gì. Câu hỏi: `qc/questions.md`; **PM đã trả lời Q-S10-1…26 ở P-10-7 (2026-11-14)**, spec lên v1.1. TC ghi «P-10-7 Qn» là đã chốt; «**chờ handoff**» = dev trả lời ở handoff + openapi (TC chạy theo mặc định QC, đối chiếu lại khi có handoff, kết quả khác → báo PM, không tự đoán).

## Môi trường pha 2
| Mục | Giá trị |
| --- | --- |
| Stack | `-p ep-qc`, `EP_PORT_OFFSET=500`, `JUDGE_EXTRA_ARGS=-no-seccomp`, `--profile ingest` (cần docling cho TC trích đề), dựng từ **commit handoff**, `SEED_ON_EMPTY_DB=true`; mock LLM `env:fake` + `fake_llm.json` (bản ghi `QUESTION_EXTRACT`, `QUESTION_GEN`, `PRACTICE_EXPLAIN`, `PRACTICE_FEEDBACK` do dev cung cấp; thiếu bản ghi nào → TC liên quan «KHÔNG KIỂM ĐƯỢC», báo PM) |
| Một stack một lúc | Không dựng khi dev đang đo tải (lệnh PM 2026-11-14); đo thời gian ghi «MacBook Apple Silicon, Colima 6 CPU / 10 GB» |
| Base | `B=https://localhost:943/api/v1` (Caddy, `curl -k`) |
| Lớp | `761987` = lớp 1 (30 SV, ngân hàng 80 APPROVED + 20 PENDING); `761988` = lớp 2 (ngân hàng nhỏ khác). `C1`, `C2` = id lớp lấy từ `GET /me/courses` |
| Tài khoản (mật khẩu `SEED_DEFAULT_PASSWORD`) | `teacher@edupilot.local` (GV lớp 1), `ta@edupilot.local`, `admin@edupilot.local`, `sv.kha`, `sv.gioi` (SV lớp 1; A và B), `sv.moi` (chưa vào lớp), SV chỉ ở lớp 2 (Q-S10-1) |

## Hàm trợ giúp (bash, dán đầu phiên)
```bash
B=https://localhost:943/api/v1; PW="$SEED_DEFAULT_PASSWORD"
tok(){ curl -sk -X POST $B/auth/login -H 'content-type: application/json' -H "Idempotency-Key: $(uuidgen)" -d "{\"email\":\"$1@edupilot.local\",\"password\":\"$PW\"}" | jq -r .access_token; }
# api USER METHOD PATH [JSON]  → in "HTTP_CODE" rồi thân; POST tự thêm Idempotency-Key (đặt IDEM=… để dùng khoá cố định); H="Header: v" để thêm tiêu đề
api(){ local u=$1 m=$2 p=$3 d=${4:-}; local k="${IDEM:-$(uuidgen)}"; curl -sk -w '\n%{http_code}\n' -X $m "$B$p" -H "authorization: Bearer $(tok $u)" -H 'content-type: application/json' ${H:+-H "$H"} $([ $m = POST ] && echo -H "Idempotency-Key: $k") ${d:+-d "$d"} | tac | { read c; echo "HTTP $c"; tac; }; }
C1=$(curl -sk $B/me/courses -H "authorization: Bearer $(tok sv.kha)" | jq -r '.items[]|select(.course.class_code=="761987").course.id')
```
Truy vấn DB (chỉ đọc, trừ khi TC ghi «ghi DB»): `docker compose -p ep-qc exec -T postgres psql -U <user> -d edupilot -c "…"` (user lấy từ `.env.local`). `llm_audit`: bảng ghi lời gọi LLM (cột `task`, `route`, `model`, `batch_id`…; chỉ đọc).

## Cột «Loại»
`API` curl/jq (QC chạy) · `GO` chạy test Go nêu trong spec, ghi 10 dòng cuối + mã thoát · `DB` truy vấn SQL · `E2E` Playwright (QC viết ở pha 2 `frontend/e2e/qc-practice-*.spec.ts`, `playwright.qc-real.config.ts`, 1440 + 375 px) · `TAY` thao tác tay + chụp `qc/shots/` · `DIFF` đọc diff (pha 2, mục kiểm chéo).

## Kiểm chéo bắt buộc cho mọi story (pha 2, ghi vào report)
1. **Phân quyền:** vai sai / SV ngoài lớp / SV lớp khác → `403` hoặc `404` đúng như spec; ADMIN không có route nội dung lớp.
2. **PII:** `docker compose logs gateway worker` (cả giờ chạy) không có tên / MSSV / email thật; `llm_audit` payload của mọi lời gọi trong lượt test không có tên / MSSV (kể cả câu hỏi do QC gõ chứa tên, MSSV).
3. **Idempotency:** mọi `POST` có `Idempotency-Key` gửi lại cùng khoá → cùng kết quả, không tạo trùng; thiếu khoá khi bắt buộc → `400/422` (ghi mã thật).
4. **Phân trang:** `limit=101` → bị chặn hoặc cắt ≤ 100; không OFFSET; `cursor` cuối trả rỗng.
5. **Trạng thái tải / rỗng / lỗi và 375 px** cho mọi màn mới (E2E, chụp ảnh).
6. **Không rò đáp án:** tìm `answer_key|correct|explanation|citations` trong **mọi** phản hồi sinh viên của lượt đang làm.
7. **Luật chữ UI / không gamify / không từ kỹ thuật AI với SV** (`bash scripts/ui-antipatterns.sh`; quét chữ SV 375 px mọi route mới: không «AI», «RAG», «citation», «model», «LLM», «prompt», «embedding», «token»; không streak, xếp hạng, huy hiệu, vòng tiến độ, pháo hoa).
8. Migration `00019_practice` chạy sạch trên DB trống; `goose down` rồi `up`.
9. Đọc diff tìm: mã đụng `exam_events`, `grade_entries` ngoài US-P9-11, `fetch(` trần, import `internal/llm` trong `internal/quiz`, thư viện lạ, test bị sửa/bỏ.

## Thứ tự chạy pha 2 (theo thứ tự thi công của spec)
P9-01 → 02 → 04 → 05 → 06 → 07 → 09 → 10 → 08 → 12; P9-03 và P9-11 là «Could, bỏ được»: **chỉ kiểm khi handoff nói đã làm**; handoff ghi «không làm» thì ghi «NGOÀI PHẠM VI SPRINT (theo spec)» chứ không FAIL.

## Danh sách file
`tc-US-P9-01.md` … `tc-US-P9-12.md`, `questions.md`.

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
- 2026-11-14 — cập nhật theo spec v1.1 và P-10-7: xem «Lịch sử sửa TC» từng tệp.
