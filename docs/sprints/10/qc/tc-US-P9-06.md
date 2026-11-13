# QC test case — US-P9-06 Luyện theo chủ đề, phản hồi ngay, giải thích thêm
Nguồn: US-P9-06 + SRS FR-9, 11–15, §3, §6 #5–#11, §7. 375 px là ngưỡng chính (SV dùng điện thoại). Chụp ảnh vào `qc/shots/p9-06/`.

| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-06-01 | AC1 | `sv.kha` có lỗi gần đây ở chủ đề T (seed ≈ 40 lượt mô phỏng) | Mở `/practice` 1440 và 375 | Khung nhìn đầu: **một** khối khuyến nghị (chủ đề + số câu + `Luyện ngay`); dưới đó `Theo chủ đề` và `Thi thử`; dòng `Tiếp tục` nếu có lượt dở; **không** số thành tích, xếp hạng, vòng tiến độ | E2E |
| TC-06-02 | AC1 | SV chưa có lượt nào (`sv.gioi` mới, xoá lượt qua DB hoặc SV khác) | `GET /practice/overview` | Khuyến nghị = chủ đề có **nhiều câu `APPROVED` nhất** | API |
| TC-06-03 | AC1 | – | `GET overview` | Có danh sách chủ đề + số câu đủ điều kiện, lượt dở (nếu có); chủ đề biến thể «TCP/IP», «tcp ip» gộp một (Q-P9-11; kiểm với câu có hai cách viết, hiển thị tên biến thể nhiều nhất) | API |
| TC-06-04 | AC2 | Chủ đề T ≥ 5 câu | `IDEM=k1 api sv.kha POST …/practice/start '{"mode":"TOPIC","topic":"T","count":10}'`; gửi lại cùng `k1`; rồi khoá mới `k2` | Lần 1 `201` lượt mới; lần 2 (cùng khoá) cùng lượt; lần 3 (khoá mới, lượt dở cùng chủ đề) `200` **lượt dở cũ** (cùng `attempt_id`); UI chuyển tới `/practice/<id>` ở **câu đang dở** | API |
| TC-06-05 | AC2 | – | `start` khác chủ đề khi còn lượt dở chủ đề T | `201` lượt mới (UNIQUE theo chủ đề) | API |
| TC-06-06 | AC2 | – | `start` thiếu `Idempotency-Key`; `count=4`; `mode` lạ | Lỗi `400/422` theo spec | API |
| TC-06-07 | AC3 | Lượt TOPIC 10 câu | Mở `/practice/<id>` 1440 và 375 | Một câu mỗi lần; tiêu đề gọn: `3/10` + `Thoát`; **không đồng hồ**; ≥ 1024 px có danh sách số câu thu gọn; 375 px không có danh sách số câu bắt buộc (không tràn) | E2E |
| TC-06-08 | AC4 | Câu 1 | Chọn đáp án, bấm `Trả lời`; đo thời gian request `PUT …/answers/{itemId}` | Phản hồi trong **cùng response**: đúng/sai, đáp án đúng, giải thích, trích dẫn bấm được → `/library` đúng tài liệu/trang; p95 ≤ 500 ms qua 50 lần; UI chữ + biểu tượng, **không** nền đỏ/xanh | API + E2E |
| TC-06-09 | AC4 | Đã có phản hồi câu 1 | Đổi đáp án qua UI; `PUT` lại với đáp án khác | UI khoá lựa chọn; API `409 ANSWER_LOCKED`; giá trị đã chốt không đổi (đọc lại `GET attempt`) | API + E2E |
| TC-06-10 | AC4 | – | `setOffline(true)`; chọn + `Trả lời`; chờ 3 s; `setOffline(false)` | Hiện «Chưa gửi được» tại câu; phản hồi đúng/sai **chưa** hiện khi offline; sau khi có mạng gửi lại (cùng khoá/`version`), phản hồi hiện; DB có **đúng 1** `practice_answers` cho item | E2E + DB |
| TC-06-11 | AC5 | Đã trả lời câu (đúng hoặc sai) | `POST …/items/{itemId}/explain`, SSE; đo sự kiện đầu | Sự kiện đầu ≤ 300 ms (p95 qua ≥ 20 lần, provider `fake`); đúng **một** lời gọi LLM (`llm_audit`: 1 dòng `task` giải thích cho lần gọi, làn INTERACTIVE); tiếng Việt, ngắn | API |
| TC-06-12 | AC5 | Chưa trả lời câu | `explain` | `409 NOT_ANSWERED`; UI nút khoá có tooltip lý do | API + E2E |
| TC-06-13 | AC5 | Câu đã `explain` ở TC-06-11 | Gọi lại `explain` trong 10 s; rồi sau khi GV sửa câu (`PUT …/questions/{id}`, `version` tăng), SV gọi lại | Lần lặp trong 10 s: cùng nội dung, **không** lời gọi LLM mới (`llm_audit` không tăng; cache Redis `ep:practice:explain:*`); sau khi câu sửa: cache vô hiệu (lời gọi mới) | API + DB |
| TC-06-14 | AC5 | – | Đọc payload `explain` ở `llm_audit` (ADMIN, `audit_log`) | Chỉ có đề, đáp án đúng, giải thích, chunk trích dẫn, đáp án SV đã chọn; **không** tên / MSSV / email (đặt tên giả vào `display_name` SV rồi kiểm 0 xuất hiện) | DB |
| TC-06-15 | AC6 | – | Quét mọi `llm_audit` task `PRACTICE_*` từ phiên | 0 chuỗi tên / MSSV / email thật | DB |
| TC-06-16 | AC7 | Lượt 10 câu đã trả lời đủ | `IDEM=s1 api sv.kha POST …/attempts/{aid}/submit`; gửi lại `s1`; gửi `s2` | `200` tóm tắt: `{đúng}/{tổng}`, chủ đề yếu nhất, nút `Luyện tiếp chủ đề này` / `Về Luyện đề`; **không** điểm / xếp hạng / pháo hoa; gọi lại (cả khoá mới) cùng kết quả | API + E2E |
| TC-06-17 | AC7 | Lượt còn câu chưa trả lời | `submit` | `200`, câu bỏ trống tính 0 (P-10-7 Q11) | API |
| TC-06-18 | AC8 | Lượt của A | `sv.gioi`: `GET attempt`, `PUT answers`, `explain`, `submit`, `PATCH`; | Cả năm `404` (không lộ) | API |
| TC-06-19 | AC8 | – | TEACHER, TA, ADMIN: `POST practice/start`, `PUT answers` | `403` (`reason: role`) | API |
| TC-06-20 | AC9 | `fake_llm.json` trả `OVERLOADED` cho explain | Bấm `Giải thích thêm` | Tại chỗ: «Chưa giải thích được. Thử lại.»; phản hồi đúng/sai và giải thích sẵn có **không** đổi; làm tiếp + `submit` bình thường | E2E |
| TC-06-21 | AC10 | Lượt, đang ở câu 6, đã chọn (chưa bấm `Trả lời`) đáp án ở câu 6 | Tải lại trang (E2E) | Quay đúng câu 6; 5 câu trước ở trạng thái đã trả lời đúng/sai; lựa chọn chưa gửi ở câu 6 còn (nháp phía máy) | E2E |
| TC-06-22 | AC11 | Chủ đề có 4 câu `APPROVED` | `start` | `409 NOT_ENOUGH_QUESTIONS`; UI «Chưa đủ câu để luyện chủ đề này» | API |
| TC-06-23 | AC11 | Lớp 2 xoá hết ngân hàng (hoặc lớp thử chưa có câu `APPROVED`; tạo lớp thử qua `/admin/courses`) | Mở `/practice` | Một dòng «Giảng viên chưa mở câu hỏi», không nút | E2E |
| TC-06-24 | AC12 | 375×812 touch | Làm trọn lượt 10 câu bằng thao tác chạm | Không cuộn ngang (`scrollWidth ≤ clientWidth` mỗi câu); mọi nút ≥ 44 px (đo `getBoundingClientRect`); thanh hành động cố định dưới (`position: sticky/fixed`, không bị cắt) | E2E |
| TC-06-25 | SRS §7 | – | `/practice` và `/practice/<id>` ở trạng thái tải / lỗi (`route.abort`) / rỗng | `PageState` đúng: skeleton; lỗi + `Thử lại`; rỗng như TC-06-23 | E2E |
| TC-06-25b | AC4, AC7 | Câu `MCQ_MULTI` đáp án đúng {a,b,c} | Trả lời lần lượt `{a,b,c}`; `{a,b}`; `{a,b,c,d}`; `{a,d}` | Chỉ `{a,b,c}` là **đúng** (trùng toàn bộ tập, P-10-7 Q25); ba cái còn lại **sai**; `đúng/tổng` luôn số nguyên (không điểm một phần) | API |
| TC-06-26 | UI | – | Quét chữ SV 375 px: `/practice`, `/practice/<id>`, `/practice/history` | Không có «AI», «RAG», «citation», «model», «LLM», «prompt», «token»; dùng «Nguồn trong tài liệu»; nút là động từ (`Luyện ngay`, `Trả lời`, `Giải thích thêm`, `Nộp bài`) | E2E |

## Nhánh lỗi (SRS §3)
Phủ bởi TC-06-10 (mất mạng), 12, 17, 20, 22.

## Phân quyền
TC-06-18, 19; SV lớp khác `start` ở `C1` → `403`/`404`; `sv.moi` → `403`/`404`.

## Lịch sử sửa TC
- 2026-11-14 — spec FEAT-practice v1.1 + PM trả lời P-10-7 (Q-S10-1…26): TC-06-17 (Q11), thêm TC-06-25b (Q25: `MCQ_MULTI` đúng = trùng toàn bộ tập).
