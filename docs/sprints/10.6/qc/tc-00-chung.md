# QC sprint 10.6 (FEAT-grading-guidance, D61/D62) — quy ước chung của bộ test case

Viết ở **pha 1** (2026-11-14) từ `docs/specs/FEAT-grading-guidance/{US,SRS,QUESTIONS}.md` v1.0 APPROVED (`6d6db6f7`; US-GD-01…11) + P-106-1 (PM xác nhận Q-G-1, 2, 3, 5, 6, 10 và nhãn S7). Không đọc mã, handoff hay test của dev. Chưa chạy gì. Câu hỏi mở: `questions.md` (Q-S106-n). Hàm trợ giúp, tài khoản seed, lớp `761987` / `761988`, ma trận vai: `docs/sprints/10/qc/tc-00-chung.md`; luật chạy / cách dựng stack: `docs/sprints/10.5/qc/tc-00-chung.md`; kịch bản đã chạy được: `docs/sprints/10.5/qc/scripts/{dmlib,dmx,stub_llm}.py`.

## Luật chạy (chủ dự án / PM) — áp cho mọi TC
| Mục | Quy tắc |
| --- | --- |
| LLM | Sinh chữ **chỉ qua 9router** (hoặc `fake` / **stub OpenAI-compatible do QC dựng** — xem dưới); **không Gemini trực tiếp**, không `gemini-decision-1`, embedding giả. Đo MAE / bench trên E3 chạy khi PM cho phép; TC kiểm cấu trúc dùng stub |
| Stub của QC | `stub_llm.py` nhận cấu hình route `GRADING` (hoặc tác vụ do `reflect` dùng) trỏ `http://host.docker.internal:8777/v1`; chế độ cho `reflect`: `ok` (mục hợp lệ bám `case_ids` lấy từ đầu vào lời gọi), `bad_criterion`, `bad_case`, `wrong_direction`, `total_keyword`, `pii_text`, `dup_text`, `low_support`, `error`, `invalid`, `slow`. Cho phép QC ép từng nhánh kiểm bằng code (SRS 4.2.8). Phụ thuộc: đầu vào / schema của lời gọi đọc được (Q-S106-1) |
| Stack | `-p ep-qc`, `EP_PORT_OFFSET=500`, `EP_PORT_MOCK_GRAPH=8590`, `EP_PORT_MOCK_IMAP=8591`, `--profile ingest` khi seed (docling), dựng từ commit handoff; **tối đa demo + 1 stack**; chỉ khi PM báo tới lượt. Lưu ý đã gặp ở 10.5: không `export DATABASE_URL` / `APP_ENV` vào shell dựng stack; chạy chuỗi lệnh nhiều bước **trong một lệnh bash** (nền bị giết khi lệnh cha kết thúc) |
| Nhãn S7 (P-106-1) | `expected` của S7 giữ **7,00** (nháp AI lượt 1; test mã dùng); `teacher_label` của S7 = **8,00** dùng cho **mọi phép đo MAE** (E3, PoC, đo lại 9.5). TC kiểm: (i) `expected_grading.json` / `expected_final_grades.csv` **không bị sửa** (`git diff`); (ii) mã đo (`benchmarks/guidance/run.py`, `cmd/regress`) đọc `teacher_label`, không đọc `expected`, cho MAE; (iii) báo cáo ghi rõ «nhãn S7 = 8,00 (teacher_label)» |
| Bất biến feature (kiểm chéo mọi story) | (1) không lưu ý nào tự áp; (2) không văn bản bài / danh tính ở bước rút lưu ý; (3) điểm tổng / trừ muộn / làm tròn là code thuần, không đổi; (4) mọi lời gọi LLM qua `internal/llm`, `BATCH`, `llm.Structured`; (5) mặc định tắt theo lớp, tắt = như trước (byte-for-byte prompt); (6) **sinh viên không bao giờ thấy** lưu ý / phiên bản / cờ / số đo |
| Dữ liệu | Seed BT03 lớp 1: 16 bài chấm đủ (đã chấm ở stack 10.5: chạy `POST …/assignments/{aid}/grade-all`); cần ≥ 8 kết quả `REVIEWED` có sửa ≥ 0,5 (AC1 US-GD-10): nếu seed chưa có, QC dựng bằng API duyệt (`review`) và ghi rõ |

## Phủ story × file
| Story | File |
| --- | --- |
| US-GD-01, 02 (cờ theo lớp; nền dữ liệu, ghim, bất biến) | `tc-US-GD-01-02.md` |
| US-GD-03, 04 (rút lưu ý; duyệt / Áp dụng) | `tc-US-GD-03-04.md` |
| US-GD-05, 06 (ghim prompt, chấm lại, chặn lô trộn; số đo) | `tc-US-GD-05-06.md` |
| US-GD-07, 08, 09 (ca hồi quy; câu tương tự Threads; lý do loại câu hỏi) | `tc-US-GD-07-09.md` |
| US-GD-10, 11 (seed, cổng, đo E3, guide; ranh giới an toàn) | `tc-US-GD-10-11.md` |
| P-106-11 (PII «dấu là tập con»), P-106-12 (cụm miễn từ khoá, `dropped` theo tiêu chí), UX theo nhật ký giaovien | `tc-P106-11-12-UX.md` (+ mẫu `TestReflectKeywordExempt.go.txt`) |

## Kiểm chéo áp cho mọi story (pha 2)
Phân quyền 4 vai + TA chỉ đọc + lớp khác `404`; không PII / văn bản bài trong log, `llm_audit`, payload tới LLM, bảng mới; `POST`/`PUT` có tác dụng phụ nhận `Idempotency-Key` / `version` → 409; danh sách có cursor ≤ 100; mỗi màn có tải / rỗng / lỗi; 375 px; `bash scripts/ui-antipatterns.sh`; không `fetch(` trần; không thư viện / dịch vụ bộ nhớ agent (`go.mod`, `package.json`); không sửa test cũ (`git diff <gốc>... -- '*_test.go' frontend/e2e` chỉ thêm); **mặc định tắt: stack seed sạch không có lời gọi `reflect`, không khối lưu ý trong prompt chấm** cho tới khi QC bật cờ; chữ UI không «AI / nháp / RAG / PII» ở màn lưu ý; tất cả mã lỗi mới có trong `openapi.yaml`.

**Cổng `-run` không rỗng (P-106-7).** Mọi lệnh `go test … -run '<tên>'` trong TC phải chạy ≥ 1 test mỗi tên (`-v` thấy `--- PASS: <tên>`); gói in `[no tests to run]` hoặc thiếu một tên trong danh sách = **FAIL** (không tính là xanh).

## Lịch sử sửa TC
- 2026-11-14 (P-106-7): thêm «Cổng `-run` không rỗng»; TC-G03-13 sửa ở `tc-US-GD-03-04.md`.
- 2026-11-14 (P-106-11, P-106-12, SRS v1.7): thêm `tc-P106-11-12-UX.md` (TC-P11-01…10, TC-P12-01…26, TC-UX-S21…o, 17…19).
