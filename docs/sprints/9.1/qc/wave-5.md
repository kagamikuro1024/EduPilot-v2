# QC đợt 5 — Chat riêng, Hỏi đáp, Kết quả của tôi, Tài liệu, Insights (US-UX91-13, 19, 20, 18, 17)

**ĐANG KIỂM DỞ, tiếp tục tối 09/11 (chủ dự án cho dừng).** Kết luận tạm: **FAIL** (đã có FAIL chắc chắn ở US-13, 19, 20, 18; US-17 còn mục không kiểm được). Bảng dưới là phần đã chạy; mục «CHƯA CHẠY» làm tiếp.

## Môi trường
| Mục | Giá trị |
| --- | --- |
| Commit kiểm | `9aea5b95` (mã sản phẩm = `ed0aa499` đợt 5, gồm `43f6212c`) |
| Hạ tầng | VPS `/srv/work/qc`, stack `-p ep-qc` dựng lại từ commit trên (hiện đang chạy ở `1ebfa26e` cho lượt kiểm lại đợt 4; không đụng mã đợt 5), Caddy `https://localhost:14443`, mock LLM (`env:fake`) |
| Spec QC | `qc-uiux91-w5a…w5h-real.spec.ts` (stack thật), `w5r-real` (kiểm lại đợt 1, 2), `w5x-real` (thăm dò + ảnh). Fixture danh sách Hỏi đáp dùng `page.route` trên stack thật |
| Ảnh K7 | `qc/shots/wave-5/` (1440 + 375); ảnh trước ở `audit/shots/*__student__*`, `*__teacher__*` |
| Đo 375 px | `locator.boundingBox()` (xem `wave-4.md`) |

