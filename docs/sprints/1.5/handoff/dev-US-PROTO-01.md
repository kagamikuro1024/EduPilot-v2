# DEV handoff — US-PROTO-01 Sinh viên
Nhánh `sprint/1.5-mock-ui` · commit: `US-PROTO-01: ...`

## Đã làm (theo thứ tự lát dọc)
- Mock data: `frontend/src/mock/{student,chat,threads,practice,library,calendar}.ts`.
- Màn hình tính năng (`frontend/src/features/`):
  - `today/StudentHome.tsx` + `StudentHome.module.css`: khuyến nghị duy nhất (A: ôn Mật mã đối xứng sai 4/7 · 15p; B: QUIZ01 đóng sau 18 giờ · 20p; nút đỏ duy nhất), dòng thời gian hôm nay (buổi 10 09:00–11:30 P.302 đang diễn ra, hạn QUIZ01), học dở. SV D: ô nhập mã tham gia thay khuyến nghị ('mã tham gia'). ActionRow redThread.
  - `chat/`: D1 (dòng ẩn 2 thông tin cá nhân, trả lời chảy useStreamedText, nêu vắng 2 buổi (10/09, 08/10), +0,75 điểm cho 3 lần phát biểu, khối số liệu gọn, nguồn tham khảo trích Quy chế và Sổ điểm danh 761987, Hữu ích/Không hữu ích/Nhờ GV hỗ trợ, không chuỗi `[[SV_1]]`), D2 (từ chối lịch sự, không lộ số liệu bạn C), D3 (AI chưa đủ chắc chắn → tự chuyển GV, tạo ticketD3, nút thành 'Đang chờ giảng viên · vừa gửi'; khi GV trả lời hiển thị câu trả lời nhãn GV và nút 'Đã rõ'). Lịch sử 4 phiên (ẩn ≤ 720px); `?state=error` giữ nguyên chữ trong composer.
  - `threads/`: 12 thread lớp 1, 2 ghim, ghim từ Insights hiện đầu. Soạn bài có MSSV/tên/'điểm của em' mở `PIIChannelDialog` hai lối (Chuyển sang chat riêng / Ẩn thông tin rồi đăng). `threads/[id]`: câu trả lời AI 'Chờ xác nhận', câu trả lời GV có đường kẻ xanh 1px + nhãn xác nhận. Báo cáo, sửa/xoá bài của mình.
  - `practice/`, `practice/[attemptId]`, `practice/history`: luyện chủ đề (chọn đáp án phản hồi ngay có giải thích) và QUIZ01 (có đếm giờ, nộp bài qua xác nhận, Chat riêng chỉ thủ tục). Lịch sử 6 lượt.
  - `library/`: tài liệu visible_to_students (không ANSWER_KEY), tìm kiếm lọc, xem chi tiết, 'Hỏi AI về tài liệu' → /chat có ngữ cảnh, đề cũ có 'Luyện đề này'.
  - `calendar/`: Tuần / Tháng / Danh sách; mobile mặc định Danh sách; đọc `KEYS.calendarExtras`; hiển thị buổi 10, QUIZ01, hạn BT, giữa kỳ 05/11, cuối kỳ tuần 16.
  - `me/`: câu nhận định điểm QT 8,3 (tạm tính); giải trình tuyến tính TB bài tập 7,5 + cộng 0,75 = 8,25 → 8,3; sau điểm danh lên 8,5; sau công bố BT03 lên 8,7. What-if CK=8,0 → 8,3; số ngoài 0–10 báo lỗi tại ô. SV A chọn lớp 2 → thông báo lớp chưa có công thức điểm chính thức. Không có từ cấm, không nhãn rủi ro.
  - `assignments/[id]`: BT03 của B: đã nộp muộn 1 ngày, trước công bố ghi 'Đang chấm' (không lộ số); sau công bố: 8,0 điểm, nhận xét 4 tiêu chí trích bài của B, 'Yêu cầu xem lại' trong 7 ngày. QUIZ01: nút làm bài.
  - `join/`, `join/[code]`: BX4P9TW xem trước An ninh mạng · 761988 · TS. Lê Thu Hà · HK1 2026–2027 → 'Tham gia lớp' → pending + chờ duyệt; AN7K2MQ → vào ngay; mã sai → câu chung 'Mã không hợp lệ hoặc đã hết hạn...'; sai 5 lần → khoá 10 phút.
