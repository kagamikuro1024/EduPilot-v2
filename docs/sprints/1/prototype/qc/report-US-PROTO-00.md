# QC report — US-PROTO-00 (Nền)  · Kết luận: FAIL
Nhánh `sprint/1-mock-ui`, chạy trên HEAD `3d90daa` (commit dev mới hơn handoff `e5d2f9f`), `next start -p 3100` từ `next build`. Spec: `docs/sprints/1/prototype/spec/` (v2). TC: `qc/tc-US-PROTO-00.md`. 2026-10-01.
**Tóm tắt:** AC3, AC6 PASS. AC1 FAIL (1 lỗi thật: dải mô phỏng không có trên điện thoại). AC2, AC4, AC5 có phần nền đạt nhưng **phần còn lại KHÔNG KIỂM ĐƯỢC** vì route còn là trang tạm — QC.md tính là FAIL; đề xuất chấm lại sau story 01–04 (proposals #14, chờ PM). 3 lỗi (1 trung bình, 2 thấp) ghi bên dưới.

## Cổng nghiệm thu đã chạy
| Lệnh | KQ | Đầu ra |
| --- | --- | --- |
| `pnpm -C frontend lint` | PASS | `eslint .` rc=0 |
| `pnpm -C frontend build` | PASS | rc=0, 31 route (ƒ Dynamic) |
| `bash scripts/ui-antipatterns.sh` | PASS | 10 dòng `✓`, 0 `✗` |
| `grep -rn 'fetch(' frontend/src … \| grep -v src/shared/` | PASS | không in gì |
| `git diff main -- frontend/package.json` | PASS | không thêm phụ thuộc ngoài `lucide-react` |
| `proto-curl.sh tc_00_static` | PASS | `localStorage` chỉ khoá `ep_demo_state`; không token ở storage; không màu cứng ngoài `tokens.css` |
| `proto-curl.sh tc_00_matrix` (4 vai × 32 route) | PASS | 128 PASS, 0 FAIL |

## AC
| AC | Cách kiểm | KQ | Ghi chú |
| --- | --- | --- | --- |
| AC1 | TC-00-01…05 | **FAIL** | PASS: không cookie → `/login` (cả `/`, `/inbox`, `/chat`, `/gradebook`, `/me`); `/login` có 4 vai, Sinh viên → A / B / C / D (tên, MSSV, mô tả); chọn B → `/`, menu hồ sơ "Trần Thu Uyên"; cookie `ep_demo_role|person|course`; dải "Bản mô phỏng · dữ liệu giả" có ở chân thanh bên desktop (4 vai). **FAIL: dải không có ở 390/375 px** (BUG-1) và `/login` không có dải đúng chữ |
| AC2 | TC-00-06…08 | **KHÔNG KIỂM ĐƯỢC (tính FAIL)** | Phần cơ chế đạt: `Đặt lại dữ liệu demo` xoá `ep_demo_state` (đặt thử `{"x":1}` → `localStorage` rỗng); không khoá nào khác, không giống token. Không kiểm được "ticket D3 sang vai GV": `/chat`, `/inbox` còn trang tạm (xem proposals #14) |
| AC3 | TC-00-09, 10 | PASS | Bảng trên |
| AC4 | TC-00-11…13 | **KHÔNG KIỂM ĐƯỢC một phần (tính FAIL)** | Bộ chọn lớp đạt: A 2 lớp (761987, 761988); B, C 1 lớp (761987); D "Chưa có lớp" không có menu; GV: 761987, 761988, "Tất cả lớp của tôi", "Quản lý lớp này" (+ "Mời trợ giảng"). Không đạt/chưa kiểm: `visible student / sv-4 \| grep -c 'mã tham gia'` = 0 (nội dung `/` SV còn trang tạm). BUG-3: TA thấy cả 761988 |
| AC5 | TC-00-14…16 | **KHÔNG KIỂM ĐƯỢC (tính FAIL)** | 15 route thử (`/inbox` empty+error, `/students`, `/gradebook`, `/documents`, `/insights`, `/analytics`, `/questions`, `/grading`, SV `/threads`, `/library`, `/calendar`, `/me`, `/practice/history`): đều 0 `Thử lại` — route trang tạm chưa dùng `PageState`. Lệnh AC5 chưa thể xanh |
| AC6 | TC-00-17…21 | PASS | Mọi lệnh `open_as` của US đúng; ma trận SRS 2 đủ 128/128; màn chặn: `h1` "Bạn không có quyền mở trang này", câu "Trang này dành cho giảng viên và trợ giảng.", nút `Về Hôm nay`, không lộ nội dung; điều hướng mỗi vai chỉ hiện route được phép (SV 7 mục; TA 12; GV 15 gồm `/observability`, `/settings/*`; Admin 6) |

## Lỗi
- **BUG-1 (trung bình):** dải "Bản mô phỏng · dữ liệu giả" **không hiện trên điện thoại**. Tái hiện: cookie SV B; viewport 390 × 844 (và 375 × 812); mở `/`; `document.body.innerText` không chứa "Bản mô phỏng"; mở drawer "Thêm" cũng không. Kỳ vọng (FR-X6, AC1): "cố định, kín đáo … đầu trang trên điện thoại". Ảnh `qc/shots/00-mobile-student-390.png`. Ở `/login` chỉ có câu "Đây là bản mô phỏng: mọi dữ liệu đều là giả", không phải dải.
- **BUG-2 (thấp, báo trước cho US-PROTO-01 AC6):** vùng chạm của thanh trên ở 390/375 px < 44 px: chip lớp 82×36, nút tìm/chuông 36×34, avatar 62×40, logo 24×24. AC6 của US-PROTO-01 đòi ≥ 44 px ở mọi route SV; shell thuộc US-PROTO-00. (Link "Bỏ qua điều hướng" đo 159×40 nhưng là skip-link ẩn.)
- **BUG-3 (thấp):** TA (Phạm Quốc Bảo) thấy bộ chọn lớp có cả 761988 và "Tất cả lớp của tôi". Kỳ vọng: SRS 4.1 "TA — lớp 1" (chỉ 761987). Tái hiện: cookie `ep_demo_role=ta`; mở chip lớp ở đầu trang.

## Kiểm phản mẫu UI / quyền
- Ảnh: `qc/shots/00-class-chooser-teacher-1440.png`, `00-class-chooser-A-1440.png`, `00-blocked-student-inbox-1440.png`, `00-mobile-student-390.png`. Nhìn bằng mắt: thanh bên có nhãn chữ, 5 nhóm, đỏ chỉ ở mục hiện tại và số đếm; menu lớp 1 danh sách kẻ, không card lồng; bottom nav 5 mục (Hôm nay, Chat riêng, Threads, Luyện đề, Thêm) không cuộn ngang. Chip lớp bị cắt "761987 · An ninh m…" ở 1440 px (nhỏ, chưa tính BUG).
- Phân quyền: đạt toàn bộ ma trận bằng URL gõ thẳng + cookie; cookie vai rác (`ep_demo_role=hacker`) không gây 5xx (TC-00-01).
- Không `fetch`, không phụ thuộc mới, không token: PASS.

## Đề nghị
1. Dev sửa BUG-1 (và BUG-2 trước khi chấm US-PROTO-01 AC6), BUG-3 nếu PM đồng ý.
2. PM quyết proposals #14: chấm lại AC2 / AC4 / AC5 của US-PROTO-00 theo story sở hữu route (US-PROTO-01…04) thay vì buộc nền tự đạt.
3. QC chạy lại TC-00 khi dev báo sửa; TC story 01–04 chạy khi có handoff.