## US-UX91-13 — Chat riêng: **FAIL**
| TC | AC | Kết quả | Ghi chú |
| --- | --- | --- | --- |
| UX13-1 | 1 | **FAIL (Thấp)** | 1440: một Panel 840 px (≤ 840 đạt), lịch sử là cột phẳng bên trái (nav 240 px, chỉ `border-right` 1 px, không bo, không bóng, nền trong suốt: đạt). Nhưng Panel **không căn giữa**: `x 540 → 1380` trong vùng nội dung `240 → 1416` (lề trái 300 px, lề phải 36 px). Nếu coi «căn giữa» là phần bên phải cột lịch sử thì lề 60 / 36 px, vẫn lệch |
| UX13-2 | 1 | CHƯA CHẠY | `ui-antipatterns.sh`, `panels.spec` (chạy ở cuối đợt, cùng X2) |
| UX13-3 | 1 | PASS có ghi chú | `Thu gọn lịch sử` có; CLS 0; nút `Phiên trước (N)` mở lại. Panel không giãn ra khi thu gọn (rộng 790 → 790), vị trí Panel sau thu gọn chưa đo |
| UX13-4, 5 | 2 | PASS có giới hạn | Hàng: tiêu đề + «Thứ Tư, 16/09 · 12:22», không thùng rác; nút duy nhất là `⋯` (32×32 ở 1440, 44×44 trong bảng phiên 375). Tiêu đề dài > 2 dòng cắt `…`: CHƯA CHẠY (dữ liệu mẫu chỉ có tiêu đề ≤ 2 dòng) |
| UX13-6, 7, 8 | 3 | PASS có ghi chú | `⋯` → `Xoá`: 0 hộp thoại xác nhận, hàng biến ngay, dòng «Đã xoá phiên» + `Hoàn tác`. Hành vi **có từ US-P3-05**: `DELETE` gửi ngay (≈ 0,25 s), `Hoàn tác` gọi `POST …/restore`; sau tải lại phiên đã xoá không còn. Bước TC-7 «không có request DELETE» và TC-8 «DELETE sau ≈ 5 s» giả định xoá trễ; AC3 ghi «hành vi hiện có» nên tính PASS (đề nghị sửa lời TC) |
| UX13-9, 10 | 4 | PASS | Phiên trống: 0 nút `Phiên mới`. Sau 1 tin: có, nút phụ (nền trắng `lab(100 0 0)`, viền 1 px, không đỏ) |
| UX13-11, 12, 13 | 5 | PASS | 3 gợi ý hàng chữ (`border-radius` 0, nền trong suốt, kẻ 1 px, cao 44): 1 từ `me/today` («Tôi nên làm gì với “Điểm bài thi Kiểm tra tuần 9 — Mật mã đã công bố”?») + 2 câu cố định. Bấm → ô nhập đầy, 0 `POST`. Khoảng trống trên ô nhập 72 / 655 = 0,11 (≤ 1/3) |
| UX13-14 | 6 | PASS | Quét chữ 1440 + 375, phiên trống và sau khi gửi: 0 từ RAG / PII / provider / fallback / trace / confidence / token / LLM / embedding / prompt / model |
| UX13-15 | 6 | PASS | «Đã ẩn 2 thông tin cá nhân trước khi gửi cho AI» |
| UX13-16 | 6 | PASS có giới hạn | `llm_audit` dòng CHAT có `pii_masked_count = 2`; bảng không lưu payload; `docker logs gateway` 3 giờ: 0 lần xuất hiện MSSV `20229002`, tên, email thật. Không đọc được payload gửi provider giả |
| UX13-17 | 6 | **FAIL (Thấp)** | Phiên trống 1440 và 375: không có chữ «chỉ bạn đọc được» ở `main` lẫn `body`. Câu «Phiên chat này chỉ bạn đọc được…» chỉ còn ở `DemoChat.tsx` và `shared/shell/intro.ts` (khung máy chủ trước khi nạp); `RealChat` thay bằng «Hỏi bất cứ điều gì về lớp này.» AC6: «giữ cam kết "chỉ bạn đọc được"» |
| UX13-18 | 7 | PASS có giới hạn | Trong lúc trả lời: gõ được, nút `Dừng` hiện. Kéo cuộn lên dừng tự cuộn: không đo (mock trả quá nhanh) |
| UX13-19 | 7 | PASS có giới hạn | Phiên trống: gõ, đợi 2,4 s, tải lại → nháp còn nguyên. Nháp trong phiên cũ chưa kiểm xong (ca QC quá hạn) |
| UX13-20 | 8 | PASS | `setOffline` ngay sau khi gửi: dải «Mất kết nối mạng. Chữ bạn đã nhập vẫn được giữ; hệ thống sẽ gửi lại khi có mạng.»; bật lại → câu trả lời hoàn tất, không `Thử lại` |
| UX13-21 | 8 | PASS | Chặn `POST …/messages`: ô nhập giữ «Tin QC kiểm gửi lại», `Gửi lại` hiện; bỏ chặn → cùng `Idempotency-Key` (2 request), chỉ 1 tin được ghi |
| UX13-22, 23 | 9 | PASS | GV, TA: `/chat` → «Bạn không có quyền xem màn này», API phiên SV 403. SV khác đọc phiên SV A: 404 |
| UX13-24 | 10 | PASS có giới hạn | 375: không tràn; `Phiên trước (N)` 351×44; bảng phiên trượt từ đáy (`y 410 → 812`), hàng 273×65–84, `⋯` 44×44; ô nhập `y 594–653` trên thanh dưới (`y 755`). Bàn phím ảo mô phỏng bằng viewport thấp 480: ô nhập đi theo (`y 297`); không có `visualViewport` thật |
| UX13-25, 26 | 11 | PASS | Tin mới: `msgIn` 160 ms `opacity + transform`; không hoạt cảnh theo token; `reduce` chỉ còn con trỏ nhấp nháy 1 ms; 6 luật `:active` có `scale` |
| UX13-27 | 12 | PASS | `private-chat.spec.ts` chỉ thêm 2 ca (gợi ý, `Phiên mới` + xoá), không đổi ca cũ; `chat-1440`, `chat-390` đổi |
| UX13-28 | K7 | PASS | `shots/wave-5/chat-1440.png`, `chat-375.png`, `chat-sheet-375.png`, `chat-after-send-1440.png` |