- Routes app: `frontend/src/app/(app)/{chat,threads,threads/[id],practice,practice/[attemptId],practice/history,library,calendar,me,assignments/[id],join,join/[code]}/page.tsx`.

## File đổi
`frontend/src/features/{chat,threads,practice,library,calendar,me,assignments,join}/**`
`frontend/src/features/today/StudentHome.tsx`, `StudentHome.module.css`
`frontend/src/mock/{student,chat,threads,practice,library,calendar}.ts`
`frontend/src/app/(app)/{chat,threads,threads/[id],practice,practice/[attemptId],practice/history,library,calendar,me,assignments/[id],join,join/[code]}/page.tsx`
`docs/sprints/1.5/shots/01-*.png`

## Lệnh QC chạy để kiểm
```bash
source ~/.zprofile
# Kiểm tra tĩnh và chạy test curl của QC
pnpm -C frontend lint && pnpm -C frontend build && bash scripts/ui-antipatterns.sh
F=http://localhost:3000 bash docs/sprints/1.5/qc/scripts/proto-curl.sh tc_01_01 tc_01_05 tc_01_08 tc_01_09
```

## Test đã chạy và kết quả
- `proto-curl.sh tc_01_01 tc_01_05 tc_01_08 tc_01_09`: PASS 100% (mọi route mở được với SV B, sạch từ cấm RAG/PII/LLM/confidence/risk, không lộ điểm nháp, xem trước lớp 761988, phân quyền chặn đúng các vai khác).
- Trình duyệt: ảnh chụp desktop 1440 và mobile 390 tại `docs/sprints/1.5/shots/01-*.png`.

## AC tự đánh giá
01-AC1 ✓ · 01-AC2 ✓ · 01-AC3 ✓ · 01-AC4 ✓ · 01-AC5 ✓ · 01-AC6 ✓ · 01-AC7 ✓ · 01-AC8 ✓ · 01-AC9 ✓.

## Nợ / chưa làm / cần hỏi
- Kịch bản What-if hiện tính theo công thức chuẩn đã chốt (QT 8,7 sau công bố, CK 8,0 → HP 8,3).
- Phần xem trước PDF bài giảng trong Thư viện hiển thị trang giả lập dạng văn bản/khung xem trước.

## Cập nhật theo spec v4 (Proposals #15, #16)
- Form tạo thread mới (`ThreadsScreen.tsx`): đầy đủ Tiêu đề (Input bắt buộc), Chủ đề (Select `THREAD_TOPICS`), Nội dung chi tiết (Textarea bắt buộc), Checkbox "Nhờ AI trả lời gợi ý (Socratic) ngay sau khi đăng" (mặc định bật), nút Đăng câu hỏi (primary). Quét PII 2 lối. Đăng thành công chuyển hướng ngay sang `/threads/${id}` (01-AC4).
- Chi tiết thread (`ThreadDetail.tsx`): cấu trúc 4 panel độc lập rõ ràng (câu hỏi gốc, câu trả lời AI, thảo luận, Reply Composer). Trích dẫn citations bấm mở rộng xem snippet đoạn trích.
- Reply Composer ở cuối trang: nhập phản hồi, nút "Hỏi trợ lý AI" gợi ý Socratic, quét PII trước khi gửi (01-AC10).

## v5 / v5.1 — bổ sung

**v5:** 01-AC10 Threads như thật (seed thảo luận 12 thread, `Thảo luận (n)` tính từ dữ liệu, một vùng thảo luận + ô soạn ở cuối); 01-AC11 `/chat` cùng trục (history 240 px, composer ghim đáy ≤ 24 px); 01-AC12 `Hỏi trợ lý AI` (mẫu S1…L1, chọn theo từ khoá không dấu, nhánh không khớp); 01-AC13 phản hồi trễ của TA + chuông (lưu `due` trong `ep_demo_state`, chạy ở `ThreadsBackground` nên sống qua đổi route / tải lại); 01-AC14 nhánh không khớp, bản nháp theo thread.

