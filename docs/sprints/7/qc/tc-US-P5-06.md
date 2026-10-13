# QC test case — US-P5-06 (`/me`: chuyên cần, điểm cộng, thời gian học; không gamify; `/settings` công tắc đo)
Nguồn: `US.md` US-P5-06 AC1…AC10 + `SRS.md` 2, 4.6, 6.2, 6.3, 7.6, 7.8; `DESIGN.md` §14.9 ("Do not gamify with streaks or ranks"); `UX.md` mục 1, 3, 6; FLOWS F7 bước 5, F8; PRD M5, M13-C. Hộp đen, không đọc code dev.
Biến như `tc-US-P5-01.md`. Seed: B (`sv.kha`) vắng 2, 2 phát biểu = +0,50. A ghi danh lớp 1 và lớp 2.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P5-06-01 | AC1 (của mình) | seed | `j "$Q/me/attendance" -H "$SVB"`; `j "$Q/me/participation" -H "$SVB"` | `attendance:{recorded,present,late,excused,absent,sessions:[{session_no,starts_at,status}]}` (mới nhất trước, phân trang con trỏ) và `participation:{speak_count,points_total,events:[{date,type,points}]}`; khớp từng số với DB (`v_attendance_summary`, `participation_events`): B vắng 2, 2 lần phát biểu, +0,50; **không** có `note`, `marked_by`, `version` hay trường nội bộ |
| TC-P5-06-02 | AC1 (không tham số student) | – | thêm `?student=<uidA>` / `?student_id=…` vào URL bằng `$SVB` | bị bỏ qua (trả dữ liệu của B) hoặc `422`; **không** trả dữ liệu của A |
| TC-P5-06-03 | AC1 (rỗng, ngoài lớp) | sinh viên mới không điểm danh; `PENDING`/`REMOVED`/ngoài lớp | gọi | `recorded:0, sessions:[]` (không lỗi); `403` mỗi ca |
| TC-P5-06-04 | AC1 (nhân viên) | – | TA, TEACHER, ADMIN gọi `…/me/attendance`, `…/me/participation` | `403` |
| TC-P5-06-05 | AC1 (phân trang) | lớp 15 buổi | `…/me/attendance?limit=5` đi hết cursor | không trùng, không sót |
| TC-P5-06-06 | AC2 (crawl không canary) | B có ghi chú canary `CANARY-OBS-4F`, `AT_RISK` (hoặc C) | bằng tài khoản sinh viên: duyệt đủ route STUDENT (`me/*`, `chat/*`, `threads`, `me/today`, `notifications`, `calendar`, `library`, SSE `/events` 30 s); lưu mọi phản hồi vào `crawl/`; `grep -rlE 'CANARY-OBS-4F\|"risk\|"observation\|at_risk\|Cần chú ý:\|Đang theo dõi:' crawl/` | 0 tệp; không khoá `risk*`, `observation*`, `at_risk`, `note` (ngoài `note` của chính tin chat) |
| TC-P5-06-07 | AC2 (duyệt UI) | `playwright-cli` đăng nhập C | duyệt `/`, `/me`, `/chat`, `/threads`, `/settings`, `/calendar`; `eval innerText` mỗi trang | không câu nhận định rủi ro, không canary |
| TC-P5-06-08 | AC3 (nội dung `/me`) | `playwright-cli`, `$SVB`, 1280 | mở `/me` | tiêu đề "Kết quả của tôi" + bộ chọn lớp nếu học nhiều lớp; (1) **Chuyên cần**: một câu ("Bạn có mặt 7/9 buổi, vắng 2.") + danh sách buổi vắng/muộn (ngày, buổi); (2) **Điểm cộng đã ghi**: một câu ("Phát biểu 2 lần · +0,50.") + sự kiện gần đây; (3) cuối trang **Thời gian học**: xu hướng nhỏ 8 tuần (`TrendChart` cao ≈ 96 px) + "Tuần này: 45 phút"; số thập phân dùng dấu phẩy |
| TC-P5-06-09 | AC3 (không giả) | – | tìm: điểm thành phần, giải trình điểm, "nếu cuối kỳ được…", việc sắp tới có điểm, ô giả, dữ liệu mô phỏng | **không** hiển thị (không mock) |
| TC-P5-06-10 | AC3 (không gamify) | – | `playwright-cli eval "document.body.innerText + document.body.innerHTML"` \| `grep -ciE 'streak\|rank\|progress ring\|confetti\|xếp hạng\|huy hiệu\|badge\|cố lên\|tuyệt vời'`; tìm `<progress>`, `role=progressbar`, thẻ KPI (lưới ≥ 3 ô số lớn) | `0`; không vòng/thanh tiến độ, không so sánh với bạn học, không câu động viên, không thẻ KPI |
| TC-P5-06-11 | AC4 (khớp AI) | `$SVB` | mở `/me` ghi số; hỏi chat "Em đã nghỉ mấy buổi và được cộng bao nhiêu điểm phát biểu rồi ạ?" | `/me` và chat cho cùng số: 2 buổi vắng; +0,50 |
| TC-P5-06-12 | AC4 (cập nhật ≤ 60 s) | giảng viên (ngữ cảnh khác) ghi thêm 1 phát biểu cho B ở `/attendance` | tải lại/làm mới `/me`; hỏi lại chat | ≤ 60 s: `/me` và chat đều +0,75; không số cũ (dữ liệu điểm danh không cache phía server: `Cache-Control`/ETag trên `…/me/attendance` — kiểm header; `no-store` hoặc dữ liệu luôn mới) |
| TC-P5-06-13 | AC5 (thời gian học của mình) | `$SVB` đã `Đã hiểu` | `j "$Q/me/study-time?weeks=8" -H "$SVB"` | `{weeks:[{week_start,minutes}],this_week_minutes,tracking:{enabled,locked}}`; chỉ của mình; qua guard lớp |
| TC-P5-06-14 | AC5 (rỗng; tắt) | sinh viên mới chưa có dữ liệu; `tracking.enabled=false` | mở `/me` | mảng rỗng → khối **không hiển thị** (không "0 phút" giả); tắt → khối hiện "Đang tắt" thay biểu đồ |
| TC-P5-06-15 | AC5 (không phán xét) | tuần thấp (C: tuần 7–9 < 60 phút) | `/me` của C | không "mục tiêu", không so sánh, không tô đỏ tuần thấp (kiểm màu `getComputedStyle` của cột/điểm thấp = màu thường) |
| TC-P5-06-16 | AC6 (375 px) | `resize 375 812`, `/me` | đo `scrollWidth`, `getBoundingClientRect` | không tràn ngang; một cột; vùng chạm ≥ 44 px; biểu đồ co theo bề rộng; chữ Việt dài không vỡ bố cục |
| TC-P5-06-17 | AC6 (a11y, Lighthouse) | – | `$PW a11y.spec.ts -g me`; `cd frontend && pnpm exec lhci autorun` (route `/me`, mobile) | 0 vi phạm nghiêm trọng; LCP ≤ 2,5 s, JS ≤ 250 KB gzip (ngân sách `UX.md` mục 3) |
| TC-P5-06-18 | AC7 (trạng thái) | – | tải chậm; lớp chưa điểm danh; `page.route` 500; offline sau khi đã tải | khung xương; "Chưa có buổi nào được điểm danh." (không nút); lỗi nói chuyện gì xảy ra + `Thử lại`; offline giữ nội dung cũ + dải mất mạng |
| TC-P5-06-19 | AC8 (phân quyền route) | – | giảng viên/TA/ADMIN mở `/me` | được chuyển về `/` |
| TC-P5-06-20 | AC8 (hai lớp không lẫn) | A ghi danh lớp 1 và 2 | đổi lớp ở bộ chọn; so số với DB từng lớp | số đổi theo lớp; không lẫn |
| TC-P5-06-21 | AC9 (mock) | repo | `grep -rn "@/mock" frontend/src/features/me \| wc -l`; `pnpm -C frontend build && pnpm -C frontend lint` | `0`; xanh |
| TC-P5-06-22 | AC10 (`/settings` công tắc theo lớp) | A ghi danh lớp 1 (cho tắt), lớp 2 (không cho tắt), đã `Đã hiểu` | `/settings` bằng `playwright-cli` | nhóm "Thời gian học" có **một dòng mỗi lớp**: nhãn mã lớp + công tắc "Ghi thời gian học"; lớp 1 đổi được (lạc quan, không toast); lớp 2 **bật và khoá** (`disabled`), tooltip lý do ở chính công tắc, không dòng mô tả |
| TC-P5-06-23 | AC10 (hiệu lực) | – | tắt công tắc lớp 1; `GET /courses/{1}/me/study-time`; `/me` lớp 1; `/me` lớp 2; sai `version` | `tracking.enabled=false`; `/me` lớp 1 "Đang tắt"; `/me` lớp 2 vẫn đủ biểu đồ; sai `version` → `409` hoàn lại + `InlineNotice` |
| TC-P5-06-24 | AC10 (chưa ack / vai) | sinh viên chưa `Đã hiểu`; TA/TEACHER/ADMIN | mở `/settings` | không thấy nhóm "Thời gian học" |
| TC-P5-06-25 | AC10 (375 px, class settings) | `resize 375 812`; `/class/settings` lớp 2 (đã có sinh viên) | – | công tắc cao ≥ 44 px; lớp 2 hiện công tắc "Cho sinh viên tắt đo thời gian học" **khoá**, giá trị "Không" |
| TC-P5-06-26 | UI | – | `bash scripts/ui-antipatterns.sh; echo $?`; `$PW me.spec.ts`; `$PW settings.spec.ts -g 'study time switch per course\|study time locked course'` | `0`; xanh; ghi số test, flaky |

