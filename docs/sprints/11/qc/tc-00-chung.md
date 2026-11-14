# QC sprint 11 (FEAT-p10-eval, P10) — quy ước chung của bộ test case

Viết ở **pha 1** (2026-11-14) từ `docs/specs/FEAT-p10-eval/{US,SRS,QUESTIONS}.md` v1.0 APPROVED + P-11-1 (PM chấp nhận cả 4: bảng `llm_request_payloads` chỉ bản đã che 30 ngày; mở chi tiết bằng `POST …/requests/{id}/open`; ADMIN không đọc câu mẫu insight; 5 endpoint điểm thi + tab «Điểm thi»). Không đọc mã, handoff hay test của dev. Chưa chạy gì. Câu hỏi mở: `questions.md` (Q-S11-n). Quy ước, hàm trợ giúp bash (`tok`, `api`), cột «Loại», tài khoản seed, lớp `761987` / `761988`: **dùng nguyên `docs/sprints/10/qc/tc-00-chung.md`** (có trên `main` sau khi sprint 10 merge).

## Khác với sprint 10
| Mục | Giá trị |
| --- | --- |
| Stack | `-p ep-qc`, `EP_PORT_OFFSET=500`, dựng từ commit handoff, `SEED_ON_EMPTY_DB=true`. **Luật RAM (PM): tối đa demo + 1 stack**; chỉ dựng khi PM báo tới lượt. Dữ liệu T1 (≥ 1 triệu `llm_audit`, 1.000 bài nộp): xem Q-S11-2 |
| LLM | Pha 2 đợt R dùng Gemini thật; TC của sprint này mặc định provider `fake` (kết quả tất định). TC cần LLM thật ghi «LLM thật» |
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
