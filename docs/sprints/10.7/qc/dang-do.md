# QC 10.7 — điểm dừng (chủ dự án tạm dừng, sau P-107-15)

**Đang dở:** không có job nền. Đã kiểm lại P-107-15 ở `be299a3e` (xem `summary.md` mục 7): chỉ còn `Esc` không đóng lý do công tắc khoá ở `documents` (Thấp).
**Bước tiếp:** (1) chạy e2e dev `a11y`, `documents`, `frame`, `panels`, `private-chat`, `decision-observability` ở HEAD mới (`pnpm run build:gate` rồi `playwright test … --project=desktop --project=mobile --retries=0` trong `~/Documents/TA_Agent_qc5/frontend`); (2) khi dev sửa `Esc`: chạy `QC_DOC_CLASS=761987|761988 playwright test -c playwright.qc-misc.config.ts -g documents` (spec `frontend/e2e/qc-docs-real.spec.ts`, copy sang qc5 trước); (3) chốt `summary.md` (verdict từng US) rồi báo PM. Stack `ep-qc` đang chạy (frontend `be299a3e`, env QC); dọn khi PM báo.
