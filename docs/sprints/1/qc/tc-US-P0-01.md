# QC test case — US-P0-01
Nguồn: `docs/specs/FEAT-demo-script/US.md` (không có SRS: story tài liệu). Sản phẩm kiểm: `docs/DEMO_SCRIPT.md`. Đối chiếu: `docs/FLOWS.md`, `docs/design/DESIGN.md` §13–§14, `docs/ARCHITECTURE.md` §7, §9.

**Chạy:** `bash docs/sprints/1/qc/scripts/tc-US-P0-01.sh TC-01 …` (không cần Docker; chỉ `git bash awk grep`). Mỗi AC có TC chạy nguyên văn lệnh của US.md **và** kiểm chặt hơn.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-01 | AC1 | – | `TC-01` (`grep -oE '^### Bước [0-9]+\..*— F[0-9]+' … \| grep -oE 'F[0-9]+$' \| xargs`) | `F2 F3 F5 F7 F9 F10 F17`; đúng 7 tiêu đề `### Bước`, số 1..7 liên tục |
| TC-02 | AC2 | – | `TC-02` (`awk -F'|' '… NF!=8'`; `grep -cE '^\| [0-9]{2}:[0-9]{2} \|'`) | Không hàng nào lệch 6 cột; ≥ 30 hàng; không ô trống; 7 bảng đúng 6 tiêu đề `Mốc, Vai trò, Route, Thao tác, Kết quả phải thấy, Phase`; mỗi bước ≥ 3 hàng |
| TC-03 | AC3 | – | `TC-03` (`sort -c`, `uniq -d`, cộng cột Thời lượng bảng mục 1) | Mốc tăng chặt; hàng cuối < 15:00; tổng thời lượng bảng mục 1 ≤ 15:00 và khớp "Hết kịch bản 13:45"; hàng của mỗi bước nằm trong khoảng ghi ở tiêu đề bước |
| TC-04 | AC4 | – | `TC-04` (vòng `grep -qF` từng route ở `DESIGN.md` và `ARCHITECTURE.md`) | Không `THIẾU`; mọi route nằm đúng trong DESIGN §14 hoặc ARCHITECTURE §7 (chặt hơn AC: không chỉ xuất hiện đâu đó trong file); vai trò chỉ gồm Admin / Giảng viên / Sinh viên / Người trình bày (Mailpit) |
| TC-05 | AC5 | – | `TC-05` (`grep -E '^\| [0-9:]+ \| Sinh viên' … \| grep -inE 'RAG\|PII\|…'`) | Không in gì; ≥ 5 hàng Sinh viên để kiểm; từ khoá mở rộng (ticket, AI, audit, SSE, điểm nháp…) liệt kê để xem tay; có cụm đúng DESIGN §13 "Cần giảng viên hỗ trợ", "Đã ẩn … thông tin cá nhân" |
| TC-06 | AC6 | – | `TC-06` (vòng 7 khoá `DEMO_MODE=true`, `fake`, `Video`, `Ai bật`, `.env.local`, `up -d gateway`, `lỗi 2 lần`) | Không `THIẾU`; cả 7 khoá nằm **trong mục 5**; có 3 tầng; lệnh bật tầng 2 khớp stack US-P0-02 (`--env-file .env.local -f docker-compose.local.yml -p edupilot`); ghi ai bật |
| TC-07 | AC7 (nhánh lỗi) | – | `TC-07` (`grep -c '^Nếu hội đồng hỏi'`; vòng `*.spec.ts` trong FLOWS) | ≥ 5; mọi spec có trong FLOWS; **từng** bước 1–6 (F2…F10) có "Nếu hội đồng hỏi" + tên spec E2E nằm đúng mục flow của bước đó trong FLOWS; mục 5 nêu: tầng 2 khi mạng/API LLM lỗi, tầng 3 khi stack không lên/tầng 2 lỗi, tầng 2 không cần mạng |
| TC-08 | AC8 (phân quyền) | – | `TC-08` (`grep -cE 'chỉ trả lời được…\|không có điểm nháp\|chỉ TEACHER\|Chốt điểm. bị khoá'`) | ≥ 4; từng cụm nằm trong **hàng bảng** đúng bước: bước 2 (hỏi người khác bị từ chối), bước 5 (không điểm nháp), bước 5/6 (chỉ TEACHER, `Chốt điểm` khoá); bước 6 có xác nhận công thức do Giảng viên làm |
| TC-09 | AC9 | – | `TC-09` | 8 hàng P1–P7,P10; 1 mục 7; mục 6 liệt kê đúng 8 phase; mục 7 có bảng 6 cột; mọi phase trong cột Phase có yêu cầu ở mục 6 và có `docs/phases/Px.md` |
| TC-10 | Chéo: dữ liệu cá nhân/secret | – | `TC-10` | Mọi email thuộc `@edupilot.local`; không chuỗi giống MSSV; không khoá thật; mật khẩu chỉ nêu bằng tên biến `SEED_DEFAULT_PASSWORD` |
| TC-11 | Chéo: tham chiếu file | – | `TC-11` | Mọi `docs/…`, `seed/…`, `scripts/…` được trích đều tồn tại; file chưa có phải là việc phase sau (mục 6) |
| TC-12 | Chéo: khớp seed + FLOWS | – | `TC-12` | Mã lớp, mã tham gia có ở tài liệu khác; FLOWS có mục F2, F3, F5, F7, F9, F10, F17 |
| TC-13 | Chéo: sạch tài liệu | – | `TC-13` | Không TODO/TBD/lorem; có dòng phiên bản; `QUESTIONS.md` tồn tại |
| TC-14 | Chéo: tài khoản seed | – | `TC-14` | Mọi tài khoản trong kịch bản có ở ARCHITECTURE §9, đúng vai Sinh viên B (`sv.kha`), C (`sv.nguyco`), D (`sv.moi`) |

## Nhánh lỗi
AC7 → TC-07 (đủ nhánh lỗi + spec E2E; quy tắc chuyển tầng ở mục 5).

## Phân quyền
AC8 → TC-08 (ranh giới quyền thể hiện bằng hàng chạy thật, đúng bước). Không có API/role thật để thử: story tài liệu.

## Kiểm chéo (QC.md mục 3–4)
Không áp dụng: idempotency, phân trang, migration, 375 px của màn mới, diff code (không có code). Áp dụng: lời văn sinh viên (TC-05), không PII/secret (TC-10), route không bịa (TC-04), khớp seed/FLOWS (TC-12, TC-14).

## Điểm khó kiểm
- AC4 chỉ chứng minh route tồn tại trong tài liệu, không chứng minh màn đó đúng hợp đồng bố cục (việc của các phase sau).
- AC5 chỉ có thể kiểm bằng danh sách từ cấm; câu chữ đúng/hay không thuộc về đọc tay (TC-05 in các dòng nghi vấn).
- AC3 "xem bằng mắt bảng mục 1": TC-03 tính tổng bằng awk thay cho việc cộng tay.

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
(chưa có — spec chưa đổi từ khi TC được viết)