## Nhánh lỗi
TC-P5-06-03, 14, 18, 23 (version). Thêm TC-P5-06-E1: mạng đứt giữa chừng khi mở `/me` → `PageState` lỗi + `Thử lại`; TC-P5-06-E2: sinh viên bị `REMOVED` khỏi lớp trong lúc đang xem → làm mới → `403` hiển thị trạng thái lỗi thân thiện (không lộ dữ liệu cũ sau khi làm mới).

## Phân quyền
TC-P5-06-02, 04, 06, 07, 19, 20, 24 (và `$SVB` không đọc `/me/*` của A: không có tham số danh tính; thử `GET $Q/students/<uidA>/attendance` bằng `$SVB` → `403`).

## Câu hỏi (Q-QC)
- **Q-QC-P5-12** AC4: "≤ 60 s sau (tải lại / làm mới)" vs "số không bị cache (luật 15)": QC kiểm bằng tải lại ngay (kỳ vọng thấy số mới tức thì, chấp nhận ≤ 60 s). Xác nhận.
- **Q-QC-P5-13** AC3 "thẻ KPI": ngưỡng cụ thể ("≥ 3 ô số lớn") do QC chọn từ `DESIGN.md` §21; xác nhận hoặc cho biết quy tắc `ui-antipatterns.sh`.

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
- 2026-10-13: viết lần đầu.
