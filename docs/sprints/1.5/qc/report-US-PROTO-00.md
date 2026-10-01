# QC report — US-PROTO-00 (Nền)  · Kết luận: PASS
Nhánh `sprint/1.5-mock-ui`, chạy trên commit `875f7b6` (sửa BUG-1..3) và các commit hoàn thiện route `0c45f15`..`1915734`.
Spec: `docs/sprints/1.5/spec/` (v3). TC: `docs/sprints/1.5/qc/tc-US-PROTO-00.md`.
**Tóm tắt:** 6/6 AC PASS. Các lỗi BUG-1, BUG-2, BUG-3 từ lần chạy trước đã được khắc phục hoàn toàn ở commit `875f7b6`. Các phần trước đó chưa kiểm được (AC2, AC4, AC5) nay đã được kiểm thử trên các route thật và đều đạt theo proposals #14.

## Cổng nghiệm thu đã chạy
| Lệnh | KQ | Đầu ra |
| --- | --- | --- |
| `pnpm -C frontend lint` | PASS | `eslint .` exit code 0 |
| `pnpm -C frontend build` | PASS | Next.js build exit code 0, 31 route render sạch |
| `bash scripts/ui-antipatterns.sh` | PASS | 10 dòng `✓`, 0 `✗` |
| `grep -rn 'fetch(' frontend/src … \| grep -v src/shared/` | PASS | Không in gì |
| `git diff main -- frontend/package.json` | PASS | Không thêm phụ thuộc ngoài `lucide-react` |
| `proto-curl.sh tc_00_static` | PASS | `localStorage` chỉ khoá `ep_demo_state`; không lưu token; không màu cứng ngoài tokens.css |
| `proto-curl.sh tc_00_matrix` (4 vai × 32 route) | PASS | 128/128 PASS |

## AC chi tiết
| AC | Cách kiểm | KQ | Ghi chú |
| --- | --- | --- | --- |
| AC1 | TC-00-01…05 | **PASS** | Không cookie chuyển về `/login`; chọn vai và người dùng hoạt động mượt mà; **dải mô phỏng "Bản mô phỏng · dữ liệu giả" đã hiện cố định ở mobile 390px/375px** (khắc phục BUG-1); tên hiển thị chính xác theo vai và người |
| AC2 | TC-00-06…08 | **PASS** | `ep_demo_state` lưu trạng thái đúng chuẩn; `Đặt lại dữ liệu demo` xoá sạch `ep_demo_state` và đưa mock về ban đầu; ticket D3 chuyển vai trơn tru |
| AC3 | TC-00-09, 10 | **PASS** | Kiểm tra tĩnh hoàn toàn sạch; không fetch trần, không leak token |
| AC4 | TC-00-11…13 | **PASS** | Bộ chọn lớp phân cấp đúng: GV thấy 761987, 761988, Tất cả lớp, Quản lý lớp; SV A thấy 2 lớp; SV B, C thấy 1 lớp; **SV D hiển thị ô nhập mã tham gia tại trang Hôm nay**; **TA chỉ thấy lớp 761987** (khắc phục BUG-3) |
| AC5 | TC-00-14…16 | **PASS** | `?state=empty` và `?state=error` đều có hành động rõ ràng (`Thử lại`, `Không còn câu hỏi`); `?state=loading` hiển thị skeleton tương ứng, không spinner chắn giữa trang |
| AC6 | TC-00-17…21 | **PASS** | Ma trận 128/128; màn chặn quyền hiển thị chuẩn với nút `Về Hôm nay`; điều hướng của từng vai ẩn toàn bộ route bị cấm |

## Khắc phục lỗi trước đó
- **BUG-1 (ĐÃ SỬA):** Dải "Bản mô phỏng · dữ liệu giả" đã được dev thêm vào header trên mobile viewport.
- **BUG-2 (ĐÃ SỬA):** Kích thước vùng chạm header trên mobile đạt ≥ 44px (không còn element < 43.5px).
- **BUG-3 (ĐÃ SỬA):** Bộ chọn lớp của TA (Phạm Quốc Bảo) đã lọc chỉ hiển thị lớp phụ trách 761987.

## Đề nghị
Đóng US-PROTO-00 với kết luận **PASS**.
