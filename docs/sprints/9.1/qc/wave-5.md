# QC đợt 5 — Chat riêng, Hỏi đáp, Kết quả của tôi, Tài liệu, Insights (US-UX91-13, 19, 20, 18, 17)

**Kết luận lượt 2 (`b02eeb68`, dev: «9.1 sửa QC đợt 5»): FAIL.** 4 trong 7 lỗi lượt 1 đã sửa (UX13-17, UX19-7, UX20-6, UX18-7). Còn mở: UX13-1 và UX20-9 (chờ PM quyết P-91-13, P-91-14), lỗi mới UX13-19 (Thấp), US-08 AC10a ảnh khung ghim (Thấp), UX17-8/9 không kiểm được (P-91-15). Không có lỗi mức Trung bình. Bảng «lượt 1» ở dưới giữ nguyên làm lịch sử.

## Kiểm lại lần 2 ở `b02eeb68`
Môi trường: Mac (Apple Silicon, Colima), stack `-p ep-qc` dựng lại **sạch** (`down -v` + seed 30 SV + docling) từ `b02eeb68`, `https://localhost:943`, mock LLM. Chạy `qc-uiux91-w5a…w5h`, `w5i` (stack thật, 21 + 3 ca), 123 ca dev (`documents`, `me`, `threads`, `private-chat`, `panels`), `ui-antipatterns.sh`, `eslint`, `tsc`.
| TC | Lượt 1 | Nay | Ghi chú |
| --- | --- | --- | --- |
| UX13-17 (AC6 «chỉ bạn đọc được») | FAIL (Thấp) | PASS | Phiên trống 1440: câu có ở `main` và `body` |
| UX19-7 (AC4, 375) | FAIL (Thấp) | PASS | Bảng lọc `y 203 → 812`, rộng 375, nút `Bộ lọc` 102×44 → «Bộ lọc · 2» 122×44; trượt từ đáy |
| UX20-6 (AC6) | FAIL (Thấp) | PASS | «Chưa có điểm nào được công bố. Điểm sẽ hiện ở đây khi giảng viên công bố.» |
| UX18-7 (AC5) | FAIL (Thấp) | PASS | Chặn ghi → công tắc hoàn lại, «Chưa lưu được.» + nút `Thử lại` (1); bấm `Thử lại` → công tắc đổi |
| **UX13-1 (AC1 Panel căn giữa)** | FAIL (Thấp) | **FAIL (Thấp), chờ PM** | Panel `x 540 → 1380` (840), lề 36 / 36 trong phần bên phải cột lịch sử (504 → 1416), lề 324 / 60 so với vùng nội dung. Dev đề xuất giữ (P-91-13); theo AC hiện hành vẫn FAIL |
| UX13-3 (thu gọn lịch sử) | PASS có ghi chú | PASS | Thu gọn: Panel `x 408 → 1248`, rộng 840, căn giữa vùng nội dung (216 → 1440); CLS 0 |
| UX13-4 (tiêu đề dài) | CHƯA CHẠY | PASS | Tiêu đề 2 dòng, `-webkit-line-clamp: 2`, `overflow: hidden`, `…` ở cuối |
| **UX13-19 (AC7, nháp sau tải lại)** | PASS có giới hạn | **FAIL (Thấp)** | Xem «Lỗi mới» dưới |
| UX13-21 (AC8 gửi lại) | PASS | PASS có ghi chú | Chặn POST → ô nhập giữ chữ, `Gửi lại`; bỏ chặn → đúng 1 tin được ghi. Không đo được hai `Idempotency-Key` (phiên mới: lần 1 là `POST /chat/sessions`, không phải `…/messages`) |
| UX19-10 (AC6 lỗi API) | CHƯA CHẠY | PASS | Lỗi tải → `Thử lại`; bấm → 3 hàng |
| UX20-8 (AC7) | PASS có giới hạn | PASS | `GET /courses/‹c›/me/grades?user_id=‹B›` gọi bởi A → 200 và `qt` **của A** (8.25, không phải 10 của B); A gọi `…/students/‹B›/explain` → 403 |
| **UX20-9 (AC8, 375)** | FAIL (Thấp) | **FAIL (Thấp), chờ PM** | `sv.kha`: 2.059 px (lượt 1: 2.387), không tràn ngang. Vẫn > 1.600; dev đề xuất ≤ 2.000 (P-91-14) |
| UX13-2 (`ui-antipatterns`, `panels.spec`) | CHƯA CHẠY | PASS | `ui-antipatterns.sh` sạch, `panels.spec` xanh |
| UX17-1…7, 10, 13 | PASS | PASS | Insights 761988: 2 Panel cấp trên, 0 khung lồng, khoảng 123 px; `:active` `scale(0,97)` |
| UX17-8, 9 | KHÔNG KIỂM ĐƯỢC | KHÔNG KIỂM ĐƯỢC (FAIL theo luật) | `/insights` hiện báo cáo mô phỏng cho cả lớp 761987 trong khi `GET …/insights` của lớp đó 404: màn không đọc dữ liệu thật, không có `audit_log` thật. Đề nghị P-91-15 |
| UX91-X1 (US-08 AC10a) | FAIL (Thấp) | **FAIL (Thấp)** | `frontend/public/landing/story-*-v1.webp` không đổi từ `c7083a3d`; `b02eeb68` chỉ sửa `scripts/landing-video` (cho chạy trên stack HTTPS) chứ chưa chụp lại |
| UX91-X2 | CHƯA CHẠY | PASS có giới hạn | `ui-antipatterns.sh` sạch; `eslint .` và `tsc --noEmit` sạch; 123 ca dev (`documents`, `me`, `threads`, `private-chat`, `panels`) xanh. Cả bộ 916 ca + Lighthouse `/` là việc của dev trước handoff, QC không chạy lại |
| UX91-X4 | CHƯA CHẠY | PASS | `w5h-real`: quét chữ SV 375 mọi route, không từ cấm |
| Các ca khác `w5a`–`w5g` | PASS | PASS | 16 ca xanh lượt chạy lại (khớp bảng lượt 1) |

