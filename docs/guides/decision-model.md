# Hướng dẫn vận hành: decision model và cascade (D60)

> **Bản phác (techlead, 2026-11-14).** Dev bổ sung số đo thật, ảnh màn `/settings/llm` và tên endpoint chốt trong `openapi.yaml` sau L5–L6 của sprint 10.5. Đánh dấu `[CHỜ L5]` / `[CHỜ L6]` là phần còn thiếu.

Hướng dẫn này dành cho người vận hành EduPilot (ADMIN). Nó trả lời bốn câu: decision model làm gì trong hệ thống, bật / tắt thế nào, chọn ngưỡng ra sao, và khi có sự cố thì xử lý thế nào.

## 1. Decision model làm gì
`gemini-decision-1` là mô hình **chỉ trả lời câu hỏi đóng** (có / không, chọn một trong danh sách) kèm xác suất. Nó không viết chữ. EduPilot dùng nó ở đúng ba chỗ, gọi là **cổng**. Mỗi cổng bật / tắt riêng cho từng lớp và mặc định **tắt**.

| Cổng | Ở đâu | Decision quyết gì | Phân vân thì sao | Tất cả lỗi thì sao |
| --- | --- | --- | --- | --- |
| `THREADS` | Kiểm tra trước khi đăng Hỏi đáp công khai | Bài có hỏi chuyện riêng / nêu tên người cụ thể không. **Chỉ được thêm chặn**, không bao giờ mở lại bài luật đã chặn | Hỏi LLM (9router) | **Không cho đăng** (người dùng thấy "đang bận, thử lại") |
| `INJECTION` | Chấm bài tự luận (việc nền) | Bài có câu nhắm vào người / máy chấm không. Chỉ **gắn cờ** cho giảng viên xem, không đổi điểm | – (cờ là phép OR với regex và LLM) | Bỏ qua lớp decision, chấm vẫn xong |
| `ROUTING` | Chat riêng, bước phân loại câu hỏi | Câu hỏi thuộc loại nào (điểm, điểm danh, lịch thi, nội dung môn…) | Dùng **luật** như trước | Dùng luật như trước |

Decision **không bao giờ** được dùng cho: phát hiện khủng hoảng (luôn bằng luật), điểm, mức rubric, công thức điểm, escalate.

## 2. Bật cho một lớp
1. `/settings/llm` → khối **Decision**: thêm nhà cung cấp loại `decision`, dán khoá Gemini, đặt hạn mức. Free Tier: 200 RPM / 50.000 TPM. Tier 1: 1.500 RPM / 1.000.000 TPM. `[CHỜ L5: ảnh màn]`
2. Gán tuyến `DECIDE` cho model `gemini-decision-1`. Môi trường dev / demo: model `decision-fake`.
3. Chọn lớp → bật từng cổng. Nên bật theo thứ tự: `INJECTION` (rủi ro thấp nhất, chạy nền) → `THREADS` → `ROUTING`.
4. Theo dõi 1–2 ngày trên `/observability` (mục 4) rồi mới bật cho lớp khác.

**Trước khi dùng dữ liệu sinh viên thật:**
- Khoá Gemini phải thuộc **gói trả phí**. Free Tier dùng dữ liệu gửi vào để cải thiện sản phẩm của Google (`PRODUCTION_READINESS.md` §1).
- Nội dung luôn được che tên / MSSV trước khi gửi, nhưng che không phải là ẩn danh hoàn toàn.

## 3. Ngưỡng
Ngưỡng nằm trong tham số của tuyến `DECIDE` và áp chung cho mọi lớp. Mặc định đặt theo PoC ngày 14/11/2026 (`docs/research/poc/jev-decision/cascade.md`).

| Khoá | Mặc định | Ý nghĩa | Tăng / giảm thì sao |
| --- | --- | --- | --- |
| `threads_eps` | 0,05 | Decision nói "không có gì riêng tư" với xác suất "có" ≤ ε thì cho qua ngay; trên ε và dưới 0,5 thì hỏi LLM | Giảm ε: hỏi LLM nhiều hơn (chậm, tốn hơn, ít sót hơn). Không đặt < 0,01 (model không trả dưới 0,01) |
| `injection_tau` | 0,5 | Xác suất ≥ τ thì gắn cờ chèn lệnh | Giảm τ: nhiều cờ hơn, nhiều cờ nhầm hơn |
| `routing_tau` | 0,95 | Độ tự tin ≥ τ thì dùng loại câu hỏi của decision; dưới τ thì dùng luật | Giảm τ: decision quyết nhiều hơn. Ở PoC, 3 câu decision sai có độ tự tin 0,63–0,91, nên **không đặt dưới 0,92** khi chưa đo lại |
| `LLM_DECISION_TIMEOUT` (env) | 800 ms | Quá hạn thì đi đường fallback | Tăng: ít fallback hơn nhưng chat có thể chậm |