## US-UX91-19 — Hỏi đáp công khai: **FAIL**
| TC | AC | Kết quả | Ghi chú |
| --- | --- | --- | --- |
| UX19-1, 2, 9 | 1, 6 | PASS | Fixture (route giả): T1–T5 → «Hướng dẫn nộp bài», «Phisher - Individual uses email…», «Ví dụ», «Tiêu đề lớn», «trích dẫn»; T6 chỉ ký tự → không đoạn trích; T7 đúng 140 ký tự kèm `…`, xuống dòng thành khoảng trắng. Dữ liệu thật: «– Individual uses email…» bắt đầu bằng gạch ngang dài (`–`, không nằm trong danh sách AC); tiêu đề seed «· Phisher», «▪ Example…» còn ký tự đầu dòng (AC chỉ nói đoạn trích) |
| UX19-3, 4, 5 | 2 | PASS | 17/20 «Chờ xác nhận»: dòng đầu «17 câu chờ xác nhận», 0 hàng mang nhãn «Chờ xác nhận», T8 «Đã được giảng viên xác nhận», T9 «Đã sửa bởi giảng viên», T10 «Đã loại» vẫn có nhãn. 10/13 (77 %): không gom, cả 10 hàng có nhãn |
| UX19-6 | 3 | PASS | SV: danh sách chỉ có nội dung bài đăng, không nội dung câu trả lời AI |
| UX19-7 | 4 | **FAIL (Thấp)** | 375: `Bộ lọc` có biểu tượng, cao 44, đặt 2 bộ lọc → «Bộ lọc · 2» (đạt). Bảng mở ra là **toàn màn hình** (`y 0 → 812`, tiêu đề ở trên, `Xong` ở đáy), không phải bảng trượt từ đáy như bảng phiên chat (`y 410 → 812`); ảnh `shots/wave-5/threads-sheet-375.png` |
| UX19-8 | 5 | PASS | «Hôm qua, 22:07»; không dạng `00:10 02-11` |
| UX19-10 | 6 | CHƯA CHẠY | Lỗi API → `ApiErrorNotice` + `Thử lại` |
| UX19-11 | 7 | PASS | `sv.moi` (không thuộc lớp): `GET /courses/‹id›/threads/‹id›` 403, danh sách 403 |
| UX19-12 | 7 | PASS | Gõ MSSV + email → «Bài này có thông tin cá nhân. Hỏi đáp công khai là nơi cả lớp cùng đọc. … 1 email, 1 MSSV», `Chuyển sang chat riêng`, `Ẩn thông tin rồi đăng` |
| UX19-13 | 8 | PASS | `h1` «Hỏi đáp công khai» |
| UX19-14 | 9 | PASS | `shots/wave-5/threads-gv-1440.png`, `threads-gv-fixture-1440.png`, `threads-sv-375.png`, `threads-sheet-375.png` |
| UX19-PG | — | PASS có giới hạn | `?limit=30`; 11 thread nên chưa thấy `cursor` |

## US-UX91-20 — Kết quả của tôi: **FAIL**
| TC | AC | Kết quả | Ghi chú |
| --- | --- | --- | --- |
| UX20-1 | 1 | PASS | Một khối: «Giữa kỳ · 20 %» + «Căn cứ · tr. 1» + 7,5 cùng dòng; 5 `li`; khối liệt kê thành phần riêng không còn. Không có cột «đóng góp» (P-91-11, đã chấp nhận) |
| UX20-2 | 2 | PASS | Dưới `h1` không dòng mã lớp trần; «Điểm quá trình hiện tại của bạn là 8,25 trên 10.» còn |
| UX20-3 | 3 | PASS | 7,5 / 8 / 6,5, bonus 0,5, trừ 0, 7,75, 8,25 khớp `GET /courses/‹id›/me/grades` |
| UX20-4 | 4 | PASS | Không có «nháp», «rủi ro», «ghi chú quan sát» trên màn |
| UX20-5 | 5 | PASS (không đổi) | `/me` không có nút `Yêu cầu xem lại`, trước và sau sprint (`AppealBlock` thuộc màn bài tập); lời TC cần sửa |
| UX20-6 | 6 | **FAIL (Thấp)** | Giả `me/grades` toàn `PENDING`: «Chưa có điểm được công bố.» (AC: «Chưa có điểm nào được công bố.» và nêu nơi sẽ xuất hiện; không có câu thứ hai) |
| UX20-7 | 6 | PASS | API lỗi → «Không tải được điểm của bạn.» + `Thử lại` |
| UX20-8 | 7 | PASS có giới hạn | SV A gọi `…/gradebook/students/‹uid B›/explain` → 403. `me/grades?user_id=B` trả 200 (tham số có bị bỏ qua không: CHƯA CHẠY, `w5i`) |
| UX20-9 | 8 | **FAIL (Thấp)** | 375 `sv.kha`: `scrollHeight` **2.387 px** (> 1.600; trước 2.645). Không tràn ngang. Dev đã báo 1.749 px trên fixture riêng |
| UX20-10 | 9 | PASS | `me.spec.ts` đổi theo AC (5 `li`, «Giữa kỳ · 20 %», «Chưa công bố»), không nới |
| UX20-11 | 10 | PASS có ghi chú | `shots/wave-5/me-375.png`, `me-nguyco-1440.png`, `me-empty-1440.png`. Không có ảnh mốc `visual.spec` `me-390` (AC ghi «QC quyết»: không bắt buộc) |

