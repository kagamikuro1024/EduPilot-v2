**Ca trượt (F=9):** TC-PE03-33 → DEF-R5-10; TC-PE03-49 → DEF-R5-03; TC-PE03-51 → DEF-R5-04 (chạy lại ở HEAD mới vẫn tái hiện); TC-PE06-27 → DEF-R5-06; TC-PE07-18 → DEF-R5-07; TC-PE08-04 → DEF-R5-11; TC-PE08-31 → DEF-R5-08; TC-PE08-38 → DEF-R5-09; TC-PU06-17 → DEF-R5-12. DEF-R5-05 (contract không tag) đã sửa ở 562f5a45 → TC-PE01-39/PE02-57 sang P. TC-PE04-40: lượt đầu ghi F do lỗi đo (epoch trừ 0), đo lại đạt (xem defects-5.md).

## Còn chưa chạy (S=33) — lý do và việc cần làm
- Cần DB sạch (stack tạm seed từ đầu, ≈12 phút + máy rảnh): TC-PE09-02, 03, 07, 12, 13 (class 761987 ở ep-r đã bị QC khác/sprint sau mở rộng: «Mật mã» có 456 lượt thay 24).
- k6 / tải (binary có: /opt/homebrew/bin/k6; chưa chạy vì tải máy và lệnh dừng): TC-PE02-48, 49, PE05-25, PE09-15, 16, 17, 19.
- Cổng/E2E tổng (gate-pe.sh, exam.spec.ts @real, F19 trọn, audit.mjs 4 vai): TC-PE04-50, PE05-60, PE06-47, PE08-68 (phần Go + lint + build đã đạt, thiếu audit.mjs), PE08-67, PE09-11, 20, 21, 22, 23, 25, 26, 28 (phần grep đã làm: chỉ định danh/chú thích; còn quét innerText), 31.
- PE02-25 (kill -9 worker khi đang chấm, cần khoá toàn cục), PE03-36 (nhà cung cấp AI giả: bỏ câu sai / chưa cấu hình / quá tải — có test Go `jobs_test.go` chưa chạy), PU06-11, 14, 16, 18 (spec shell/account/lcp/a11y/today chưa chạy; đo tay đã có trong `evidence-5/lh/`).
- Môi trường dùng chung: kịch bản/dữ liệu QC của lượt này: lớp R5C4 (SV r5x) + bài «R5M …» trong R5C1; 5.000 SV `qc2m5-bulk-*` đã xoá sau khi đo.
