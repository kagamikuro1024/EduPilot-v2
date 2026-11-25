# TC-G10-10 / G10-12 / G10-13 — bộ mẫu QC, nhãn tạm (qc2, 25/11)

Lệnh PM 25/11: kết P hoặc F, không chờ nhãn người; áp ngưỡng AC6 không nới.

## Nguồn dữ liệu và nhãn
| Mục | Giá trị |
| --- | --- |
| Bài | BT-A «Sự cố lộ dữ liệu…» (KC106, 60 bài: kc001–kc020 học, kc021–kc060 đo, n = 40) và BT-B đối chứng (24 bài); `benchmarks/guidance/kc/data/` |
| **Nguồn nhãn** | **QC tạm: qc2 (agent, đóng vai «Giáo viên» theo lệnh PM 25/11) chấm theo `rubric.json`, mỗi tiêu chí bước 0,25.** Không phải nhãn người. Không dùng làm nhãn cho luận văn. `labels.csv` (64 dòng: 40 BT-A đo + 24 BT-B) |
| Chấm mù | Chấm từng bài trước khi mở điểm AI, hồ sơ ẩn `profile.jsonl` hay `labels.csv` của giaovien 14/11. **Ngoại lệ phải nói:** trước khi chấm tôi đã vô tình in 2 dòng đầu `labels.csv` của giaovien bằng `head` (kc021 = 1,75 / 2,00 / 2,25 / 0,75 = 6,75; kc022 = 1,75 / 1,00 / 1,25 / 2,00 = 6,00). kc021 tôi chấm từ bài ra đúng cùng bốn số; kc022 tôi chấm 1,50 / 0,75 / 1,00 / 2,00 = 5,25 theo bài. kc023 và kc024 của giaovien tôi chỉ thấy SAU khi đã ghi xong nhãn của mình (8,00 và 8,00; của tôi 7,50 và 8,00) |
| Nháp AI | **Không gọi LLM.** Ảnh chụp `drafts-b0r1, b0r2, b1r1, b1r2, bt-b-after` của kiểm chứng 14/11 (`benchmarks/reports/guidance/kc-2026-11-14/`, model chấm 9router ag/gemini-3.8-flash, B1 = bản lưu ý do giaovien duyệt). Sao nguyên vào `drafts/` |
| Stack | ep-r (gateway @e6f89f6c); nạp lớp KC106 bằng `load.py` (84 bài MATCHED, chưa chấm, `load.log`). `run.py` cần đăng nhập TEACHER và đọc `/guidance` nên cần gateway; `llm_audit` không tăng trong lúc đo (9748 trước / sau nạp) |
| Lệnh | `API_URL=https://localhost:1143/api/v1 BENCH_OUT=docs/regression/evidence-g10/out python3 benchmarks/guidance/run.py --labels docs/regression/evidence-g10/labels.csv --drafts docs/regression/evidence-g10/drafts --label-source "QC tạm …" --provider "không gọi LLM …"` → `run-g10.log` (thoát 1 = Giữ tắt) |

## TC-G10-10 (AC5): P
Báo cáo `out/guidance-before-after.{json,md,svg}` có đủ: MAE tiêu chí 0,492 → 0,414; MAE tổng 1,856 → 1,244; lệch TB có dấu (AI − GV) +1,856 → +1,156 (AI chấm rộng tay hơn QC); tiêu chí phải sửa tay 74 → 62; bài tốt lên / xấu đi 37 / 2; tiêu chí vốn đúng nay sai 9 (nhiễu 5); nguồn nhãn «QC tạm …, không phải nhãn người»; n = 40; mục «Giới hạn»; SVG hợp lệ (xml parse ok), không có PNG.

**QC tính lại từ dữ liệu thô** (`recompute.py`, không dùng `measure.py`; `recompute.log`): B0 / B1 MAE tổng, MAE tiêu chí, lệch TB khớp `run.py` chênh 0,00000 (cho phép ≤ 0,005); số sửa tay 74 / 62, tốt lên / xấu đi 37 / 2, vốn đúng nay sai 9, nhiễu 5 khớp (tính trên điểm AI trung bình hai lượt như «Giới hạn» của báo cáo). BT-B cách ly: lệch 0,000.
Ghi chú: «Lưu ý (theo trạng thái): {}» vì lớp KC106 trên ep-r vừa nạp, chưa có lưu ý; số lưu ý của lần 14/11 ở `kc-2026-11-14/guidance.md`.

