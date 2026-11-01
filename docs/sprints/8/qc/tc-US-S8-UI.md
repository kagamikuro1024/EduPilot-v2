# QC test case — US-S8-UI (bỏ câu mô tả thừa ở `/settings/llm`, `/settings/integrations`, khối bảo mật `/settings`)
Nguồn: `docs/specs/FEAT-gradebook/US.md` US-S8-UI (AC1–AC3, cuối tệp); luật cứng chữ UI (`~/.claude/CLAUDE.md`, 09/10/2026); `docs/sprints/8/handoff/dev-US-S8-UI.md` (**chỉ đọc ở pha 2**; TC này viết hộp đen từ US + luật). Phạm vi sản phẩm: không cần spec riêng.

**Luật dùng để chấm** (nguyên văn tinh thần): không câu mô tả dưới tiêu đề khối, dưới từng dòng, hay lặp một câu cảnh báo trên mọi dòng; câu thừa **xoá**, không dời vào tooltip; không rải icon Info/CircleHelp; tooltip chỉ cho thứ thiếu nó người dùng làm sai (cú pháp nhập, lý do nút khoá, hệ quả lựa chọn), đặt trên chính nhãn/nút; mỗi dòng: nhãn + điều khiển + giá trị; trạng thái 1–2 từ; ví dụ cú pháp ở `placeholder`. Ngoại lệ hợp lệ: `InlineNotice` báo trạng thái/lỗi/hệ quả có điều kiện, trạng thái rỗng/lỗi.

