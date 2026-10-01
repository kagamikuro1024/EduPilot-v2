# QC report — US-P0-01  · Kết luận: PASS
Story tài liệu, sản phẩm `docs/DEMO_SCRIPT.md` (phiên bản 1). Spec `docs/specs/FEAT-demo-script/US.md`. Không có handoff dev. Chạy 2026-10-01. TC: `docs/sprints/1/qc/tc-US-P0-01.md` (14 TC). Kết quả: `docs/sprints/1/qc/run-US-P0-01.log` — **108 PASS, 0 FAIL**. 9/9 AC PASS.

## Cổng nghiệm thu đã chạy (lệnh nguyên văn trong US.md → PASS/FAIL)
| AC | Lệnh | Đầu ra | KQ |
| --- | --- | --- | --- |
| AC1 | `grep -oE '^### Bước …— F[0-9]+' \| grep -oE 'F[0-9]+$' \| xargs` | `F2 F3 F5 F7 F9 F10 F17` | PASS |
| AC2 | `awk -F'|' … NF!=8`; `grep -cE '^\| [0-9]{2}:[0-9]{2} \|'` | không in gì; 37 (≥ 30) | PASS |
| AC3 | `… \| sort -c && echo tang-dan` | `tang-dan`; hàng cuối `13:05` | PASS |
| AC4 | vòng `grep -qF` route ở DESIGN/ARCHITECTURE | không `THIẾU` | PASS |
| AC5 | `grep -E '^\| [0-9:]+ \| Sinh viên' … \| grep -inE 'RAG\|PII\|…'` | không in gì | PASS |
| AC6 | vòng 7 khoá | không `THIẾU` | PASS |
| AC7 | `grep -c '^Nếu hội đồng hỏi'`; vòng `*.spec.ts` | 5; không `THIẾU` | PASS |
| AC8 | `grep -cE 'chỉ trả lời được…\|…'` | 4 (≥ 4) | PASS |
| AC9 | `grep -E '^\| P(1\|…\|10) \|' \| wc -l`; `grep -c '^## 7. Nhật ký chạy lại'` | 8; 1 | PASS |

## AC (kiểm chặt hơn lệnh trong US.md)
| AC | Kết quả | Ghi chú |
| --- | --- | --- |
| AC1 | PASS | Đúng 7 tiêu đề `### Bước`, số 1..7, thứ tự F2→F3→F5→F7→F9→F10→F17; mỗi luồng có mục tương ứng ở FLOWS |
| AC2 | PASS | 37 hàng, đủ 6 cột, không ô trống; 7 bảng cùng tiêu đề `Mốc / Vai trò / Route / Thao tác / Kết quả phải thấy / Phase` |
| AC3 | PASS | Mốc tăng chặt (không trùng); tổng thời lượng bảng mục 1 = 13:45 ≤ 15:00, khớp dòng "Hết kịch bản 13:45"; hàng mỗi bước nằm trong khoảng ghi ở tiêu đề (cả 7 bước) |
| AC4 | PASS | 14 route, đều nằm đúng DESIGN §14 hoặc ARCHITECTURE §7 (không chỉ xuất hiện đâu đó trong file); vai trò gồm Admin, Giảng viên, Sinh viên (A–D), Người trình bày (chỉ cho Mailpit `http://localhost:8025`, ghi rõ là công cụ dev) |
| AC5 | PASS | Không từ cấm. Ba hàng sinh viên có "AI" (`trước khi gửi cho AI`, `AI chưa đủ chắc chắn…`) — đúng nguyên văn DESIGN §14.2 / PRD, không thuộc danh sách cấm §13; "không có điểm nháp" là câu khẳng định sinh viên **không** thấy. Dùng đúng cụm §13: "Cần giảng viên hỗ trợ", "Đã ẩn N thông tin cá nhân" |
| AC6 | PASS | 7 khoá đều nằm trong mục 5; 3 tầng; lệnh bật tầng 2 dùng đúng stack US-P0-02 (`--env-file .env.local -f docker-compose.local.yml -p edupilot up -d gateway`); ghi ai bật (Chủ dự án) |
| AC7 | PASS | Bước 1–6 (F2…F10) đều có "Nếu hội đồng hỏi" kèm tên spec E2E, và mỗi spec nằm đúng mục flow của bước (class-join→F2, private-chat→F3, escalation→F5, attendance→F7, assignment-lifecycle→F9, gradebook→F10). Mục 5: tầng 2 khi mạng/API LLM lỗi hoặc một bước lỗi 2 lần; tầng 3 khi stack không lên/tầng 2 lỗi; tầng 2 không cần mạng ngoài |
| AC8 | PASS | Cả 4 cụm nằm trong hàng bảng đúng bước: bước 2 (từ chối hỏi người khác), bước 5 (không điểm nháp), bước 5/6 (chỉ TEACHER; `Chốt điểm` bị khoá khi công thức chưa xác nhận); việc xác nhận công thức do Giảng viên làm (khớp CLAUDE.md) |
| AC9 | PASS | Mục 6 có đúng P1–P7, P10; mục 7 có bảng nhật ký 6 cột; mọi phase ở cột Phase đều có yêu cầu ở mục 6 và `docs/phases/Px.md` tồn tại |

## Lỗi
- **BUG-1 (rất thấp, trình bày):** Bước 6 (F10) viết "Nếu hội đồng hỏi:" giữa dòng (sau câu "`Chốt điểm` lớp 1 không làm trực tiếp…") chứ không ở đầu dòng như bước 1–5, nên `grep -c '^Nếu hội đồng hỏi'` đếm 5 chứ không phải 6. Nội dung và spec E2E đủ nên AC7 vẫn PASS. Gợi ý (BA): xuống dòng cho nhất quán.

## Kiểm chéo
- Dữ liệu cá nhân/secret: mọi email thuộc `@edupilot.local`; không MSSV; không khoá; mật khẩu chỉ nêu bằng tên biến `SEED_DEFAULT_PASSWORD`.
- Khớp seed: tài khoản trong kịch bản có ở ARCHITECTURE §9, đúng vai B (`sv.kha`), C (`sv.nguyco`), D (`sv.moi`); mã lớp có ở ARCHITECTURE.
- Tham chiếu file: `seed/demo/fake_llm.json`, `seed/demo/quy-che-lop2.pdf` chưa tồn tại — đúng, là việc P1 và P6 (mục 6 kịch bản; US "Ngoài phạm vi").
- `DEMO_MODE` chưa có ở ARCHITECTURE §8: kịch bản giao cho P1 (mục 6) — ghi nhận cho P1, không lỗi của story này.
- Q1–Q9 của FEAT-demo-script không chặn story (US nêu rõ).
- Không áp dụng: idempotency, phân trang, migration, 375 px màn mới, diff code.

## Đề nghị
PASS, nhận US-P0-01. Gợi ý nhỏ cho BA ở BUG-1.
