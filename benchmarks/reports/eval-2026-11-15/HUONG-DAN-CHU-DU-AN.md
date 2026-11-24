# Bốn việc gán nhãn của chủ dự án (E1, E2, E3, E5)

Không việc nào gọi LLM. Ba việc đầu chỉ cần một trình sửa bảng tính (Numbers, Excel, LibreOffice). E5 cần stack `ep-qc` đang chạy.

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
| E5 duyệt câu AI sinh | `http://localhost:580` → Ngân hàng câu hỏi | 100 | 1,5–2 giờ |

Làm theo thứ tự nào cũng được, dừng giữa chừng được: lệnh tính lại chỉ đếm dòng đã điền. Khi lưu bản chép: chọn **CSV UTF-8, phân cách bằng dấu phẩy** (Excel tiếng Việt có thể mặc định dấu chấm phẩy — đổi lại), giữ nguyên dòng tiêu đề và thứ tự cột.

## E1 — gán nhãn PII (P-11-3)

Tệp: `benchmarks/reports/eval-2026-11-15/e1-soat-nhan.csv`. 57 câu đăng Threads (32 có dữ liệu cá nhân, 25 không), xáo thứ tự, **không hiện nhãn máy** để bạn gán mù.

1. Chép tệp sang `benchmarks/eval/human/e1_soat.csv` rồi mở bản chép.
2. Mỗi dòng, đọc `van_ban` và điền cột `nhan_chu_du_an`:
   - `CHẶN`: câu lộ dữ liệu cá nhân của một người cụ thể (tên, MSSV, email, số điện thoại, CCCD…), không được đăng công khai.
   - `CHO QUA`: câu hỏi học thuật, không lộ ai.
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

Cần stack `ep-qc` đang chạy. Câu và kết quả duyệt nằm trong volume Postgres của `ep-qc`: dừng / khởi động lại stack không mất; chỉ `docker compose down -v` mới mất (bị cấm).

1. Mở `http://localhost:580`, đăng nhập `teacher@edupilot.local` (mật khẩu = `SEED_DEFAULT_PASSWORD` trong `~/Documents/ep-qc-env/env.sh`), chọn lớp **761987** → **Ngân hàng câu hỏi**.
2. Lọc **Nguồn = Sinh từ tài liệu**, **Trạng thái = Chờ duyệt**, rồi lần lượt **Chủ đề** = một trong 10 chủ đề của bảng «Từng lô» ở `e5.md` (lớp còn câu chờ duyệt khác do QC tạo, không thuộc E5).
3. Mở từng câu:
   - đúng như AI viết → `Duyệt` (tính «duyệt nguyên»);
   - phải sửa → `Chỉnh sửa`, lưu, rồi `Duyệt` (tính «sửa nhẹ»);
   - sai, lạc đề, trùng → menu `⋯` → `Loại`.

Xong (hoặc giữa chừng): `python3 benchmarks/eval/e5.py report` → `e5.md` hiện số đã duyệt và tỉ lệ (chỉ tính trên câu đã duyệt).

## Khi xong cả bốn

```bash
make eval-report      # tính lại e1…e6 từ raw/ + bản đã điền; không gọi LLM (E5 đọc gen-stats từ ep-qc)
```

Rồi commit `benchmarks/eval/human/*.csv` và `benchmarks/reports/eval-2026-11-15/` (hoặc báo PM để đội commit). Báo cáo E1 / E2 đổi từ `WAITING_HUMAN` sang `SOÁT (… đồng ý; bất đồng → BA phán: …)`, E3 / E5 đổi sang `OK` khi đủ mẫu.
