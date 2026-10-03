# Sprint 3 — báo cáo (PU Nền giao diện + P1 LLM Gateway)

Mục tiêu: trọn `docs/phases/PU.md` (nền frontend dùng chung cho mọi màn) và `docs/phases/P1.md` (cổng LLM Go: provider tương thích OpenAI, Scheduler ba làn, cầu dao, audit, `/settings/llm` thật) · Kết quả: **10/10 story PASS · cổng PU PASS (LCP có điều kiện) · cổng P1 PASS** sau 1 vòng sửa cổng · Nhánh `sprint/3-pu-p1` (từ `main` sau sprint 2)

| Story | Phase | Trạng thái | QC (TC) |
| --- | --- | --- | --- |
| US-PU-01 Token `--ep-*`, phông Be Vietnam Pro, primitive `shared/` | PU | PASS sau sửa | `qc/report-US-PU-01.md` (37) |
| US-PU-02 Primitive tương tác: Button, Field, Dialog / Drawer / ConfirmIrreversible, DataTable | PU | PASS sau sửa | `qc/report-US-PU-02.md` (48) |
| US-PU-03 Tầng dữ liệu: `apiClient`, TanStack Query, `useSSE`, `PageState`, `useAutosaveDraft`, `useUndoableAction` | PU | PASS sau sửa | `qc/report-US-PU-03.md` (58) |
| US-PU-04 Khung ứng dụng: điều hướng theo vai, bảng "Thêm" mobile, cổng token dev bị loại khỏi build thường | PU | PASS sau sửa | `qc/report-US-PU-04.md` (60) |
| US-PU-05 Cổng UI: ảnh mốc, axe, Lighthouse CI, `ci/ui-drift` | PU | PASS sau vòng sửa cổng | `qc/report-US-PU-05.md` (45) |
| US-P1-01 Migration `00002_llm`, mã hoá khoá AES-256-GCM, registry provider | P1 | PASS sau sửa | `qc/report-US-P1-01.md` (36) |
| US-P1-02 `internal/llm` trên openai-go/v3: chat / stream / structured / embed, fallback, replay | P1 | PASS sau sửa | `qc/report-US-P1-02.md` (51) |
| US-P1-03 Scheduler ba làn + hạn mức + cầu dao + deadline | P1 | PASS sau sửa | `qc/report-US-P1-03.md` (51) |
| US-P1-04 `llm_audit`, ngân sách, API quản trị | P1 | PASS sau sửa | `qc/report-US-P1-04.md` (49) |
| US-P1-05 `/settings/llm` thật (nhà cung cấp, tuyến + hoàn tác, dự phòng, nhúng, mức dùng, ngân sách) | P1 | PASS | `qc/report-US-P1-05.md` (58) |

Cổng: `qc/report-GATE-PU.md` (18 TC), `qc/report-GATE-P1.md` (27 TC). Spec: `docs/specs/FEAT-ui-foundation/` v1.3, `docs/specs/FEAT-llm-gateway/` v1.6.

## Số liệu
- Test case: **538** (PU 248 · P1 245 · GATE 45). Vòng 1 các story: 14 lỗi → sửa hết; cổng vòng 1: FAIL (CI đỏ, `llmload`) → vòng sửa 1 → PASS.
- Playwright 155 ca, 0 đỏ, không retry; axe 0 critical / 0 serious ở 102 lượt quét; `audit.mjs` 674 PASS / 0 FAIL; `proto-curl.sh` 493 PASS.
- Lighthouse (mobile mô phỏng): CLS 0, JS ≤ 230 KB, TBT sáu route thật ≤ 111 ms; **LCP 2,7–3,4 s** (ngưỡng 2,5 s — xem Nợ).
- Nhà cung cấp thật: dev ghi replay OpenAI `gpt-4o-mini` + Gemini flash, nhúng 1536 chiều. GATE-P1 TC-13 / TC-18 (QC): hai nhà trả lời, tắt nhà chính thì nhà kia trả lời (`fallback_index: 0`), ghi / phát lại PASS; QC tốn 13 lời gọi, ≈ 1.100 token. Anthropic BLOCKED (không có khoá).
- 95 commit, `backend-go` + `frontend` +24.680 dòng. Góp ý #1–#34, PM chấp nhận cả 34.