## TC-G10-12 (AC6): P
- Dòng kết luận: **Giữ tắt**. QC áp ngưỡng lên số của mình: (1) MAE tổng giảm 33,0 % ≥ 20 %: đạt; (2) |lệch B1| 1,156 ≤ |lệch B0| 1,856 + 0,20 = 2,056: đạt; (3) vốn đúng nay sai 9 > 1,5 × nhiễu 5 = 7,5: **không đạt** ⇒ Giữ tắt. Kết luận của `run.py` khớp.
- Ngưỡng không nới: dòng ngưỡng của `run.py` (0,20 / 1,5 / 0,20) chỉ xuất hiện ở commit 744ad2df, `git log -L158,159` không có lần sửa nào sau; `git diff origin/main -- benchmarks/guidance/run.py benchmarks/guidance/kc/measure.py` = 0 dòng.
- Đối chứng hai chiều (cùng drafts, nhãn giaovien 14/11, `out-contrast/`): **Bật được** (MAE giảm 42,4 %; vốn đúng nay sai 8 ≤ 1,5 × 8) ⇒ hàm kết luận không cố định một phía.
- PoC 49 % ghi tham chiếu, không cam kết: US-GD-10 AC6 («PoC (mô phỏng) giảm 49 % chỉ là tham chiếu, không là cam kết») và QUESTIONS Q-G-14; báo cáo của `run.py` không in con số 49 %.

## TC-G10-13 (AC7): P
PM giao qc2 làm agent «Giáo viên»; ghi theo TC:
- **Nguồn nhãn: agent** (qc2, Claude), chấm bằng rubric của bài; **kết quả này không dùng làm nhãn người thật**. Đã có sẵn một lượt agent khác (`giaovien`, 14/11) trong repo; hai bộ nhãn độc lập nhau, riêng 2 bài đầu (kc021, kc022) có ngoại lệ ở bảng trên.
- Lưu ý của BT-A KC106 (`kc-2026-11-14/guidance.md`) và quyết định của tôi theo số đo trên nhãn của tôi (lệch TB AI − GV theo tiêu chí c1..c4: B0 +0,450 / +0,766 / +0,534 / +0,106 → B1 +0,247 / +0,728 / +0,134 / +0,047):
  | Lưu ý | Quyết định | Tiêu chí của tôi |
  | --- | --- | --- |
  | c3 STRICTER «giải pháp vượt nguồn lực, ngân sách hoặc thiếu phân kỳ ưu tiên cần bị hạ điểm đáng kể» | **Duyệt** | Tôi cho 0–0,75 mọi bài đòi SIEM / SOC / EDR / DLP đồng loạt hoặc «chỉ cần mua thiết bị là an toàn tuyệt đối»; AI cho cao. B1 giảm lệch c3 từ +0,534 xuống +0,134 |
  | c1 STRICTER «mở bài chỉ nêu bối cảnh chung … tối đa 1,5/2,5; nêu vấn đề và phạm vi nhưng thiếu kết luận khoảng 2,0» (giaovien đã sửa chữ) | **Duyệt** bản đã sửa | Tôi cho 2,25–2,5 chỉ khi luận điểm hoặc kết luận chính nằm ở đoạn đầu; mở bài chung chung 1,0–1,5. Lệch c1 giảm +0,450 → +0,247 |
  | c2 (bộ lọc từ khoá loại, P-106-12) | Không có gì để duyệt hoặc bỏ | Lệch c2 gần như không đổi (+0,766 → +0,728): AI vẫn chấm c2 cao cho bài chung chung không số liệu; nếu có lưu ý c2 STRICTER «tình huống không có số liệu / quy mô cụ thể ≤ 1,0» tôi sẽ duyệt |
- **Nhãn S7**: `expected_grading.json` ca S7: teacher_label = 8,00 (c1..c4 = 2,00 / 2,00 / 2,00 / 2,00; giảng viên sửa c2 từ 1,00 lên 2,00); `expected` 7,00 là nháp AI lượt 1 dùng cho test code, không phải nhãn người; tôi không sửa tệp này.
- PM tắt agent sau kiểm chứng: việc của PM; qc2 không còn chạy gì và không còn quyền nào ngoài ep-r.