**Quy ước.** `playwright-cli` ở 1440 × 900 và 390 × 844; ADMIN `admin@edupilot.local` cho `/settings/llm`, `/settings/integrations`; mọi vai có `/settings` (STUDENT, TEACHER, ADMIN) cho khối bảo mật. Mỗi màn: lưu ảnh `shots/ui-<route>-<vp>.png` + dump `main.innerText` ra `shots/ui-<route>.txt`. "Câu mô tả" = đoạn `<p>`/dòng chữ thường (không phải nhãn, không phải giá trị, không phải nút, không phải lỗi) ngay dưới tiêu đề trang/khối hoặc dưới một dòng điều khiển. QC đối chiếu từng đoạn trong dump và liệt kê đoạn nào còn.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-S8-UI-01 | AC1 | ADMIN, stack + seed | Mở `/settings/llm` 1440 px; `page.locator('main').innerText()`; liệt kê mọi phần tử `h1/h2/h3` và phần tử văn bản liền kề kế tiếp | dưới `h1` và dưới tiêu đề **bốn khối** (Kết nối nhà cung cấp, Mô hình theo tác vụ, Chuỗi dự phòng, Mức dùng và ngân sách) **không** có câu mô tả; liền kề tiêu đề là điều khiển/bảng/trạng thái |
| TC-S8-UI-02 | AC1 | như trên | Từng dòng (nhà cung cấp, tác vụ, dự phòng) | mỗi dòng chỉ nhãn + điều khiển + giá trị; trạng thái 1–2 từ ("Đang chạy", "Lỗi"); không câu cảnh báo lặp trên mọi dòng |
| TC-S8-UI-03 | AC1 | như trên | `page.locator('svg[data-lucide="info"], svg[data-lucide="circle-help"], [aria-label*="Thông tin"], [data-icon="info"]').count()` + chụp mắt | `0` icon Info / CircleHelp |
| TC-S8-UI-04 | AC1 | như trên | Rê chuột / focus mọi `[title]`, `[data-tooltip]`, `[role=tooltip]` (liệt kê bằng script) | tooltip chỉ ở nhãn/nút thuộc loại cho phép: cú pháp nhập, lý do nút khoá, hệ quả lựa chọn; tooltip **nào lặp lại câu mô tả đã xoá** (dời vào tooltip) → FAIL; QC liệt kê từng tooltip + lý do hợp lệ |
| TC-S8-UI-05 | AC1 | như trên | Khối "Mô hình tìm kiếm": đổi mô hình | dòng "Đổi mục này cần lập chỉ mục lại tài liệu." (hệ quả của lựa chọn) có thể còn **nếu** là cảnh báo có điều kiện/hệ quả — ghi nhận cách hiển thị (xuất hiện thường trực hay chỉ khi đổi). Thường trực dưới nhãn mà không có điều kiện → đánh dấu "nghi ngờ" trong báo cáo, PM quyết (luật nói "hệ quả của một lựa chọn" thì được tooltip, không phải dòng cố định) |
| TC-S8-UI-06 | AC1 | như trên | 390 px | không câu mô tả; không tràn ngang; nhãn không bị cắt |
| TC-S8-UI-07 | AC2 | ADMIN | Mở `/settings/integrations` 1440 px; dump + liệt kê | dưới `h1` không câu mô tả; mỗi tích hợp (3 dòng): **nhãn + điều khiển + giá trị**; không dòng mô tả từng tích hợp; trạng thái 1–2 từ |
| TC-S8-UI-08 | AC2 | như trên | Ô nhập có cú pháp (URL, token, tên miền): đọc `placeholder` | ví dụ cú pháp nằm ở `placeholder` (không ở đoạn chữ bên dưới) |
| TC-S8-UI-09 | AC2 | như trên | icon Info/CircleHelp, tooltip | như TC-03, TC-04: 0 icon; tooltip hợp lệ |
| TC-S8-UI-10 | AC2 | như trên, 390 px | xem | gọn, không tràn |
| TC-S8-UI-11 | AC3 | STUDENT và TEACHER và ADMIN | Mở `/settings`; tìm khối bảo mật ("Đổi mật khẩu", "Phiên đăng nhập") | không câu mô tả dưới tiêu đề khối ("Đổi mật khẩu và xem nơi tài khoản của bạn đang đăng nhập." hay tương tự); mỗi dòng phiên: thiết bị + thời điểm + nút `Đăng xuất`; mật khẩu: nhãn + ô nhập |
| TC-S8-UI-12 | AC3 | như trên | Quét toàn trang `/settings` (các khối khác: hồ sơ, thông báo, thời gian học…) | khối ngoài phạm vi AC3 **không bị sửa lan**: so ảnh với trước (QC lấy ảnh từ `main` bằng worktree khác hoặc từ `frontend/e2e/visual.spec.ts-snapshots` cũ); thay đổi ngoài ba màn đã nêu → báo là lan phạm vi |
| TC-S8-UI-13 | AC3 | repo | `bash scripts/ui-antipatterns.sh; echo rc=$?` | `rc=0`, sạch |
| TC-S8-UI-14 | AC3 | repo | `pnpm -C frontend exec playwright test settings-llm.spec.ts shell.spec.ts --workers=2`; ghi số test, flaky | xanh toàn bộ (≥ 3 flaky → FAIL) |
| TC-S8-UI-15 | AC3 | repo | `git diff <trước sprint>..HEAD --stat -- 'frontend/e2e/*' '*_test.go'` và đọc diff từng test sửa | test cũ chỉ bị sửa nếu có số proposal PM chấp nhận (**#1**: `lcp.spec.ts` thêm `mockLlmApi(page)` cho ca `/settings/llm`, **không** đổi ngưỡng/assertion); sửa nào khác không có số → **nghiêm trọng** |
| TC-S8-UI-16 | AC3 | repo | `grep -rn` từng chuỗi dev ghi "đã xoá" trong `docs/specs/**`, `frontend/src/**`, `docs/**` (lấy danh sách từ handoff ở pha 2) | không còn chuỗi nào ở SRS/Confluence-nguồn (luật cứng PO: đổi chữ có trong SRS thì vá SRS cùng đợt); chuỗi còn ở ảnh/ghi chú nào → báo |
| TC-S8-UI-17 | AC3 | repo | `pnpm -C frontend lint && pnpm -C frontend build` | xanh |
| TC-S8-UI-18 | ảnh mốc | repo | `playwright test visual.spec.ts` (đúng image `mcr.microsoft.com/playwright:v1.63.0-noble` như dev nêu) cho `settings-llm-1440/390` | các ảnh mốc khớp UI mới; ảnh được cập nhật chỉ ở 2 màn ảnh hưởng (`git diff --stat -- '*snapshots*'`) |
| TC-S8-UI-19 | a11y | `/settings/llm`, `/settings/integrations` | `playwright test a11y.spec.ts` (hoặc `axe` qua `playwright-cli`) | không vi phạm mới; nhãn của điều khiển không phụ thuộc câu mô tả đã xoá (`aria-describedby` trỏ phần tử đã xoá → lỗi) |
| TC-S8-UI-20 | LCP | `/settings/llm` | `playwright test lcp.spec.ts` | xanh; ngưỡng không đổi (proposal #1) |

## Nhánh lỗi
| TC-id | Tình huống | Bước | Kết quả mong đợi |
| --- | --- | --- | --- |
| TC-S8-UI-21 | Trạng thái lỗi/rỗng vẫn hiện | chặn API cấu hình LLM 500 / không có nhà cung cấp | khối lỗi (nguyên nhân + `Thử lại`) và trạng thái rỗng còn nguyên (không bị xoá nhầm cùng câu mô tả) |
| TC-S8-UI-22 | Cảnh báo có điều kiện còn | ADMIN xem `InlineNotice` "Chỉ quản trị viên…" | còn (không phải câu mô tả) hoặc đã được PM đồng ý bỏ — ghi nhận |

## Phân quyền
| TC-id | Tình huống | Bước | Kết quả mong đợi |
| --- | --- | --- | --- |
| TC-S8-UI-23 | `/settings/llm`, `/settings/integrations` với vai khác ADMIN | STUDENT, TEACHER mở | trang 403 chuẩn của shell (hành vi không đổi); `code -H "$TCH" $GW/api/v1/admin/llm/providers` → `403` |

## Script chạy được
`playwright-cli` thủ công (dump `innerText`); `frontend/e2e/{settings-llm,shell,lcp,a11y,visual}.spec.ts` (chỉ chạy, không sửa).

## Câu hỏi cho BA / PM (không tự đoán)
- Q-UI-a (TC-S8-UI-05): dòng "Đổi mục này cần lập chỉ mục lại tài liệu." — luật cho phép hệ quả của lựa chọn ở tooltip hoặc cảnh báo có điều kiện; không cho dòng cố định dưới tiêu đề. QC chấm theo cách hiển thị thật và báo.

## Lịch sử sửa TC
(chưa có)