**v5.1 (#19, #20):**

| E | Đã sửa | AC |
| --- | --- | --- |
| E1 | Luyện đề: màn kết quả cùng route, ghi lịch sử đúng lúc bấm `Xem kết quả`, mở lại đúng câu dở, chấm nguyên từ; sập ở `Xem kết quả` do chỉ số vượt quá cuối bài | 01-AC15 |
| E2 | SV D chưa vào lớp: màn "Bạn chưa vào lớp nào" (SSR không có dữ liệu lớp), sidebar chỉ `Hôm nay`; dữ liệu cá nhân khoá theo `studentId` (chat, luyện đề, QUIZ01) | 01-AC17 |
| E4, E26 | Kho thông báo `mock/notes.ts` (SRS 4.9), chấm chưa đọc `data-part=bell-dot`, mục "đang chờ chấm" tính từ dữ liệu | 01-AC23 |
| E5 | `mock/pii.ts` (email, SĐT, MSSV, họ tên, điểm gắn danh tính) ở 4 điểm gửi; dialog nêu số lượng theo loại, đúng hai nút; sau ẩn: "Đã ẩn n thông tin cá nhân" (#20c) | 01-AC18 |
| E10 | Một bảng tài liệu `mock/docs.ts` (N5) cho Thư viện, Nguồn tham khảo, chuông | 01-AC24 |
| E11 | `QUIZ01_ITEMS` 8 câu riêng, 0 trùng với ngân hàng luyện | 01-AC16 |
| E13, E14 | Chủ đề mặc định "Chọn chủ đề", không có "Thông báo" với SV; chi tiết/danh sách đúng chủ đề + tuần, chip "Của bạn", trích đoạn ≤ 120 ký tự, URL `t-new-<n>` | 01-AC19 |
| E15 | `Hỏi trợ lý AI` có chữ: phản hồi của người trước, AI ngay dưới theo mẫu khớp; ô soạn trống sau khi bấm | 01-AC20 |
| E20 | `/chat` < 1100 px (kể cả 720–1099 theo #20b): nút `Phiên trước (n)` + bảng trượt; `Hỏi tiếp` | 01-AC22 |
| E24 | Lịch tuần cao theo nội dung, nhãn tối đa 2 dòng | 01-AC25 |
| E27 | `/me` "Cập nhật" lấy giờ giả lập của sự kiện cuối | 01-AC26 |
| E32, E34 | Form `Đặt câu hỏi` mở tại chỗ; dòng "Còn thiếu: …" cạnh nút khoá + `aria-describedby` | 01-AC21 |
| E36 | Thẻ "Câu hỏi gốc" không còn khoảng trống (đáy thẻ − phần tử cuối = 21 px) | 01-AC27 |
| – | J1/J2: một `Gợi ý thêm` mỗi thread, dòng thông báo ở lần bấm sau; phản hồi trễ cũng chạy qua `Hỏi trợ lý AI` có chữ (chỉ SV, một lần mỗi thread) | 01-AC28 |

Tự kiểm (đã chạy): `/threads/t-cbc` ô trống bấm `Hỏi trợ lý AI` → +1 bài (n = 5), bấm thêm hai lần → n giữ 5 + dòng "Trợ lý đã đưa hết gợi ý…"; thread mới "Dùng lại IV trong CTR có sao không?" → S2 với trình tự soạn → chảy chữ → `Nguồn tham khảo (1)`; P1 → dialog "1 địa chỉ email, 1 số điện thoại", `Ẩn thông tin rồi đăng` → `Mail của em là [đã ẩn], SĐT [đã ẩn].`, `ep_demo_state` không còn `0912345678`; `t-salt` "Em hiểu rồi ạ" → "Phạm Quốc Bảo đang trả lời…" ở 2 s, TA trích bài của B và chuông có chấm khi đang ở `/calendar` (không cần ở lại trang).

### AUDIT v5.1 (prod/dev server :3300, trình duyệt headless)

Chạy `AUDIT` + `TOUCH` + `LEFT` (US.md "Quy ước kiểm chung") trên mọi route × vai: 135 lượt đo (SV 15 route × 1440/390/375; TA 17 + GV 20 + Admin 6 route × 1440/390 (+ 375 cho `/attendance`, `/inbox`)).

| Vai | Số lượt | `ox` ≠ 0 | `cut` | `ell` | `TOUCH` (≤ 390) |
| --- | --- | --- | --- | --- | --- |
| Sinh viên | 45 | 0 | 0 | 0 | 0 |
| Trợ giảng | 36 | 0 | 0 | 0 | – |
| Giảng viên | 42 | 0 | 0 | 0 | 0 (`/attendance`, `/inbox`) |
| Admin | 12 | 0 | 0 | 0 | – |

`LEFT` (`[data-part=page-title]`): mọi route = **240** ở 1440, **16** ở 390 và 375. `curl` bộ `proto-curl.sh all` (bản đã sửa lỗi biến `$s»` của bash UTF-8): 496 PASS, 1 FAIL giả ở `tc_04_08` — xem góp ý #23 (`grep -c` đếm dòng, HTML SSR chỉ một dòng; `grep -o … | wc -l` ra đúng 3). `pnpm -C frontend lint`, `tsc --noEmit`, `bash scripts/ui-antipatterns.sh` sạch.
Ảnh: `docs/sprints/1.5/shots/v5/` (thread, danh sách, Hôm nay GV, inbox 1440/375, điểm danh 390, chat 390/900/1440, SV D chưa vào lớp, `/settings/llm`, luyện đề 390, analytics).

## Sửa lỗi QC v5 vòng 1 (US-PROTO-01)

| Lỗi | Đã sửa | Cách tự kiểm |
| --- | --- | --- |
| BUG-v5-01-5 = DEMO-2 đoạn trích không phải bài của B | Bỏ `BT03_EXCERPTS` (student.ts); `/assignments/bt03` đọc `BT03_EVIDENCE` (assess.ts) — cùng nguồn với `/grading` | Công bố BT03 → SV B: 4 đoạn trích nói về SolarWinds / SBOM / Golden SAML |
| BUG-v5-DEMO-4 mất câu trả lời GV sau `Đã rõ` | `TeacherHandoff`: câu trả lời + nhãn giảng viên ở lại, `Đã rõ` đổi thành `Câu hỏi đã đóng` | GV trả lời tk-d3 → SV `Đã rõ` → câu trả lời còn; tải lại vẫn còn |
| BUG-v5-DEMO-7 SV D lớp 761988 thấy Buổi 10 + học dở | `StudentHome`: lớp không học hôm nay (≠ lớp 1) → 'Hôm nay lớp không có buổi học', bỏ 'Học dở', buổi tiếp theo Thứ Ba 03/11 | SV D vào 761988 → Hôm nay |
| BUG-v5-DEMO-11 thân tin nhắn ≠ chip | `Msg.stats` chụp số liệu lúc trả lời; chip đọc `m.stats` | Ghi +0,25 sau D1 → thân và chip cùng '3 lần · +0,75' |
| BUG-v5-00-3 nhãn chờ giảng viên đứng yên, chuông thiếu D3 | `sentLabel()` (derive.ts) theo `ago()`; `ticketD3(simNowMs())`; `noteTicketSent` đẩy mục vào chuông SV | Lệch đồng hồ +59 phút: chuông 'Câu hỏi của bạn đang chờ giảng viên · 59 phút trước'; +23 giờ: nhãn '23 giờ trước' |
| BUG-v5-01-3 nguồn D1 chỉ 1, thu gọn | `D1_CITATIONS` thêm 'Sổ điểm danh lớp 761987'; danh sách mở sẵn | D1: 'Nguồn tham khảo (2)' mở ngay |
| BUG-v5-01-4 What-if trống báo lỗi giả | `MeScreen`: trống không `aria-invalid`, không câu lỗi | Xoá chữ ô điểm giả định: `aria-invalid` null, 0 `[role=alert]` |
| BUG-v5-01-6 nhãn lịch tháng cắt … | `.chip` bỏ `nowrap/ellipsis` | `/calendar` Tháng 1100: nhãn QUIZ01 hiện đủ 2 dòng |

**Tự kiểm (build production :3400, trình duyệt headless):** `regress-v24.mjs` 30/30 PASS · `audit.mjs` 495/495 PASS (SV 165, TA 106, GV 170, Admin 54) và `states:true` GV + Admin 544/544 PASS · `demo-run.mjs` 21/21 hàng PASS, 173 s (< 13:45) · `proto-curl.sh all` 497 PASS / 0 FAIL · `pnpm lint` sạch · `pnpm build` OK · `bash scripts/ui-antipatterns.sh` 0 ✗. Commit: `4f7b5b5` (00) · `5086445` (02) · `821be9d` (01) · `0fc7ffb` (03) · `94dbf38` (04) · `6a043ae`.
