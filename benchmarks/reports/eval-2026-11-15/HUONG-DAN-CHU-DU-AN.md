# Bốn việc gán nhãn của chủ dự án (E1, E2, E3, E5)

Không việc nào gọi LLM. Cả bốn việc chỉ cần một trình sửa bảng tính (Numbers, Excel, LibreOffice). Người gán từ 2026-11-25: vai `giaovien` do chủ dự án chỉ định.

Lệnh chạy từ thư mục repo:

```bash
. ~/Documents/ep-qc-env/env.sh        # API_URL, mật khẩu seed (SEED_DEFAULT_PASSWORD) của ep-qc
mkdir -p benchmarks/eval/human        # nơi lưu bản đã điền (E1, E2, E3)
python3 benchmarks/eval/human.py status   # tiến độ E1 / E2 / E3 bất cứ lúc nào
```

| Việc | Tệp / URL | Số mục | Ước lượng |
| --- | --- | --- | --- |
| E1 gán nhãn PII | `e1-soat-nhan.csv` | 57 | 15–20 phút |
| E2 soát nhãn câu hỏi | `e2-soat-nhan.csv` | 24 | 30–45 phút |
| E3 chấm tay tự luận | `e3-cham-tay.csv` | 60 | 2–3 giờ |
| E5 duyệt câu AI sinh | `benchmarks/eval/human/e5_duyet_goi.csv` | 100 | 1,5–2 giờ |

Làm theo thứ tự nào cũng được, dừng giữa chừng được: lệnh tính lại chỉ đếm dòng đã điền. Khi lưu bản chép: chọn **CSV UTF-8, phân cách bằng dấu phẩy** (Excel tiếng Việt có thể mặc định dấu chấm phẩy — đổi lại), giữ nguyên dòng tiêu đề và thứ tự cột.

## E1 — gán nhãn PII (P-11-3)

Tệp: `benchmarks/reports/eval-2026-11-15/e1-soat-nhan.csv`. 57 câu đăng Threads (32 có dữ liệu cá nhân, 25 không), xáo thứ tự, **không hiện nhãn máy** để bạn gán mù.

1. Chép tệp sang `benchmarks/eval/human/e1_soat.csv` rồi mở bản chép.
2. Mỗi dòng, đọc `van_ban` và điền cột `nhan_chu_du_an`:
   - `CHẶN`: câu lộ dữ liệu cá nhân của một người cụ thể (tên, MSSV, email, số điện thoại, CCCD…), không được đăng công khai; **hoặc hỏi điểm, chuyên cần, thi lại, phúc khảo của chính mình** (Q4, PRD M1: câu hỏi cá nhân không bao giờ lưu công khai, chuyển sang chat riêng).
   - `CHO QUA`: câu hỏi học thuật, không lộ ai, không hỏi dữ liệu học tập của chính mình.

   *Vế Q4 thêm 2026-11-25, SAU lượt gán của `giaovien` (853cb185): lượt đó gán theo định nghĩa cũ (chỉ vế đầu), nên 4/4 mẫu nhóm S6 bị gán `CHO QUA`; BA phán `CHẶN` (7c717d0b). PM quyết không gán lại 4 mẫu đó (đã thấy bất đồng = mất tính mù).*
3. Phân vân thì ghi vào `ghi_chu`. Không đổi cột `ma`.

Xong: `python3 benchmarks/eval/human.py status` → dòng E1 hiện số đồng ý và mã mẫu bất đồng (BA phán). Đủ khi ≥ 50 mẫu (≥ 25 có, ≥ 25 không có dữ liệu cá nhân).

## E2 — soát 24 nhãn câu hỏi (P-11-4)

Tệp: `benchmarks/reports/eval-2026-11-15/e2-soat-nhan.csv`. 24 mục E2-005, E2-010 … E2-120 (cách 5 mục một). Tài liệu nguồn ở `seed/documents/` (cột `nguon`, `trang`).

1. Chép tệp sang `benchmarks/eval/human/e2_soat.csv` rồi mở bản chép.
2. Mỗi dòng, đọc `cau_hoi`, `tra_loi_duoc`, `phai_chua`, `bang_chung`; mở PDF ở trang ghi trong `trang` nếu cần. Điền `ket_luan`:
   - `Đúng`: tài liệu trả lời được đúng như cột `tra_loi_duoc`, và câu trả lời đúng phải chứa ít nhất một biến thể của mỗi nhóm trong `phai_chua` (các nhóm ngăn bởi `|`, biến thể trong nhóm ngăn bởi `/`).
   - `Sai nhãn`: `tra_loi_duoc` sai, hoặc `phai_chua` thiếu / sai ý.
   - `Câu hỏi mơ hồ`: câu hỏi có thể hiểu nhiều cách.
