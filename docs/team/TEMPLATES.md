# Mẫu tài liệu

## US.md — User story

```markdown
# <FEAT-ID> <Tên feature>
Nguồn: PRD §<module>, FLOWS <Fx>, phase <Px> lát <Ly>

## US-<id>: <Vai trò> muốn <việc> để <lợi ích>
Ưu tiên: Must | Should | Could · Ước lượng: S | M | L · Sprint: N

### Tiêu chí nghiệm thu
- AC1. Given <bối cảnh> When <hành động> Then <kết quả quan sát được>
- AC2. ...
- ACn (nhánh lỗi). Given ... When ... Then ...

### Ngoài phạm vi của story này
- ...

### Phụ thuộc
- US-<id khác>, migration <số>, quyết định D<n>
```

## SRS.md — Đặc tả yêu cầu cho feature

```markdown
# SRS <FEAT-ID> <Tên feature>
Phiên bản · Ngày · Trạng thái: DRAFT | APPROVED

## 1. Mục đích và phạm vi
## 2. Người dùng và quyền (ma trận vai trò × hành động)
## 3. Luồng chính (sơ đồ mermaid) và các nhánh lỗi (bảng: tình huống → hệ thống phản ứng → người dùng thấy gì)
## 4. Yêu cầu chức năng (FR-1…: câu "Hệ thống phải…", mỗi FR trỏ về AC nào)
## 5. Dữ liệu: bảng/cột mới hoặc đổi, ràng buộc, index; migration số mấy
## 6. API: endpoint, quyền, request/response, mã lỗi; tool agent nếu có
## 7. Giao diện: route, khung nhìn đầu, primitive dùng, trạng thái tải/rỗng/lỗi, mobile; tham chiếu DESIGN.md §14.x
## 8. Phi chức năng áp dụng: hiệu năng, riêng tư, idempotency, phân trang, cache (trích từ SYSTEM_DESIGN/UX)
## 9. Kiểm thử: unit, tích hợp, E2E spec tên gì; dữ liệu seed cần gì
## 10. Câu hỏi mở (link QUESTIONS.md) và quyết định đã chốt
## 11. Truy vết: PRD → FLOWS → phase → US → FR → test
```

## QUESTIONS.md

```markdown
| # | Câu hỏi | Phương án BA đề xuất | Trả lời của chủ dự án | Ngày |
| --- | --- | --- | --- | --- |
```

## handoff/dev-<story>.md

```markdown
# DEV handoff — US-<id>
Nhánh / commit cuối: ...
## Đã làm (theo thứ tự lát dọc)
migration → store → service/handler → frontend → test → seed
## File đổi
## Lệnh QC chạy để kiểm
## Test đã chạy và kết quả
## AC tự đánh giá (AC1 ✓/✗ …)
## Nợ / chưa làm / cần hỏi
```

## qc/tc-<story>.md

```markdown
# QC test case — US-<id>
Nguồn: `docs/specs/<feature>/US.md` + `SRS.md`. Viết trước khi có code, không đọc code của dev.
| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-01 | AC1 | ... | ... | ... |
## Nhánh lỗi (mỗi tình huống ở SRS mục 3 một TC)
## Phân quyền (vai trò sai, sinh viên ngoài lớp → 403 / không thấy)
## Script chạy được: `frontend/e2e/**` hoặc `docs/sprints/N/qc/scripts/`
## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
```

## qc/report-<story>.md

```markdown
# QC report — US-<id>  · Kết luận: PASS | FAIL
## Cổng nghiệm thu đã chạy (lệnh → PASS/FAIL, 10 dòng đầu ra)
## AC (bảng: AC → cách kiểm → kết quả → ghi chú)
## Lỗi (BUG-n: mức, bước tái hiện, kỳ vọng, thực tế, file:dòng nghi ngờ)
## Kiểm phản mẫu UI / luật mở rộng / phân quyền (STUDENT gọi API giảng viên → 403?)
## Đề nghị
```

## sprints/N/report.md

```markdown
# Sprint N — báo cáo
Mục tiêu · Kết quả: X/Y story PASS
| Story | Feature | Trạng thái | QC | Ghi chú |
## Luồng end-to-end đã đi trọn (FLOWS F?)
## Số liệu: test xanh/đỏ, coverage nếu có, thời gian, chi phí LLM nếu đo được
## Việc nợ chuyển sang sprint kế
## Rủi ro và điều cần chủ dự án quyết
## Đề xuất sprint N+1
```
