import base from "../../../frontend/playwright.config";
// QC2: cùng cấu hình, ảnh mốc ghi vào thư mục tạm riêng (KHÔNG đụng ảnh mốc Linux trong repo) — dùng cho TC-PU05-01/04/05 trên macOS.
export default { ...base, snapshotPathTemplate: "/tmp/qc2-snap/{arg}{ext}" };
