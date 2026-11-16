# Bản nháp lỗi — QC2 hồi quy sprint 5 / 5.5 (phần hiệu năng + tĩnh)

| Ca | Triệu chứng | Tái hiện | Mức | Sprint gốc | Do sprint sau? |
| --- | --- | --- | --- | --- | --- |
| TC-PU06-17 / TC-UI7-06 | `pnpm -C frontend lint` rc=1: 32 lỗi (no-explicit-any, no-require-imports) + 15 cảnh báo ở 10 tệp QC `frontend/e2e/qc2-r4,r8,r10,r91/*` đã commit; CI chạy đúng lệnh này ⇒ đỏ. Mã sản phẩm sạch. | `cd frontend && pnpm lint` ở HEAD eeffb053 (logs/pq/lint.log) | Trung bình | 5 / 5.5 (cổng lint) | Có — tệp QC của lượt hồi quy, không phải mã sản phẩm; sửa: ignore `e2e/qc2-*` trong eslint.config.mjs hoặc sửa kiểu |
| TC-PU06-17 | `next build` in cảnh báo "metadataBase property in metadata export is not set" (TC đòi 0 cảnh báo) | `pnpm -C frontend build` (logs/pq/build.log); nguồn `app/page.tsx` openGraph | Thấp | 5 (PU-06) | Có — trang giới thiệu 9.1/10.7 |
| Ghi nhận (không FAIL, #26/#14) | LCP `/exams/<id>/take` mobile 3,0 s (simulate) / 5,5 s (devtools) > 2,5 s; phần tử `<p integrityText>` hiện sau khi dữ liệu bài thi về | lh/take_lh.mjs nav | Thấp | 5 (PE-09) | — |
| Ghi nhận | TBT 12×: /settings/llm 688, /chat 524, /inbox 478 ms (5.5 dev: 118–163), nhưng máy bi ≈ 3,0k + tải ~9; cổng CI hiện tại (multiplier theo bi) 24–60 ms ĐẠT | lh/run.sh tbt12 5 simulate 12 … | Thấp | 5 / 5.5 | Có — P-11-46 đổi cách đo |
| Ghi nhận | `/gradebook` JS 248,4 KB, còn 7,5 KB tới trần 256000 (+13 KB so với sprint 5) | lh/sim5 | Thấp | 5 | Có — 9.x/10.x |
| Ghi nhận | PROGRESS.md không còn dòng "sprint 5.5"/"UI-0x"/"Chủ dự án đã xem… gật" (TC-UI7-09, LT: P-107-10) | grep | Thấp | 5.5 | Có |
