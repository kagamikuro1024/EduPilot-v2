# Prompt cho `research` (Tech Lead): PoC sprint 6

Worktree `/Users/kuro/Documents/TA_Agent_v2-s6`, nhánh `sprint/6-p3-p8`. Kết quả ghi vào `docs/research/2026-10-10-docling-arm64.md`.

## Câu hỏi
1. **`docling-serve` trên colima arm64, chỉ CPU** (`docker info`: 4 CPU, 8 GiB; máy chủ 18 GB, đang dùng chung với project khác):
   - Image chính thức nào chạy được arm64? Image nặng bao nhiêu?
   - RAM lúc nghỉ và đỉnh RAM khi trích từng PDF trong `seed/documents/` (3 tệp: 1,6–3,1 MB).
   - Thời gian trích mỗi tệp.
   - Cấu hình tối thiểu: tắt OCR, tắt mô hình bảng hoặc ảnh nếu tài liệu là PDF chữ; bật lại khi nào thì cần.
   - **Ngưỡng chấp nhận:** RAM ≤ 3 GiB, mỗi PDF ≤ 60 s.
   - Không đạt thì đưa 2–3 phương án, mỗi phương án có số đo. Không tự chọn: D46 khoá `docling-serve`, việc đổi phải do PM hỏi chủ dự án.
2. **Hợp đồng HTTP:** endpoint, đầu ra (markdown hay JSON có cấu trúc), timeout, giới hạn kích thước, cách worker Go gọi (đồng bộ hay async), healthcheck cho compose.
3. **Chunk:** đề xuất kích thước chunk và overlap cho tài liệu tiếng Việt, đơn vị là ký tự hay token. Có số liệu hoặc nguồn chính. Chunk được nhúng 1536 chiều (`content_chunks.embedding vector(1536)`) qua `internal/llm`, embed theo lô.
4. **Tìm lai vector + từ khoá trong PostgreSQL 18:**
   - cách trộn điểm (RRF hay trọng số);
   - từ khoá tiếng Việt không dấu: dùng hàm `vn_fold` đã có (00005) hay `tsvector` `simple`;
   - index cần thêm.

## Luật
- PoC chạy trong `/tmp` hoặc `docs/research/poc/`, dùng container tạm, đặt tên có tiền tố `poc-`.
- Xong thì `docker rm -fv` các container PoC, `docker image rm` image PoC nếu không giữ, rồi `docker volume prune -f`.
- **Không đụng stack `edupilot` đang chạy** (chủ dự án đang xem) và không đụng container `llm-eval-*`.
- Không sửa code, spec hay test.
- Commit `sprint 6: research docling arm64 + RAG`, push, báo PM ≤ 8 dòng: đạt / không đạt, số đo chính, đề xuất. Rồi dừng.