## Lỗi thật tìm ra và đã sửa
- **LLM:** HTTP 504 xếp sai loại (BUG-P102-1); `Structured` thử sai thứ tự phương thức (BUG-P102-2 → #1: mỗi loại provider một phương thức cố định, một lời gọi).
- **Scheduler:** chat hết hạn 30 s nhận "kết nối đóng" thay vì `504 DEADLINE_EXCEEDED` (BUG-P103-1). BATCH chiếm hết 10 chỗ khiến chat đầu chờ ~720 ms (BUG-P103-2 → #6: BATCH ≤ `ceil(MAX × share)`). Hạn của chính yêu cầu bị tính là lỗi nhà cung cấp và làm mở cầu dao (vòng sửa cổng).
- **API:** số vượt biên cột `numeric` trả 500 (BUG-P104-1); `APP_ENCRYPTION_KEY` sai định dạng vẫn được nhận (BUG-P101-1).
- **Frontend:** focus hàng DataTable, hộp thoại không khoá cuộn nền (BUG-PU02-1/2); `traceId` bỏ qua `X-Request-Id`, `reconnect` mở 2 kết nối (BUG-PU03-1/2); bảng "Thêm" không đóng bằng vuốt / bấm ngoài (BUG-PU04-1); ảnh mốc lệch trên CI Linux (BUG-PU05-1). Chữ phụ `--ep-ink-3` không đạt tương phản WCAG AA ở mọi route (#25).
- **Hạ tầng test:** test rò volume Docker (157 volume, 38 GB) do Ryuk tắt + `docker rm` thiếu `-v`; đã sửa và thêm luật dọn định kỳ.

## Quyết định trong sprint
- openai-go/v3; `Structured` một phương thức cố định theo loại provider (#1). Image gateway ≤ 80 MB, worker < 40 MB (#5). BATCH luôn ≤ `ceil(MAX × share)` chỗ (#6).
- `--ep-ink-3` hạ còn `oklch(53% 0.014 25)` (#25). Phông còn 400 / 600 / 700 (#26). Ma trận `/dev/ui` 25 khối (#29). TBT của trang chỉ-dev `/dev/ui` ở `warn` (#34).
- `fallback_index` đếm trong chuỗi provider **đang bật**: Admin tắt nhà chính không tính là dự phòng (#33).
- Vai mới: research kiêm **Tech Lead**, dev hỏi kỹ thuật qua `docs/sprints/N/techlead.md`.

## Nợ
- **LCP > 2,5 s** trên Lighthouse mô phỏng (đo thật 0,49 s): assertion LCP `warn` tạm; **cổng P2 (sprint 4) phải đưa về `error` ≤ 2,5 s** khi mock rời bundle layout (#26).
- Ghi replay Anthropic khi có khoá.
- `gate-pg.sh` không được QC chạy vì nó `down -v` stack test dùng chung — tách tên stack cho QC (P10).

## Chủ dự án tự kiểm
- `pnpm -C frontend dev`, mở `/dev/ui` ở 375 px và 1440 px: 25 khối, mỗi khối đủ trạng thái tải / rỗng / lỗi.
- `/settings/llm` (Admin): thêm khoá OpenAI + Gemini, kéo thứ tự dự phòng, tắt nhà chính rồi hỏi thử — vẫn có trả lời, `fallback_index: 0`.
- `pnpm -C frontend build` rồi tìm `Dán token` trong `.next/static`: không có.
- Đọc diff `backend-go/internal/llm/scheduler` — phần mọi lời gọi LLM về sau đi qua.