## US-UX91-18 — Tài liệu: **FAIL**
| TC | AC | Kết quả | Ghi chú |
| --- | --- | --- | --- |
| UX18-1 | 1 | PASS | 1440 và 1100: ô «Loại» là `select` rộng 167–169 px, `scrollWidth = clientWidth`; «Quy chế môn học», «Tài liệu khác», «Đề tham khảo» hiển thị đủ |
| UX18-2, 3 | 2 | PASS | 761987 (tài liệu của lớp): không «Chia sẻ từ». 761988: 5 tài liệu chia sẻ từ 761987 đều có «Chia sẻ từ 761987» |
| UX18-4 | 3 | PASS | 12 công tắc có tên «Dùng cho AI: bật / tắt» (6) và «Hiện cho sinh viên: …» (6); bật = mực, tắt = xám nhạt, đồng nhất mọi hàng. Công tắc khoá (tài liệu chia sẻ, đáp án) mờ 0,55 (dev đã khai) |
| UX18-5 | 3, 7 | PASS | 375: công tắc 44×44; không tràn ngang |
| UX18-6 | 4 | PASS | SV `/library`: có «Đề tham khảo tuần 5», không có «Đáp án» |
| UX18-7 | 5 | **FAIL (Thấp)** | Chặn `PATCH` / `PUT` / `POST`, đổi công tắc: công tắc hoàn lại (đạt), dòng lỗi «Chưa lưu được. Thử lại.» hiện, nhưng **không có nút `Thử lại`** (0 nút; câu chỉ là chữ). AC5 và TC-7 đòi `Thử lại` bấm được |
| UX18-8 | 6 | PASS | SV `/documents` → «Bạn không có quyền xem màn này»; API 403 |
| UX18-9 | 7 | PASS | Xem UX18-5 |
| UX18-10 | 8 | PASS có ghi chú | `shots/wave-5/documents-761987-1440.png`, `-375.png`; không có ảnh mốc `visual.spec` `documents-1440` (AC: «QC quyết») |
| UX18-PQ | — | PASS | TA, lớp lạ: `GET /courses/‹id›/documents` 403 |
| UX18-ID | — | PASS | Bấm nhanh 5 lần: true → false, tải lại vẫn false (đúng lần cuối) |

## US-UX91-17 — Lỗ hổng kiến thức và Quan sát AI: **FAIL (có mục không kiểm được)**
**Phát hiện:** `/insights` và `/observability` chạy bằng **dữ liệu mô phỏng** (`frontend/src/mock/insights.ts`); đổi lớp sang `QC117283` vẫn hiện báo cáo «An ninh mạng – 761988». Không có API thật → AC5 (`audit_log`) và AC6 (lớp chưa có báo cáo) không kiểm được. Đề nghị PM quyết (P-91-12).
| TC | AC | Kết quả | Ghi chú |
| --- | --- | --- | --- |
| UX17-1, 2 | 1 | PASS | Insights: 2 Panel («Chủ đề nên dạy lại», «Tài liệu chưa đề cập»), 0 phần tử con vừa có viền vừa bo góc; chủ đề ngăn bằng đường kẻ. Khối «So với báo cáo ngày …» không hiện ở dữ liệu mẫu |
| UX17-3 | 2 | PASS có ghi chú | Còn một `meta` mỗi khối («Câu hỏi trong 7 ngày liền trước đó, đã ẩn danh»); hộp riêng tư là «Đã bỏ tên và MSSV · chủ đề dưới 3 sinh viên không hiện câu mẫu» = 14 từ: đúng chữ mẫu của US dòng 25 nhưng vượt «≤ 12 từ» (US tự mâu thuẫn) |
| UX17-4, 5 | 3 | PASS có ghi chú | Số «hôm nay» là một dòng chữ (không dải ô); số 7 ngày là bảng `Chỉ số / Giá trị / Ghi chú`, hàng đầu «Yêu cầu 7 ngày 412». Yêu cầu, p95, tỷ lệ lỗi xuất hiện ở cả dòng «hôm nay» và bảng 7 ngày (hai khoảng thời gian khác nhau) |
| UX17-6 | 4 | PASS | Khoảng trước tiêu đề khối: 80 px (Insights), 57 px (Quan sát AI), > 32 và lớn hơn khoảng dưới tiêu đề (4 px) |
| UX17-7 | 5 | PASS | Giảng viên chọn hàng yêu cầu: không hiện nội dung câu hỏi / prompt |
| UX17-8 | 5 | KHÔNG KIỂM ĐƯỢC | Admin chọn hàng: mở 1 hộp thoại, nhưng màn là dữ liệu mô phỏng, không có `audit_log` thật |
| UX17-9 | 6 | KHÔNG KIỂM ĐƯỢC | Không có lớp «chưa có báo cáo» (mô phỏng) |
| UX17-10 | 6 | PASS có giới hạn | `Tạo báo cáo mới`: «Đang tạo báo cáo · Đang chạy · Gom câu hỏi của lớp…» rồi «Xong»; rời trang, quay lại báo cáo mới 29/10 hiện |
| UX17-11 | 7 | PASS | SV: `/insights`, `/analytics`, `/observability` → «Bạn không có quyền xem màn này». TA: `/insights`, `/analytics` mở được (quyền hiện có), `/observability` bị chặn |
| UX17-12 | 8 | PASS | 375: không tràn (Insights cao 3.034 px, Quan sát AI 3.261 px) |
| UX17-13 | 9 | PASS | `:active` của `Tạo báo cáo mới` = `matrix(0,97…)` |
| UX17-14 | 10 | PASS | `shots/wave-5/insights-1440.png`, `-375.png`, `observability-1440.png`, `-375.png`; ảnh mốc `insights-*`, `observability-*` có trong `visual.spec` |
| UX17-PQ | — | PASS | TA lớp lạ: `GET /courses/‹id›/documents` 403 (màn Insights không có API) |

