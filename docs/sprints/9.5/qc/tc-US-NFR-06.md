# QC test case — US-NFR-06 Thử hỗn loạn nhỏ (giết worker, Redis 70 s, `docling-serve` 2 phút)
Nguồn: US-NFR-06 + SRS FR-11, §3, §6. Tải: chấm 200 bài + ingest 3 tài liệu + chat 1 QPS. QC kiểm `chaos.json` và **tự chạy lại ít nhất một tình huống** (đề xuất: `docling-serve` 2 phút, ít tốn CPU nhất; Q-N-17). Lệnh dừng/giết của QC luôn có `-p ep-qc`, không `docker kill <tên>` trần.

| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-N06-01 | AC4 | `benchmarks/reports/chaos.json` | `jq`; `node benchmarks/check-meta.mjs` | `meta` hợp lệ; 3 tình huống (kill worker, Redis 70 s, docling 2 phút); mỗi tình huống có bảng đối chiếu: tổng việc vào, việc xong, dead-letter (kèm lý do), thất lạc, chấm đôi | FILE |
| TC-N06-02 | AC4 | – | Tính lại: `vào == xong + dead-letter` mỗi tình huống | Đúng; thất lạc = 0; chấm đôi = 0 (đếm theo `submission_id`); dead-letter nào cũng có lý do | FILE |
| TC-N06-03 | AC1 | Stack `ep-qc`, tải chấm (200 bài) | Giữa hai lượt chấm: `docker compose -p ep-qc kill -s KILL worker` (hoặc `kill -9` trong container); đợi tối đa 10 phút; worker tự lên lại / `up -d worker` nếu spec cho phép | Sau ≤ 10 phút: mọi bài có **đúng một** kết quả: `SELECT submission_id, count(*) FROM <bảng kết quả chấm> GROUP BY 1 HAVING count(*) > 1` rỗng; 0 việc `PENDING` mồ côi (`grading.jobs`; định nghĩa «mồ côi»: Q-N-18); số bài có kết quả = 200 | RERUN |
| TC-N06-04 | AC2 | Tải chat 1 QPS + SSE thông báo | `docker compose -p ep-qc stop redis`; sau 70 s `start redis`; trong lúc đó gửi chat bằng SV | Trong lúc Redis tắt: phản hồi chat là lỗi **suy giảm có kiểm soát** (mã và thông điệp theo spec: «đang bận, thử lại»; Q-N-15), **không** `5xx` thô, **không** treo (mỗi lời gọi trả trong ≤ 10 s, ghi số thật) | RERUN |
| TC-N06-05 | AC2 | Client SSE đã mở trước khi dừng Redis, ghi `id:` mỗi sự kiện | Sau khi Redis lên: client nối lại bằng `Last-Event-ID` = id cuối đã nhận | Nối lại được; **không mất sự kiện** (chuỗi id liên tục, hoặc server phát lại phần thiếu); không sự kiện trùng ngoài phát lại có chủ ý | RERUN |
| TC-N06-06 | AC2 | Có lô chấm đang chạy khi Redis tắt | Sau khi Redis lên | Không lô chấm mồ côi (`grading.batch` mất ánh xạ lô — nợ sprint 8–9): nếu xảy ra → ghi **thành lỗi** (TL-n), không «dọn» | RERUN |
| TC-N06-07 | AC3 | Ingest 3 tài liệu đang chạy (tải lên mới) | `docker compose -p ep-qc stop docling` 2 phút rồi `start` | Job ingest: retry theo luật (3 lần) rồi dead-letter **hoặc** hoàn tất sau khi dịch vụ về; không tài liệu nào kẹt `PROCESSING` > 10 phút sau khi docling lên (`SELECT status, count(*) FROM documents`) | RERUN |
| TC-N06-08 | AC3 | Trong lúc docling tắt | Chat hỏi về tài liệu đã `READY` | Trả lời bình thường (không bị ảnh hưởng), `200`, có trích dẫn | RERUN |
| TC-N06-09 | AC4 | Sau mỗi tình huống ở TC-N06-03/04/07 | Đối chiếu: việc vào / xong / dead-letter do QC đếm trực tiếp từ DB + hàng đợi (`XLEN`, `XPENDING` của Redis Streams) | Khớp bảng của dev về cấu trúc: vào = xong + dead-letter; 0 thất lạc | RERUN |
| TC-N06-10 | AC5 | Nếu một tình huống FAIL (việc thất lạc / chấm đôi) | Đọc báo cáo + `techlead.md` | FAIL được ghi, chuyển US-NFR-10; **bằng chứng nguyên trạng** (chụp trước khi dọn: truy vấn + kết quả trong báo cáo) | FILE |
| TC-N06-11 | AC6 | repo | `grep -nE 'docker (kill|stop|rm)\b' benchmarks/**/*chaos* scripts/*chaos*`; đọc lệnh trong README | Mọi lệnh dừng/giết có `docker compose -p <project>` hoặc `$COMPOSE_PROJECT`; **không** `docker kill`/`stop` theo tên chung (`worker`, `redis`) | DIFF |
| TC-N06-12 | AC6 | Stack demo đang chạy (nếu có) | `docker ps --format '{{.ID}} {{.Names}} {{.Status}}'` (chỉ container `edupilot-*`) trước và sau TC-N06-03/04/07 | Không container demo nào bị dừng / khởi động lại (ID và uptime không đổi) | RERUN |
| TC-N06-13 | AC1 | – | Bỏ qua tải, kill worker **giữa** một lượt chấm: kiểm `kill -9` 3 lần liên tiếp | Mỗi lần: bài đúng một kết quả; không `FAILED` oan | RERUN |
| TC-N06-14 | SRS §3 | Redis tắt | `GET /api/v1/readyz` | Không `200` giả; báo không sẵn sàng đúng spec | RERUN |

## Phân quyền
Không có route mới. Kiểm chéo: khi Redis tắt, STUDENT vẫn không đọc được dữ liệu người khác (`403`/`404`; không lộ qua lỗi suy giảm).

## Lịch sử sửa TC
(trống)