### Lỗi mới — UX13-19 / US-13 AC7: nháp biến mất sau tải lại khi phiên đã có tin — Thấp
Tái hiện (đã chạy thật, `sv.kha`): `/chat` → gửi «Điểm danh của tôi thế nào?» (đợi trả lời xong) → gõ «nháp sau khi gửi» trong ô nhập → đợi 2,6 s → tải lại trang.
- Sau tải lại màn mở một **phiên mới trống**, ô nhập trống; URL là `/chat` (không giữ phiên). Nháp vẫn được lưu: bấm lại hàng đầu của lịch sử thì ô nhập có «nháp sau khi gửi».
- Phiên trống không có tin (nháp gõ ngay khi mở): tải lại giữ nháp (đạt).
- TC-UX13-19 «Gõ nháp, đợi 2,2 s, tải lại → Nháp còn»: với người dùng đang trò chuyện tiếp, chữ có vẻ mất. Đề nghị mở lại đúng phiên đang xem sau khi tải lại (giữ `?session=` — màn Hỏi đáp đã dùng `/chat?session=…`), hoặc nêu trong AC rằng nháp gắn với phiên.

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

## Lỗi cần sửa (tóm tắt, lượt 2)
| # | Mức | Vị trí | Trạng thái |
| --- | --- | --- | --- |
| 2, 3, 4, 6 (lượt 1) | Thấp | US-13 AC6, US-19 AC4, US-20 AC6, US-18 AC5 | Đã sửa, PASS |
| 1 | Thấp | US-13 AC1 Panel căn giữa | Mở, chờ PM (P-91-13) |
| 5 | Thấp | US-20 AC8 trang 2.059 px > 1.600 | Mở, chờ PM (P-91-14) |
| 7 | Thấp | US-08 AC10a ảnh `story-*` chưa chụp lại | Mở: dev chạy `scripts/landing-video/run.sh` với `STILLS_ONLY=1` và `EP_BASE_URL=https://localhost:<cổng>` |
| 8 | Thấp | US-13 AC7 nháp sau tải lại khi phiên đã có tin | Mới (xem «Lỗi mới») |
| — | — | US-17 AC5, AC6 (UX17-8, 9) | Không kiểm được vì màn mô phỏng (P-91-15) |
