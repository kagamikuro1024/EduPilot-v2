# QC test case — US-PROTO-00 (Nền: vào bằng một vai, đổi vai)
Nguồn: `docs/sprints/1/prototype/spec/US.md` + `SRS.md` (v2 APPROVED; spec dời vào đây theo PM). Viết hộp đen, không đọc code dev.
Chạy: dev server `pnpm dev` hoặc `pnpm -C frontend exec next dev -p 3000` → `F=http://localhost:3000`. Mỗi shell `source ~/.zprofile`.
Công cụ: **C** = `F=… bash docs/sprints/1/prototype/qc/scripts/proto-curl.sh <TC>`; **B** = trình duyệt thật qua `sweep.mjs` / thao tác tay trong Eval (`browser`), ảnh vào `qc/shots/`.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-00-01 | AC1 | Không cookie | **C** `tc_00_01`: `curl -s -o /dev/null -w '%{redirect_url}' $F/` cho `/`, `/inbox`, `/chat`, `/gradebook`, `/me` | Mọi route chuyển về `/login` |
| TC-00-02 | AC1 | – | **B** Mở `/login` (xoá cookie + localStorage) | Có 4 vai: Sinh viên, Trợ giảng, Giảng viên, Admin; chọn Sinh viên → chọn tiếp A / B / C / D (tên + mô tả một dòng) |
| TC-00-03 | AC1 | – | **B** Chọn Sinh viên B | Vào `/`; menu hồ sơ ghi "Trần Thu Uyên"; dải "Bản mô phỏng · dữ liệu giả" hiện, chữ phụ, **không đỏ**; kiểm lại ở 4 vai và ở 390 px (đầu trang) |
| TC-00-04 | AC1 | – | **C** `tc_00_01` (phần `visible … grep -c 'Bản mô phỏng'`) | ≥ 1 ở `/` của cả 4 vai |
| TC-00-05 | AC1 / FR-X1 | – | **B** Sau khi chọn vai, `document.cookie` | Có `ep_demo_role`, `ep_demo_person`, `ep_demo_course`; không `httpOnly` yêu cầu (ngoài phạm vi) |
| TC-00-06 | AC2 / FR-X2, X3 | B đã gửi D3 ở `/chat` (làm sau US-PROTO-01) | **B** Menu hồ sơ → `Đổi vai` → Giảng viên → `/inbox` | Có ticket D3; không phải tải lại tay; `Đổi vai` mở lại lựa chọn ngay tại chỗ |
| TC-00-07 | AC2 | Sau TC-00-06 | **B** `Đặt lại dữ liệu demo` | Ticket D3 biến mất; điểm, điểm danh, công thức… về dữ liệu gốc SRS 4.1 |
| TC-00-08 | AC2 / FR-X3 | – | **B** DevTools → Application → Local Storage; cookies | Chỉ một khoá `ep_demo_state`; không giá trị giống token/JWT (`eyJ`, `Bearer`, `token`); sau `Đặt lại` khoá bị xoá |
| TC-00-09 | AC3 / FR-X7, X10 | Cây sạch | `pnpm -C frontend lint && pnpm -C frontend build && bash scripts/ui-antipatterns.sh` | Cả ba exit 0; 0 `✗` |
| TC-00-10 | AC3 | – | **C** `tc_00_static` | Không `fetch(` ngoài `src/shared/`; `package.json` không thêm phụ thuộc ngoài `lucide-react`; `localStorage` chỉ khoá `ep_demo_state`; không token ở storage; không màu cứng ngoài `tokens.css` |
| TC-00-11 | AC4 / FR-X8 | – | **B** GV mở bộ chọn lớp | Có 761987, 761988, "Tất cả lớp của tôi", "Quản lý lớp này"; đổi lớp → mọi màn đổi dữ liệu theo lớp |
| TC-00-12 | AC4 | – | **B** Sinh viên A / B / C / D mở bộ chọn lớp | A thấy 2 lớp; B, C 1 lớp (761987); D không có lớp |
| TC-00-13 | AC4 | – | **C** `tc_00_04` | `visible student / sv-4 \| grep -c 'mã tham gia'` ≥ 1; "Hôm nay" của D là ô nhập mã |
| TC-00-14 | AC5 (nhánh lỗi) | – | **C** `tc_00_05` (`?state=empty|error` trên `/inbox`, `/students`, `/gradebook`, `/documents`, `/insights`, `/analytics`, `/questions`, `/grading`, SV: `/threads`, `/library`, `/calendar`, `/me`, `/practice/history`) | `error` có `Thử lại`; `/inbox?state=empty` có "Không còn câu hỏi" |
| TC-00-15 | AC5 | – | **B** `?state=loading` trên 3 route mỗi nhóm (SV: `/threads`, `/me`, `/library`; GV: `/inbox`, `/students`, `/gradebook`; Admin: `/admin/users`, `/observability`, `/`) | Skeleton đúng hình vùng nội dung; **không spinner giữa trang** (DESIGN §21) |
| TC-00-16 | AC5 | – | **B** `?state=error` → bấm `Thử lại` | Bỏ tham số, về trạng thái thường; câu lỗi nêu vấn đề + cách khắc phục (DESIGN §15), không từ kỹ thuật với SV |
| TC-00-17 | AC6 (phân quyền) | – | **C** `tc_00_06` | Các lệnh `open_as` của US: GV bị chặn `/chat /me /practice /library /join`; SV bị chặn `/inbox /attendance /gradebook /grading /admin/courses /observability`; Admin bị chặn `/chat /inbox /gradebook /calendar`; TA mở `/class/members`, bị chặn `/admin/users` |
| TC-00-18 | AC6 | – | **C** `tc_00_06` (nội dung màn chặn) | Có "Bạn không có quyền mở trang này" (FR-X4), một câu "Trang này dành cho …", nút `Về Hôm nay`; không lộ nội dung trang |
| TC-00-19 | AC6 / FR-X4 | – | **C** `tc_00_matrix` (toàn bộ ma trận SRS mục 2, 4 vai × 31 route) | Mọi ô đúng MO/CHAN; R (TA `/gradebook`, GV `/observability`) mở được |
| TC-00-20 | AC6 | – | **B** Điều hướng (sidebar / bottom nav / ⌘K) của từng vai | Không hiện route bị chặn của vai đó (kể cả kết quả ⌘K) |
| TC-00-21 | AC6 | – | **B** Đang ở `/attendance` (GV) → `Đổi vai` sang Sinh viên | Không để lại nội dung `/attendance` cho SV: về `/` hoặc màn chặn |

## Nhánh lỗi
TC-00-14, 15, 16 (`?state=`); TC-00-01 (không cookie); thêm: cookie vai rác `ep_demo_role=hacker` → không 5xx (`tc_00_01`).

## Phân quyền
TC-00-17…21 (ma trận SRS mục 2, gõ thẳng URL bằng cookie vai khác).

## Kiểm chéo
- Phản mẫu UI: `ui-antipatterns.sh` (TC-00-09) + xem ảnh 1440 / 390 theo DESIGN §21, §22 (sweep `shell`, `/login`, màn chặn).
- 375/390 px: shell + `/login` + màn chặn không cuộn ngang; bottom nav ≤ 5 đích.
- Không PII/secret: storage chỉ `ep_demo_state` (TC-00-08).
- Không gọi mạng ra ngoài trừ font: DevTools Network không có host lạ (B).

## Điểm khó kiểm
- `curl` chỉ thấy HTML phía server; nội dung chỉ sinh ở client (localStorage) phải kiểm bằng trình duyệt (B).
- AC1 cho phép "HTML trả về là màn chọn vai" thay vì redirect: `tc_00_01` chấp nhận redirect; nếu không redirect, so tay.

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
(chưa có)
