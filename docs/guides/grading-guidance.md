# Hướng dẫn: học từ lần giảng viên sửa (D61, D62) — sprint 10.6

Dành cho giảng viên (dùng) và người vận hành (bật, đo, sự cố). Mỗi lớp bật riêng; mặc định **tắt**; tắt thì hệ thống chạy như trước 10.6. Sinh viên **không thấy gì** về tính năng này.

## 1. Bốn công tắc theo lớp
`/class/settings` → nhóm **Học từ lần sửa** (giảng viên; ADMIN thêm «Ca kiểm thử»). Cùng API: `GET|PUT /courses/{id}/guidance-flags[/{feature}]` (`GRADING_GUIDANCE`, `THREAD_SUGGEST`, `QUESTION_HINTS`, `REGRESSION_CAPTURE`). Công tắc cứng toàn hệ thống: `GUIDANCE_ENABLED=false` → API ghi trả `409 GUIDANCE_DISABLED`, không job nào chạy.
Ảnh chụp cờ nằm trong bộ nhớ gateway + worker; `PUT` ghi DB rồi `PUBLISH` Redis; mất tin thì tự đúng lại sau ≤ 60 s (P-106-6).

| Công tắc | Việc nó bật |
| --- | --- |
| Lưu ý chấm | M1: rút / duyệt / áp dụng lưu ý, chấm theo bản lưu ý, chấm lại bài nháp cũ |
| Ca kiểm thử | M2: ghi ca hồi quy (chỉ tham chiếu + số) khi giảng viên sửa điểm / phúc khảo / xử lý bài đăng |
| Gợi ý câu hỏi tương tự | M4: gợi ý ≤ 3 thread đã xác nhận khi sinh viên soạn câu hỏi |
| Gợi ý khi sinh câu hỏi | M6: lý do loại câu hỏi → khối «Tránh…» ở lần sinh câu sau |

## 2. Vòng M1 (ngăn bài tập, khối «Lưu ý chấm»)
1. Duyệt xong ≥ `GUIDANCE_MIN_RESULTS` bài, có ≥ `GUIDANCE_MIN_EDITS` lần sửa lệch ≥ `GUIDANCE_MIN_DELTA` → **Rút lưu ý** (job nền, ≤ `REFLECT_MAX_PER_HOUR` lần / giờ / bài tập). Đầu vào LLM chỉ có điểm AI, điểm giảng viên, nhận xét gốc của AI — không văn bản bài, không tên / MSSV.
2. Mỗi lưu ý là nháp («Chờ duyệt»): **Duyệt** / **Bỏ** / **Sửa** chữ; có «Hoàn tác» 5 s và **Đặt lại** tới khi Áp dụng. Chữ có điểm tổng, phạt, làm tròn, tên người bị từ chối.
3. **Áp dụng** (xác nhận, không hoàn tác) gom thành phiên bản N+1; KHÔNG tự chấm lại. Bài chấm sau đó ghim đúng bản lúc chấm.
4. Bài nháp chấm theo bản cũ chưa ai sửa → **Chấm lại** (`/grading`, ≤ `REGRADE_MAX_BATCH`). Không bao giờ chạm bài đã duyệt, đã sửa tay, đã công bố. Bài đã duyệt chấm bản cũ vẫn công bố được; hộp công bố nêu số bài («a đã sửa tay, b duyệt nguyên nháp»); lưới an toàn `409 MIXED_GUIDANCE` chỉ cho bài chưa từng duyệt.
5. Số đo: `GET …/guidance/stats` — MAE tiêu chí / tổng, lệch TB có dấu (dương = AI cao hơn giảng viên) theo phiên bản; n < 5 → ẩn. **Cho nghỉ** một lưu ý khi lệch TB đổi dấu quá tay.

## 3. Ca hồi quy và `cmd/regress` (M2, môi trường đo)
Ghi ca trong giao dịch nghiệp vụ khi cờ `REGRESSION_CAPTURE` bật (ADMIN). `expected` chỉ số / enum, chuỗi ≤ 64 ký tự, không văn bản. CLI vận hành, không trong image, CI không chạy:
```bash
cd backend-go
go run ./cmd/regress run --course <uuid> --write-baseline ../benchmarks/guidance/baseline.json   # lần đầu
go run ./cmd/regress run --course <uuid> --baseline ../benchmarks/guidance/baseline.json         # thoát 1 nếu «Có lùi»
go run ./cmd/regress run --course <uuid> --e2                                                   # ca THREAD_*: bài AI bị sửa / loại mà vẫn tự tin
```
Cần `DATABASE_URL`, `REDIS_URL`, cấu hình LLM (9router ở môi trường đo). «Có lùi» = MAE tăng quá `REGRESSION_MAX_REL` (0,10) so baseline, hoặc |lệch TB| xấu thêm > 0,20. `export` từ chối mặc định (`REGRESSION_EXPORT_ALLOWED=false`); khi cho phép chỉ ghi NGOÀI repo, đã che PII, id băm, có `audit_log`.

## 4. Demo và đo
```bash
SEED_GUIDANCE_DEMO=1 node scripts/seed.mjs && SEED_GUIDANCE_DEMO=1 node scripts/check-guidance-seed.mjs
bash benchmarks/guidance/run.sh            # chụp số đo theo phiên bản
bash benchmarks/guidance/run.sh --cycle    # Rút → Duyệt → Áp dụng → Chấm lại → chụp lại (cần provider sinh chữ)
```
Seed ghi thẳng DB (`gateway admin seed-guidance`, chỉ dev | test): bật cờ lớp 761987 + 8 kết quả BT03 đã duyệt, tiêu chí đầu giảng viên luôn thấp hơn AI 1,0. Không gọi Gemini thật.

## 5. «Hôm nay»
Giảng viên: «Lưu ý chấm chờ duyệt · <bài>». Giảng viên + TA: «N bài chấm theo bản cũ · <bài>». Chỉ lớp bật cờ; sự kiện `grading_guidance.changed` làm mới ngay.

## 6. Sự cố nhanh
| Triệu chứng | Xử lý |
| --- | --- |
| `Rút lưu ý` mờ | Chưa đủ bài / lần sửa (rê chuột nút xem số hiện có) hoặc cờ lớp tắt |
| Job rút lưu ý `FAILED` | Xem `jobs.error`; `REFLECT_FAILED` đã thử 3 lần → tin ở `ep:grading.reflect.dead` |
| Lệch TB đổi dấu sau khi áp dụng | `Cho nghỉ` lưu ý quá tay rồi `Áp dụng`; hoặc tắt «Lưu ý chấm» (chấm mới ghim v0) |
| Compose không nhận biến | `TestComposeGuidanceEnvReachesGatewayAndWorker` — thêm biến vào cả gateway và worker |
| Xoá sinh viên | Lưu ý và ca không chứa danh tính nên không cần dọn riêng (`TestGuidanceDeleteStudent`) |
