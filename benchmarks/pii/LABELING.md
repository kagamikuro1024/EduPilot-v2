# E1 — bộ dữ liệu gắn nhãn PII (US-P3-07, SRS FEAT-private-chat-pii 9.3)

- `e1_dataset.jsonl`: 200 mẫu (100 dương `expect_block=true`, 100 âm), nhóm S1…S8 / N1…N5 đúng bảng SRS 9.3; `split` `dev` 60 / `test` 140 phân tầng theo nhóm (mẫu rải đều theo thứ tự trong nhóm, **không** chọn theo kết quả chạy).
- Dữ liệu **mô phỏng** (D44): tên / MSSV / email lấy từ roster seed lớp 1 (`roster_seed.json`, dựng bằng `node benchmarks/pii/dump_roster.mjs`, cùng hạt giống `SEED_RNG` với `scripts/seed.mjs`); SĐT, CCCD, email ngoài roster là số / địa chỉ bịa; tên ngoài roster liệt kê ở `outside_roster_names.json` (nhóm S7).
- Ô điền: `{{SVnn.name}}`, `.name.noaccent`, `.name.reversed` (đảo hoàn toàn thứ tự âm tiết), `.mssv`, `.mssv.spaced` (`2022 4786`), `.mssv.dotted` (`2022.4786`), `.email`; `nn` là số thứ tự trong `roster_seed.json`.
- Mẫu `dev` cho phép chỉnh luật / ngưỡng; mẫu `test` **không** được đọc khi chỉnh. `eval_pii.py` và `TestE1DatasetMeetsGate` chỉ in lỗi của tập `dev`.

## Soát độc lập (QC)
QC soát ≥ 50 mẫu (≥ 25 dương, ≥ 25 âm); bất đồng nhãn → BA phán và ghi ở bảng dưới.

| id | nhãn Dev | nhãn QC | BA phán | ghi chú |
| --- | --- | --- | --- | --- |