Đổi ngưỡng là đổi hành vi. Ghi lý do vào `proposals.md` và đo lại bằng `docs/research/poc/jev-decision/cascade.py` trên dữ liệu đã gắn nhãn.

## 4. Đọc số
`/observability` và `GET /admin/llm/usage?group=cascade` `[CHỜ L5: tên chốt]`. Mỗi cổng hiện:

| Số | Bình thường | Đáng lo |
| --- | --- | --- |
| Tỉ lệ đi đường nhanh (`FAST`) | THREADS ≈ 85–95 %, ROUTING ≈ 80–90 % (theo PoC) | Tụt mạnh: model đổi hành vi, hoặc dữ liệu lớp khác hẳn dữ liệu đo |
| Tỉ lệ `FALLBACK` do lỗi | ≈ 0 | > 5 %: hạn mức, mạng, hoặc khoá hỏng. Xem `error_kind` |
| p95 trễ decision | ≤ 800 ms | Gần 800 ms: nhiều ca hết giờ, chat chậm |
| Chi phí / ngày | ≈ $1 ở 1.000 sinh viên `[ƯỚC TÍNH]` | Vượt hạn mức ngày của lớp / hệ thống |

`[CHỜ L6: bảng TTFT chat trước / sau, trễ precheck, chi phí đo thật]`

## 5. Sự cố
| Triệu chứng | Nguyên nhân thường gặp | Làm gì |
| --- | --- | --- |
| Sinh viên không đăng được bài Threads ("đang bận") | Decision **và** 9router cùng lỗi → cổng THREADS fail-closed | Tắt cổng `THREADS` của lớp: luồng trở về luật như trước. Kiểm 9router (`curl 127.0.0.1:20128/v1/models`) và khoá Gemini |
| `error_kind = RATE_LIMIT` nhiều | Vượt RPM / TPM của gói | Hạ số lớp bật cổng, hoặc nâng gói. Kiểm `rpm_limit` của provider khớp gói |
| `error_kind = AUTH` | Khoá sai / bị thu hồi | Dán khoá mới ở `/settings/llm` |
| `KindBlocked` (lọc an toàn của Google) | Nội dung nhạy cảm | Bình thường: cascade tự đi fallback. Không cần làm gì |
| Chat chậm sau khi bật ROUTING | Decision chậm hơn lần nhúng | Xem p95 decision; tắt `ROUTING` nếu TTFT p95 tăng > 10 % |
| Bài bị chặn nhầm ở Threads | Decision / LLM quá nhạy | Ghi mẫu (đã che) vào bộ E1 `dev`, đo lại, cân nhắc giảm độ nhạy. Không sửa từng bài bằng tay trong DB |

## 6. Chi phí
- $0,10 / 1 triệu token đầu vào; đầu ra không tính tiền.
- Mỗi lời gọi khoảng 170–460 token, tuỳ cổng (PoC).
- Mọi số chi phí ghi vào `llm_audit.cost_est` theo giá đặt ở model, đơn vị đồng / 1 triệu token.
- Dev / test / demo / eval dùng `decision-fake`, không tốn tiền. Mỗi lượt gọi thật phải được PM cho phép (D60 (6)).

## 7. Tham chiếu
- Quyết định: `docs/DECISIONS.md` D60, D47(2).
- Thiết kế mã: `docs/sprints/10.5/techlead.md` TL-1.
- Số đo và ngưỡng: `docs/research/2026-11-14-jev-decision-model.md` mục 6, `docs/research/poc/jev-decision/cascade.md`.
- Tài liệu Google: `ai.google.dev/gemini-api/docs/decisions-api`, `…/models/gemini-decision-1`, `…/pricing#gemini-decision`, `…/rate-limits#gemini-decision`.
