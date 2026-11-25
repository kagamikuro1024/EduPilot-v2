# E1 — bộ dữ liệu gắn nhãn PII (US-P3-07, SRS FEAT-private-chat-pii 9.3)

- `e1_dataset.jsonl`: 200 mẫu (100 dương `expect_block=true`, 100 âm), nhóm S1…S8 / N1…N5 đúng bảng SRS 9.3; `split` `dev` 60 / `test` 140 phân tầng theo nhóm (mẫu rải đều theo thứ tự trong nhóm, **không** chọn theo kết quả chạy).
- Dữ liệu **mô phỏng** (D44): tên / MSSV / email lấy từ roster seed lớp 1 (`roster_seed.json`, dựng bằng `node benchmarks/pii/dump_roster.mjs`, cùng hạt giống `SEED_RNG` với `scripts/seed.mjs`); SĐT, CCCD, email ngoài roster là số / địa chỉ bịa; tên ngoài roster liệt kê ở `outside_roster_names.json` (nhóm S7).
- Ô điền: `{{SVnn.name}}`, `.name.noaccent`, `.name.reversed` (đảo hoàn toàn thứ tự âm tiết), `.mssv`, `.mssv.spaced` (`2022 4786`), `.mssv.dotted` (`2022.4786`), `.email`; `nn` là số thứ tự trong `roster_seed.json`.
- Mẫu `dev` cho phép chỉnh luật / ngưỡng; mẫu `test` **không** được đọc khi chỉnh. `eval_pii.py` và `TestE1DatasetMeetsGate` chỉ in lỗi của tập `dev`.

## Soát độc lập (QC)
QC soát ≥ 50 mẫu (≥ 25 dương, ≥ 25 âm); bất đồng nhãn → BA phán và ghi ở bảng dưới.

| id | nhãn Dev | nhãn QC | BA phán | ghi chú |
| --- | --- | --- | --- | --- |
| S6-13 (M13) | CHẶN | CHO QUA | **CHẶN** | Nhóm S6 «câu hỏi riêng tư không định danh» (SRS FEAT-private-chat-pii 9.3, dương): hỏi chuyên cần của chính mình, không lộ ai nhưng là dữ liệu học tập riêng của người hỏi; Q4 (chủ dự án) và PRD M1 chốt «hỏi điểm cá nhân không bao giờ lưu công khai», lý do `PERSONAL_QUESTION`, chuyển sang chat riêng. Nhãn QC giữ nguyên (đúng theo định nghĩa trong hướng dẫn, vốn chỉ nêu «lộ dữ liệu của một người cụ thể») |
| S6-01 (M22) | CHẶN | CHO QUA | **CHẶN** | Nhóm S6 «câu hỏi riêng tư không định danh» (SRS FEAT-private-chat-pii 9.3, dương): hỏi điểm của chính mình, không lộ ai nhưng là dữ liệu học tập riêng của người hỏi; Q4 (chủ dự án) và PRD M1 chốt «hỏi điểm cá nhân không bao giờ lưu công khai», lý do `PERSONAL_QUESTION`, chuyển sang chat riêng. Nhãn QC giữ nguyên (đúng theo định nghĩa trong hướng dẫn, vốn chỉ nêu «lộ dữ liệu của một người cụ thể») |
| S6-26 (M28) | CHẶN | CHO QUA | **CHẶN** | Nhóm S6 «câu hỏi riêng tư không định danh» (SRS FEAT-private-chat-pii 9.3, dương): hỏi tình trạng thi lại của chính mình, không lộ ai nhưng là dữ liệu học tập riêng của người hỏi; Q4 (chủ dự án) và PRD M1 chốt «hỏi điểm cá nhân không bao giờ lưu công khai», lý do `PERSONAL_QUESTION`, chuyển sang chat riêng. Nhãn QC giữ nguyên (đúng theo định nghĩa trong hướng dẫn, vốn chỉ nêu «lộ dữ liệu của một người cụ thể») |
| S6-38 (M39) | CHẶN | CHO QUA | **CHẶN** | Nhóm S6 «câu hỏi riêng tư không định danh» (SRS FEAT-private-chat-pii 9.3, dương): khiếu nại điểm của chính mình, không lộ ai nhưng là dữ liệu học tập riêng của người hỏi; Q4 (chủ dự án) và PRD M1 chốt «hỏi điểm cá nhân không bao giờ lưu công khai», lý do `PERSONAL_QUESTION`, chuyển sang chat riêng. Nhãn QC giữ nguyên (đúng theo định nghĩa trong hướng dẫn, vốn chỉ nêu «lộ dữ liệu của một người cụ thể») |

**Nhận xét của BA (E1, 57 mẫu giaovien, 53/57 đồng ý):** cả 4 bất đồng đều thuộc nhóm S6, và S6 có đúng 4 mẫu trong gói 57, nên QC gán S6 = CHO QUA ở 4/4 mẫu. Đây là lệch **định nghĩa** của hướng dẫn gán nhãn (`HUONG-DAN-CHU-DU-AN.md` mục E1: CHẶN chỉ gồm «lộ dữ liệu cá nhân của một người cụ thể»), không phải nhiễu của người gán. Phán xử: nhãn máy (`expect_block = true`) đúng cả 4; không đổi nhãn của giaovien. Nếu soát lại mà muốn số «đồng ý» phản ánh sản phẩm, thêm vào định nghĩa CHẶN: «hoặc hỏi điểm, chuyên cần, thi lại, phúc khảo của chính mình (Q4)» rồi cho gán lại riêng 4 mẫu S6; việc này không do BA quyết.

## `--strict-synthetic`
Ngoài ô điền và `outside_names`, script quét mọi cụm ≥ 2 từ viết hoa chữ đầu trong văn bản (đã điền ô) và từ chối cụm không thuộc roster seed (`roster_seed.json`), danh sách bịa (`outside_roster_names.json`) hay danh sách từ thường viết hoa (`benign_capitalized.json`: Hà Nội, Hoa Kỳ…). So khớp không phân biệt dấu / thứ tự âm tiết, bỏ từ đầu câu viết hoa ("Bạn Trần Khánh Ly" khớp "Trần Khánh Ly"). Muốn thêm tên bịa: ghi vào `outside_roster_names.json`.
