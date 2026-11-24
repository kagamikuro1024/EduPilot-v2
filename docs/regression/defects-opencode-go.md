# Lỗi OpenCode Go (P-11-106…110) — qc3

## DEF-OC-SI-01 — Thấp — log `structured_invalid` còn chép một phần nội dung đầu ra của mô hình
Ca: TC-OC-70 (US-P1-02 AC10: «không nội dung đầu ra»). Mã: `sprint/11-p10-eval` @8f548396 (dev `9e738647`).
Tái hiện: `evidence-oc/zz_qc3_si_test.go` (đặt vào `backend-go/internal/llm`, `go test -run TestQC3_OC70 -v`).
- Đầu ra `{"label":"hoi_dap","confidence":0.5,"BIMAT123":1}` → `err="$: trường thừa \"BIMAT123\" (additionalProperties=false)"` — tên trường do mô hình sinh nằm trong log.
- Đầu ra `BIMAT123 không phải json` → `err="không phải JSON hợp lệ: invalid character 'B' looking for beginning of value"` — ký tự đầu của đầu ra nằm trong log (BA đã nêu rủi ro này ở P-11-110).
Đúng: sai enum / sai kiểu / thiếu trường / cụt ở trần / rỗng — chỉ đường dẫn + loại lỗi.
Đề xuất: với «trường thừa» chỉ ghi `$: trường thừa (additionalProperties=false)`; với lỗi cú pháp JSON chỉ ghi loại («không phải JSON hợp lệ: ký tự không mong đợi / kết thúc đột ngột») không kèm ký tự. Test của dev (`TestStructuredInvalidLogsTokensNotContent`) chỉ kiểm giá trị chuỗi mật ở thuộc tính `label`, nên không bắt được hai nhánh này.

## Quan sát (không đổi nhãn)
- Form «Thêm nhà cung cấp» khi tên trùng: UI hiện «Dữ liệu vừa thay đổi. Hãy tải lại và thử lại.» (409 `CONFLICT` dùng thông điệp chung); API trả đúng 409 «Đã có nhà cung cấp trùng tên».
- Đã đóng sau P-11-108/109 và dev: Test kết nối 10 s → 30 s (TC-OC-57/62/63 đạt); 429 không tính mạch (TC-OC-42); 400 model lạ không chuyển nhà (TC-OC-54); `llm_audit.provider` khi degraded (TC-OC-58); `BAD_RESPONSE` tính mạch (TC-OC-61).
