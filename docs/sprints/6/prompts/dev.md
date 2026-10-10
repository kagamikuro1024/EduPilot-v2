# Prompt cho `dev`: thi công sprint 6 (P3 + P8)

**Nơi làm:** worktree `/Users/kuro/Documents/TA_Agent_v2-s6`, nhánh `sprint/6-p3-p8`.

## Đọc
- `docs/team/DEV.md`.
- `docs/sprints/6/plan.md`: thứ tự story.
- Hai spec **APPROVED**: `docs/specs/FEAT-private-chat-pii/` và `docs/specs/FEAT-docs-calendar/`. Mỗi spec đọc đủ US, SRS, QUESTIONS (cột Trả lời) và mục "Quyết định PM" trong `TL-REVIEW.md`.
- `docs/sprints/6/proposals.md`, phần "Quyết định PM". Mục #7 là cấu hình `docling-serve`, chunk, RRF, cột `tsv`.
- `docs/research/2026-10-10-docling-arm64.md`: lệnh, cấu hình và khối YAML compose.
- `CLAUDE.md`: 16 luật. Riêng sprint này nhớ kỹ luật 2, 3, 4, 11, 12, 16.

## Thứ tự (theo plan)
1. US-P3-01, sau đó làm song song US-P8-01 và US-P3-02.
2. US-P3-03 → US-P3-04 → US-P3-05 → US-P3-06 → US-P3-07.
3. US-P8-02 và US-P8-03. US-P8-03 chỉ cần US-P3-01, nên chen vào được khi đang chờ.
4. Cuối cùng là US-P3-08 (seed, Hôm nay, k6, `gate-p3.sh`, `gate-p8.sh`).

Mỗi story giao như sau:
- test Go và frontend xanh, CI xanh;
- handoff `docs/sprints/6/handoff/dev-<story>.md` ghi đã làm gì, file đổi, lệnh để QC chạy, nợ;
- commit `US-P3-0N: …`, push.

PM sẽ tự giao QC khi thấy handoff. Không chờ QC xong mới làm story tiếp.

## Luật riêng sprint này
- **Danh tính:** tool cá nhân không nhận `student_code` / `user_id` từ LLM, chỉ đọc `trusted_context`.
- **Che danh tính:** `mask` chỉ cắm ở một chỗ trong `internal/llm`; `unmask_stream` đặt ngay trước SSE. Không log ánh xạ, không log nội dung chat. `TestNoPayloadLeak` và `TestUnmaskStream` bắt buộc có.
- **D47:** mỗi tin nhắn một lần phân loại, một lời gọi sinh chữ, không vòng lặp agent. Sự kiện SSE đầu ≤ 300 ms.
- **Migration:** `00007` / `00008` / `00009`, cộng migration thêm `content_chunks.tsv` (đánh số tiếp theo). Không sửa migration cũ. Index phức hợp bắt đầu bằng `course_id`.
- **`docling-serve`:** chỉ bật khi cần (compose profile). Image `ghcr.io/docling-project/docling-serve-cpu:v1.36.0` đã có sẵn trên colima (7 GB), không kéo lại.
- **Stack của chủ dự án:** một stack `edupilot` từ worktree s55 đang chạy để chủ dự án xem. Lần đầu cần dựng stack thì **báo PM trước**; PM sẽ tắt stack đó. Không tự `down` stack của người khác.
- **RAM có hạn:**
  - chỉ một stack chạy;
  - Playwright tối đa 2 worker;
  - `docker volume prune -f` sau mỗi story.
- Phân vân kỹ thuật thì hỏi Tech Lead (`docs/sprints/6/techlead.md`). Spec sai thì ghi `docs/sprints/6/proposals.md`.
