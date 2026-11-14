# Sprint 10.6: Harness học từ lần giảng viên sửa (bộ nhớ agent)

Chủ dự án duyệt 2026-11-14 (báo cáo `docs/research/2026-11-14-agent-memory-tham-dinh-pm.html`, research `docs/research/2026-11-14-agent-memory.md` trên `research/agent-memory`). Làm **ngay sau sprint 10.5** (decision model), rồi chạy một lượt kiểm + bench 1.000 bài chung, rồi sprint 11 một mạch.

Nhánh `sprint/10.6-guidance`, tách từ `sprint/10.5-decision` (luồng chấm đã sửa ở 10.5 L3); dev bắt đầu viết mã khi 10.5 xong mã (L1–L6), tài liệu làm song song ngay.

## Phạm vi (gói «harness», chủ dự án chọn cả bốn)
| # | Nội dung | Cỡ |
| --- | --- | --- |
| M1 | **Lưu ý chấm theo bài tập**: job `grading.reflect` (BATCH, `llm.Structured`, qua 9router) rút lưu ý từ các lần GV sửa điểm nháp (≥ 5 bài đã duyệt; KHÔNG đọc văn bản bài; đầu ra qua `privacy.Detect`) → lưu ý `DRAFT` → GV duyệt từng cái → phiên bản rubric mới; mỗi lượt chấm ghim `guidance_version`; đổi phiên bản → «N bài chấm theo bản cũ · Chấm lại»; không công bố lô trộn hai phiên bản; lưu bất biến `ai_comments` | M |
| M2 | **Mỗi lần sửa → ca hồi quy** (đã che, không danh tính) vào golden set E2 / E3; chạy lại khi đổi prompt / model, không cho lùi | S |
| M4 | **Threads: câu tương tự đã có đáp án được xác nhận** gợi ý lúc soạn câu (từ `knowledge_items` sẵn có) | S |
| M6 | **Mẫu câu bị loại** (thống kê lý do `REJECTED`) đưa vào prompt sinh câu hỏi lần sau | S |

Mặc định **tắt theo lớp** (cờ), giống D60. Không đụng điểm tổng / trừ muộn / làm tròn (code `decimal`). Không bộ nhớ theo sinh viên (M11, M12 không làm).

## ADR (chủ dự án đã duyệt hướng; techlead-mem viết vào `docs/DECISIONS.md`)
- **D61** Không dùng hệ bộ nhớ agent bên ngoài (Hindsight…); bộ nhớ = dữ liệu có cấu trúc trong Postgres + người duyệt. Lý do: D46, luật 3, D47, PII.
- **D62** Lưu ý chấm là phiên bản rubric do giảng viên duyệt, ghim theo lượt chấm; không tự áp; chấm lại lô chưa công bố khi đổi phiên bản.

## Kiểm chứng bằng «Giáo viên» tạm (chủ dự án yêu cầu)
PM tạo agent tạm `giaovien` (model `claude-fable-5-2`, effort medium) đóng vai giảng viên thật, CHỈ trong đợt kiểm chứng: chấm tay bộ bài E3 (≥ 60 bài, nhãn người), duyệt / bỏ lưu ý trên màn thật, sửa điểm nháp như giảng viên. Không xem mã, không đọc nhãn kỳ vọng. Xong kiểm chứng → PM tắt. Kết quả đo: MAE nháp trước / sau trên bài tập thứ hai cùng lớp, lệch trung bình (chống chấm gắt quá tay), số lần GV sửa tay.

## Tài liệu (bắt buộc, viết kỹ)
| Tài liệu | Ai |
| --- | --- |
| `docs/specs/FEAT-grading-guidance/` US + SRS (11 mục, sơ đồ tuần tự vòng học, bảng lỗi, quyền, cấu hình) + QUESTIONS | ba |
| `docs/DECISIONS.md` D61, D62; `ARCHITECTURE.md` (bảng, job, cờ); `SYSTEM_DESIGN.md` (làn BATCH, chi phí) | techlead-mem |
| `docs/FLOWS.md` F9 chấm bài (vòng lưu ý), F4 Threads (câu tương tự), luồng sinh câu hỏi | ba |
| `docs/guides/grading-guidance.md` (vận hành: bật cờ, duyệt lưu ý, chấm lại, đọc số đo, quên / xoá) | dev + techlead-mem |
| `openapi.yaml` | dev |

## Thứ tự
BA spec + techlead-mem TL-1 (thiết kế mã) song song ngay → QC TC → dev L1 migration → L2 job reflect → L3 prompt ghim phiên bản → L4 màn Duyệt bài → L5 M2 / M4 / M6 → L6 tài liệu + guide → QC kiểm từng lát → kiểm chứng với `giaovien` → merge.
