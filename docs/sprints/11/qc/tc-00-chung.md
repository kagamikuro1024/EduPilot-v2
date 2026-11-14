# QC sprint 11 (FEAT-p10-eval, P10) — quy ước chung của bộ test case

Viết ở **pha 1** (2026-11-14) từ `docs/specs/FEAT-p10-eval/{US,SRS,QUESTIONS}.md` v1.0 APPROVED + P-11-1 (PM chấp nhận cả 4: bảng `llm_request_payloads` chỉ bản đã che 30 ngày; mở chi tiết bằng `POST …/requests/{id}/open`; ADMIN không đọc câu mẫu insight; 5 endpoint điểm thi + tab «Điểm thi»). Không đọc mã, handoff hay test của dev. Chưa chạy gì. Câu hỏi mở: `questions.md` (Q-S11-n). Quy ước, hàm trợ giúp bash (`tok`, `api`), cột «Loại», tài khoản seed, lớp `761987` / `761988`: **dùng nguyên `docs/sprints/10/qc/tc-00-chung.md`** (có trên `main` sau khi sprint 10 merge).

## Khác với sprint 10
| Mục | Giá trị |
| --- | --- |
| Stack | `-p ep-qc`, `EP_PORT_OFFSET=500`, dựng từ commit handoff, `SEED_ON_EMPTY_DB=true`. **Luật RAM (PM): tối đa demo + 1 stack**; chỉ dựng khi PM báo tới lượt. Dữ liệu T1 (≥ 1 triệu `llm_audit`, 1.000 bài nộp): xem Q-S11-2 |
| LLM | **Luật chủ dự án 2026-11-14:** mọi LLM sinh chữ (chat, chấm, trích công thức, insight, eval) qua 9router trên Mac (`http://127.0.0.1:20128/v1`; từ container `http://host.docker.internal:20128/v1`; model `ag/gemini-3.8-flash`, dự phòng model khác của 9router). `GEMINI_API_KEY` chỉ cho `google/gemini-decision-1` (decision model) và embedding `gemini-embedding-001`. QC **không gọi Gemini trực tiếp để sinh chữ**; ở `/settings/llm` chọn nhà cung cấp 9router. TC của sprint này mặc định provider `fake` (tất định); TC cần LLM thật ghi «LLM thật» = 9router. Khoá 9router không ghi vào báo cáo / log |
| Admin | `admin@edupilot.local`; GV lớp 1 `teacher@edupilot.local`; TA lớp 1 `ta@edupilot.local`; SV: `sv.kha`, `sv.gioi`, `sv.moi` (chưa vào lớp), SV chỉ ở lớp 2 |
| Mã cuối | `P` = PASS, `F` = FAIL, `K` = KHÔNG KIỂM ĐƯỢC (tính FAIL, ghi lý do). Kết luận story: một AC FAIL → story FAIL |

## Loại TC
`API` · `GO` (chạy test Go nêu trong spec, ghi 10 dòng cuối + mã thoát) · `DB` (SQL chỉ đọc) · `E2E` (Playwright QC viết ở pha 2, `frontend/e2e/qc-p10-*.spec.ts`, 1440 + 375 px) · `TAY` (thao tác + ảnh `qc/shots/`) · `DIFF` (đọc diff).

## Phủ story × file
| Story | File |
| --- | --- |
| US-P10-01, 02, 03 (Observation A, cảnh báo) | `tc-US-P10-01-03.md` |
| US-P10-04, 05 (Báo cáo lỗ hổng, `/insights`) | `tc-US-P10-04-05.md` |
| US-P10-06, 07 (điểm thi `/analytics`, seed + cổng) | `tc-US-P10-06-07.md` |
| US-P10-08, 09 (LCP, Nợ L6) | `tc-US-P10-08-09.md` |
| US-P10-10, 11 (`make eval`, E2E F1–F17) | `tc-US-P10-10-11.md` |
| US-P10-12 (bảo mật, README, visual QA, demo) | `tc-US-P10-12.md` |

## Kiểm chéo áp cho mọi story (pha 2)
Phân quyền 4 vai + SV ngoài lớp; không PII trong log / payload LLM / `llm_request_payloads`; `POST` có tác dụng phụ nhận `Idempotency-Key`; danh sách có cursor; mỗi màn có tải / rỗng / lỗi; 375 px; `bash scripts/ui-antipatterns.sh`; không `fetch(` trần; không `OFFSET` mới ở `internal/store/queries/`; không sửa test cũ (`git diff <gốc>... -- '*_test.go' 'frontend/e2e'` chỉ thêm).

## Quy tắc chi phí (PM 2026-11-14, bổ sung)
Không gọi Gemini trực tiếp ở bất kỳ TC nào (kể cả decision model `gemini-decision-1`, đo, eval, smoke, PoC). Sinh chữ chỉ qua 9router. Embedding: stack `ep-qc` dùng embedding giả (`fake`), không nạp lại / reindex bằng Gemini trừ khi PM cho phép; TC cần embedding thật (RAG, insight cụm) ghi «chờ PM cho phép embedding thật» và kiểm bằng embedding giả (thứ hạng / cô lập lớp / PII vẫn kiểm được). TC đợt 6 (decision cascade): thiết kế + kiểm bằng provider giả.

## Lịch sử sửa TC
- 2026-11-14: tc-00-chung, TC-04-11, TC-10-10, TC-12-30 — luật LLM mới của chủ dự án (PM): LLM sinh chữ chỉ qua 9router; Gemini trực tiếp chỉ decision model + embedding. Thêm kiểm chéo ở pha 2: `llm_audit.provider`/`route` của lời gọi sinh chữ ≠ Gemini trực tiếp; không khoá Gemini trong cấu hình provider sinh chữ.
