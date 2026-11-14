# QC sprint 10.5 (FEAT-decision-model, D60) — quy ước chung của bộ test case

Viết ở **pha 1** (2026-11-14) từ `docs/specs/FEAT-decision-model/{US,SRS,QUESTIONS}.md` v1.0 APPROVED (`05911274`; US-DM-01…09). Không đọc mã, handoff hay test của dev. Chưa chạy gì. Câu hỏi mở: `questions.md` (Q-S105-n). Hàm trợ giúp bash (`tok`, `api`), cột «Loại», tài khoản seed, lớp `761987` / `761988`: **dùng `docs/sprints/10/qc/tc-00-chung.md`** (có trên `main`); ma trận vai / tài khoản sprint 11: `docs/sprints/11/qc/tc-00-chung.md`.

## Luật chạy (chủ dự án / PM 2026-11-14) — áp cho MỌI TC của sprint này
| Mục | Quy tắc |
| --- | --- |
| Provider | **Mặc định `decision-fake`.** QC **không gọi `gemini-decision-1` thật** và không nạp / reindex embedding bằng Gemini. TC cần bản thật (đo, lượt thật AC6 US-DM-08) ghi «chờ phép PM»; không kiểm = «KHÔNG KIỂM ĐƯỢC (chờ phép)», **không tính FAIL sản phẩm** nhưng ghi rõ ở báo cáo |
| Guard | `LLM_DECISION_ALLOW_REAL=false` (mặc định) phải còn nguyên suốt pha 2. Mọi TC «bản thật» kiểm **đường chặn** (chưa gọi mạng); TC kiểm đường mã bản thật (header, thân, ánh xạ lỗi, trần ngày) dùng **máy chủ giả cục bộ** nếu base URL cấu hình được (Q-S105-1), không bao giờ khoá Gemini thật. Phát hiện bất kỳ gói tin ra `generativelanguage.googleapis.com` ở stack QC (log, `lsof`, `tcpdump` nhẹ) → **lỗi nghiêm trọng**, dừng, báo PM |
| Sinh chữ | Chỉ 9router (`http://host.docker.internal:20128/v1`, model `ag/gemini-3.8-flash`); TC kiểm TTFT / chất lượng chat dùng 9router thật, còn lại `fake` |
| Embedding | `fake` ở `ep-qc` |
| Khoá | Không ghi khoá (9router, Gemini) vào repo, log, báo cáo; `grep` khoá trong diff + log ở mỗi story |
| Stack | `-p ep-qc`, `EP_PORT_OFFSET=500`, dựng từ commit handoff, `SEED_ON_EMPTY_DB=true`; **tối đa demo + 1 stack**; chỉ dựng khi PM báo tới lượt |

## Biến môi trường để QC điều khiển bản giả (`SRS §4.6, §8`)
`LLM_DECISION_FAKE_MODE` ∈ {`error`, `timeout`, `always_yes`, `always_no`} (E2E); `FAKE_DECISION_PROB` (ghi đè xác suất / `P(có)` cố định cho mọi lời gọi giả; P-105-1 Q2; tên chính xác theo handoff); `LLM_DECISION_FAKE_P50_MS` / `…_P95_MS` (mặc định 426 / 682); `LLM_DECISION_FAKE_ERROR_RATE`; `LLM_DECISION_TIMEOUT` (800 ms); `THREADS_STRUCTURED_TIMEOUT` (6 s); `LLM_DECISION_RPM` (150); `LLM_DECISION_ALLOW_REAL`, `LLM_DECISION_REAL_DAILY_CAP` (300). Đổi biến = khởi động lại `gateway` (ghi vào báo cáo từng TC). Bản phát lại: `internal/llm/testdata/decision-replay.json` — **bảng ca (đầu vào → P/nhãn) dev nêu ở handoff** (P-105-1 Q2); đầu vào ngoài bản phát lại → bản giả trả **`noul = 0,25`** (dưới ngưỡng, ép fallback; P-105-1 Q3, SRS 4.6 BA sửa theo). Base URL provider `decision` cấu hình được (env + bảng provider; P-105-1 Q1) → QC dựng máy chủ giả Python chuẩn để kiểm đường mã thật mà không chạm Gemini.

## Quan sát được (hộp đen)
`llm_audit` (cột `task='DECIDE'`, `cascade_gate`, `cascade_path`, `status`, `error_kind`, `provider`, `model`, `tokens_in`, `latency_ms`, `cost_est`); `audit_log` (`entity='decision_gate'` / `'course_decision_gate'`); API SRS §6 #1–7; giao diện `/settings/llm`, `/observability`; hộp thoại chặn ở Threads; cờ ở `/grading`.

## Phủ story × file
| Story | File |
| --- | --- |
| US-DM-01, 02 (`Decide()`, provider, cascade, cấu hình) | `tc-US-DM-01-02.md` |
| US-DM-03, 04 (cổng Threads, cổng chèn lệnh) | `tc-US-DM-03-04.md` |
| US-DM-05, 09 (cổng chat, ranh giới an toàn) | `tc-US-DM-05-09.md` |
| US-DM-06, 07 (`/settings/llm`, `/observability`) | `tc-US-DM-06-07.md` |
| US-DM-08 (đo trước / sau, seed, guide, cổng) | `tc-US-DM-08.md` |

## Kiểm chéo áp cho mọi story (pha 2)
Phân quyền 4 vai + SV ngoài lớp; không PII trong log / `llm_audit` / payload tới provider (kể cả bản giả: QC bắt thân yêu cầu ở máy chủ giả); `PUT` có `version` → 409; danh sách có cursor ≤ 100; mỗi màn có tải / rỗng / lỗi; 375 px; `bash scripts/ui-antipatterns.sh`; không `fetch(` trần; không thư viện / SDK mới (`go.mod`, `package.json`, `go list -deps` không có `google.golang.org/genai`); không sửa test cũ (`git diff <gốc>... -- '*_test.go' frontend/e2e` chỉ thêm); **mặc định tắt: stack seed sạch không có lời gọi decision nào** (`llm_audit task='DECIDE'` = 0) cho tới khi QC bật cổng.

## Lịch sử sửa TC
- 2026-11-14: cập nhật theo P-105-1 (Q1 base URL cấu hình được; Q2 bảng replay + `FAKE_DECISION_PROB`; Q3 bản giả trả 0,25 cho đầu vào lạ; Q6 đối chứng = 6 bài). Spec: BA vá SRS 4.6 / US-DM-04 AC5.
