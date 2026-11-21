# Chạy lại TC của DEF đã sửa (qc2, 2026-11-21)
Bảng DEF → sha: docs/regression/env/def-sha.tsv. ep-r đã dựng lại ở HEAD qc2 (merge sprint/11 @6201b665 + migration 00029). Mỗi DEF: đọc mục DEF trong docs/regression/defects-<sprint>.md (bước tái hiện, TC), chạy lại ĐÚNG TC trên ep-r, ghi P/F có bằng chứng.
- P → đánh dấu ĐÓNG: thêm dòng vào docs/regression/defects-closed.md (DEF, TC, sha, kết quả), và ghi dòng P vào file kết quả của sprint (xem bên dưới).
- F → ghi F + lý do (còn tái hiện) vào file kết quả, bổ sung vào mục DEF («CHẠY LẠI F @HEAD …»).
Kết quả: một dòng TSV  DEF<TAB>TC<TAB>P|F<TAB>ghi chú (số đo/lệnh/đường dẫn bằng chứng) vào docs/regression/evidence-defrerun/results-<tên-agent>.tsv (bản ghi sau thắng). Ảnh/log vào evidence-defrerun/<DEF>/.