## Kiểm chéo cuối sprint (đợt 5)
| TC | Kết quả | Ghi chú |
| --- | --- | --- |
| UX91-X1 (US-08 AC10a) | **FAIL (Thấp)** | `frontend/public/landing/story-*-v1.webp` không đổi từ `c7083a3d`. Ảnh `story-chat` còn thùng rác từng hàng + Panel lịch sử cũ, `story-gradebook` còn bố cục cũ, `story-review` khác UI mới. Handoff đợt 5 không ghi chụp lại. Dev chạy lại `scripts/landing-video/run.sh` với `STILLS_ONLY=1` (≈ 1 phút) |
| UX91-X2 | CHƯA CHẠY | `playwright` cả bộ, `ui-antipatterns.sh`, `pnpm lint && build`, Lighthouse `/`. Dev báo 916 pass / 1 flaky (spec QC `w3.spec:280`) |
| UX91-X3 | PASS có ghi chú | Không migration mới; không `fetch(` trần mới; 3 `Demo*.tsx` bị sửa từ đợt 1+2 (đổi chữ «Threads» → «Hỏi đáp công khai» theo US-07 AC4; `DemoDocuments` đổi `progressStyle`), trái câu «không sửa `Demo*.tsx`» của US-01 nhưng cần cho US-07 AC4 |
| UX91-X4 | CHƯA CHẠY | `w5h-real.spec.ts` đã viết, chưa chạy |

## CHƯA CHẠY (làm tiếp tối 09/11)
UX13-2, UX19-10, UX20-8 (tham số `user_id`), UX13-4 (tiêu đề dài), UX13-3 (vị trí Panel sau thu gọn), UX13-19 (phiên cũ), X2, X4. Ca chuẩn bị sẵn: `qc/scripts/pending/qc-uiux91-w5i-real.spec.ts.txt`.

## Lỗi cần sửa (tóm tắt)
| # | Mức | Vị trí | Tái hiện |
| --- | --- | --- | --- |
| 1 | Thấp | US-13 AC1 | `/chat` 1440: Panel `x 540–1380`, lề trái 300 / phải 36 so với vùng nội dung |
| 2 | Thấp | US-13 AC6 | Phiên trống không còn «chỉ bạn đọc được» (chỉ ở `DemoChat`, `intro.ts`) |
| 3 | Thấp | US-19 AC4 | 375: `Bộ lọc` mở toàn màn hình, không là bảng trượt từ đáy |
| 4 | Thấp | US-20 AC6 | Rỗng: «Chưa có điểm được công bố.» (thiếu «nào», thiếu nơi sẽ xuất hiện) |
| 5 | Thấp | US-20 AC8 | 375 `sv.kha`: trang cao 2.387 px (> 1.600) |
| 6 | Thấp | US-18 AC5 | Lỗi đổi công tắc: «Chưa lưu được. Thử lại.» là chữ, không có nút `Thử lại` |
| 7 | Thấp | US-08 AC10a | Ảnh khung ghim trang giới thiệu chưa chụp lại từ UI mới |