3. Lý do ghi vào `ghi_chu`.

Xong: `python3 benchmarks/eval/human.py status` → dòng E2. Đủ khi cả 24 mục có kết luận.

## E3 — chấm tay 60 bài tự luận (Q-E-18)

Tệp: `benchmarks/reports/eval-2026-11-15/e3-cham-tay.csv`. 60 bài BT-A của KC106, thứ tự xáo, **không có điểm AI** (tránh neo). Bài ở cột `tep_bai` (`benchmarks/guidance/kc/data/bt-a/KC106xxx.txt`); rubric 4 tiêu chí, mỗi tiêu chí 0–2,5 (`benchmarks/guidance/kc/data/rubric.json`).

1. Chép tệp sang `benchmarks/eval/human/e3_scores.csv` rồi mở bản chép.
2. Mỗi bài, đọc tệp bài, điền 4 cột `c1`…`c4` (dấu phẩy hay dấu chấm thập phân đều được, ví dụ `2,5` hoặc `1.5`). Không đổi cột `ma_bai`.
3. Bài nào chưa chấm thì để trống cả 4 ô; dòng thiếu ô bị bỏ qua.

Xong: `python3 benchmarks/eval/human.py status` → «E3 đã chấm n/60 bài». Đủ 60 bài thì báo cáo E3 tính QWK / MAE / Pearson người – AI.

## E5 — duyệt 100 câu trắc nghiệm AI sinh (AC7)

Tệp: `benchmarks/eval/human/e5_duyet_goi.csv`. 100 câu (10 lô × 10, MCQ một đáp án) theo thứ tự lô; mỗi dòng có đề, phương án `A`…`D`, `dap_an_danh_dau` (đáp án AI đánh dấu đúng), `giai_thich`, `nguon` (tài liệu, trang) và `trich_doan` (đoạn tài liệu AI trích). **Không có điểm hay nhận định nào của máy.** Tài liệu gốc ở `seed/documents/`.

1. Chép tệp sang `benchmarks/eval/human/e5_duyet.csv` rồi mở bản chép.
2. Mỗi dòng, điền `quyet_dinh`:
   - `DUYỆT`: dùng được nguyên như AI viết (đề rõ, đúng một đáp án đúng và đúng là đáp án đánh dấu, khớp tài liệu);
   - `SỬA`: dùng được sau khi sửa nhẹ: ghi bản sửa vào cột `sua_de` / `sua_A`…`sua_D` / `sua_dap_an` (chữ cái, ví dụ `B`) / `sua_giai_thich`; ô để trống = giữ nguyên;
   - `BỎ`: sai, lạc đề, trùng, không cứu được bằng sửa nhẹ; ghi `ly_do_bo` (một trong `MO_HO`, `SAI_DAP_AN`, `TRUNG_LAP`, `LAC_DE`, `KHO_DE_BAT_THUONG`, `SAI_DINH_DANG`; trống = không nêu).
3. Lý do / phân vân ghi vào `ghi_chu`. Không đổi cột `ma`.

Xong: commit + push `e5_duyet.csv`. Tech Lead áp quyết định qua API của stack `ep-eval` trên VPS (DB `ep-qc` 2026-11-25 + gateway 708dc729, bản ep-qc sinh lô 09–10 của E5; lô 01–08 sinh ở 31c19e0d, image đã xoá) bằng `e5.py apply` (vai giảng viên: DUYỆT → duyệt; SỬA → lưu bản sửa rồi duyệt, tính «sửa nhẹ»; BỎ → loại), rồi `e5.py report` đọc gen-stats.

## Khi xong cả bốn

```bash
make eval-report      # tính lại e1…e6 từ raw/ + bản đã điền; không gọi LLM (E5 đọc gen-stats từ ep-eval khi stack còn chạy)
```

Rồi commit `benchmarks/eval/human/*.csv` và `benchmarks/reports/eval-2026-11-15/` (hoặc báo PM để đội commit). Báo cáo E1 / E2 đổi từ `WAITING_HUMAN` sang `SOÁT (… đồng ý; bất đồng → BA phán: …)`, E3 / E5 đổi sang `OK` khi đủ mẫu.
