# Prompt cho `ba` — sprint 1, bản 3 (sau D45–D48)

Bỏ bản 2. Đọc: `docs/DECISIONS.md` D45, D46, D47, D48; `docs/phases/P0.md`; `docs/sprints/1/plan.md` (bản 3). Lưu ý: `CLAUDE.md`, `ARCHITECTURE.md`, các phase file đang được PM cập nhật theo D46 (bỏ Python); chỗ nào còn nói service Python AI / gRPC / `src/` thì D46 thắng.

## Việc
1. Xoá `docs/specs/FEAT-ci-mailhog/` (lỗi thời) và mọi file dở của bản 2.
2. Viết spec hạ tầng: `US.md` + `SRS.md` rút gọn (mục 1, 4, 6, 8, 9, 11; mục khác ghi "Không áp dụng — story hạ tầng") + `QUESTIONS.md`.

| Feature | Story | Nguồn |
| --- | --- | --- |
| FEAT-scaffold | US-P0-02 | P0 L1; D45, D46, D48; ARCHITECTURE §2 cấu trúc `backend-go`, mục env |
| FEAT-ci | US-P0-03 | P0 L2; CLAUDE.md "Lệnh" |

- AC trong `plan.md` (bản 3) là sàn; viết thành Given/When/Then, mỗi AC kèm lệnh kiểm chạy được.
- FEAT-scaffold SRS mục 4: bảng đầy đủ từ `git ls-files` ở gốc repo — mỗi mục gốc: dời vào `legacy/`, giữ nguyên, hay dời sang `seed/documents/` (PDF môn học trong `data/`, D10). Không bỏ sót mục gốc nào.
- Stack local (D48): postgres 18 + pgvector, redis 8, minio, mailhog, gateway Go, frontend Next.js 16. Không service Python. `docling-serve` chưa thêm (P8).
- Frontend: Next.js 16 App Router + React 19 + pnpm, Node 24; dùng lại `frontend/src/shared/styles/tokens.css` và `frontend/public/brand/`.
- AC phân quyền: "không áp dụng — chưa có API nghiệp vụ" kèm lý do.

## Khi xong
Commit `sprint 1: spec FEAT-scaffold, FEAT-ci` (chỉ `docs/specs/**`), tóm tắt ≤ 10 dòng, dừng.
