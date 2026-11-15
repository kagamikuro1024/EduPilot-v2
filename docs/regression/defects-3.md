### DEF-R3-01 — Bảng «Mức dùng theo tác vụ» hiện chữ thô «EMBEDDING»
- Mức: Thấp
- Sprint gốc: 3 · TC gốc: TC-P105-44 (AC11 không từ kỹ thuật) · Hồi quy do sprint sau?: KHÔNG RÕ (nhãn tác vụ nằm từ US-P1-05; chữ thô chỉ lộ khi `llm_audit` có dòng EMBEDDING; sprint sau thêm lời gọi nhúng thật nên mới thấy)
- Môi trường: ep-r, be0c0d96, ADMIN, 1440 và 390
- Bước tái hiện: 1. đăng nhập admin@edupilot.local; 2. mở `/settings/llm`; 3. cuộn tới «Mức dùng và ngân sách» → bảng «Mức dùng theo tác vụ». (`GET /admin/llm/usage?group=task` có key `EMBEDDING`.)
- Kỳ vọng: mọi tác vụ có nhãn tiếng Việt, không từ kỹ thuật (regex `/\b(RAG|PII|trace|fallback|embedding|prompt)\b/i` = 0 khớp ở văn bản hiển thị) · Thực tế: hàng «EMBEDDING | 173 lượt | …»
- Bằng chứng: `evidence-3/shots/settings-llm-1440.png`; DOM: `TD:EMBEDDING @usage-section` · Nghi ngờ mã: `frontend/src/features/settings/llm/labels.ts:13` (`taskLabel` chỉ biết 6 tác vụ + DECIDE; `features/observability/api.ts:44` đã có nhãn «Đánh chỉ mục»)
- Chạy lại lần 2: ổn định

### DEF-R3-02 — `docs/design/DESIGN_TOKENS.css` lệch `frontend/src/shared/styles/tokens.css` ở 2 token
- Mức: Thấp
- Sprint gốc: 3 · TC gốc: TC-PU01-01 (kit ⊆ tokens.css) · Hồi quy do sprint sau?: CÓ (góp ý #25 hạ `--ep-ink-3`; US-04 uiux-91 đặt `--ep-slow` 200 ms; kit không cập nhật)
- Môi trường: ep-r, be0c0d96, so tệp
- Bước tái hiện: so `--ep-ink-3` và `--ep-slow` ở hai tệp: kit `oklch(58% …)` / `250ms`; code `oklch(53% 0.014 25)` / `200ms`.
- Kỳ vọng: kit và tokens.css đồng bộ (TC cũ: mọi khai báo kit có mặt y nguyên) · Thực tế: lệch 2 giá trị (TC-PU01-01/03 đã xếp «LỖI THỜI» vì kit đổi hợp lệ 51→62 dòng ở P-107-8/10, P-108-8)
- Bằng chứng: `evidence-3/scripts/` (lệnh `comm` trong pu01.py) · Nghi ngờ mã: `docs/design/DESIGN_TOKENS.css`
- Chạy lại lần 2: ổn định

### DEF-R3-03 — `pnpm lint` có 5 cảnh báo (nền sprint 3 = 0)
- Mức: Thấp
- Sprint gốc: 3 · TC gốc: TC-PU01-05, TC-GATEPU-01 · Hồi quy do sprint sau?: CÓ (10.x)
- Môi trường: bản sao `frontend/` ở be0c0d96, `pnpm lint` rc=0
- Bước tái hiện: `cd frontend && pnpm lint` → `✖ 5 problems (0 errors, 5 warnings)`: `src/app/dev/ui/cells.tsx:39` (`text` không dùng), `features/grading/QueueTab.tsx:115,123` (react-hooks/exhaustive-deps `setFlagOnly`, `setStatus`), `features/landing/Landing.tsx:48` (`<img>`), `features/settings/llm/LlmSettings.tsx:13` (`s` không dùng)
- Kỳ vọng: 0 lỗi / 0 cảnh báo mới so nền · Thực tế: 5 cảnh báo
- Bằng chứng: `evidence-3/lint.out` · Nghi ngờ mã: các dòng trên
- Chạy lại lần 2: ổn định

### DEF-R3-04 — `DecryptKey(` được gọi ngoài `internal/llm/` và `internal/llmconfig/`
- Mức: Thấp
- Sprint gốc: 3 · TC gốc: TC-P101-24 (AC8) · Hồi quy do sprint sau?: KHÔNG (có sẵn từ US-P1-04, commit 25c0bf24 — sau lượt QC P1-01 vốn báo grep = 0)
- Môi trường: grep mã
- Bước tái hiện: `grep -rn 'DecryptKey(' backend-go --include=*.go | grep -v _test.go | grep -v internal/llm/ | grep -v internal/llmconfig/` → `internal/httpapi/llmhttp/admin.go:242`, `verify.go:302` (kiểm kết nối khi sửa nhà không đổi khoá)
- Kỳ vọng: 0 dòng (khoá rõ chỉ ở llm/llmconfig) · Thực tế: 2 dòng ở lớp HTTP; không có rò khoá (TC-P104-16/17 đều 0)
- Bằng chứng: kết quả grep · Nghi ngờ mã: như trên
- Chạy lại lần 2: ổn định
