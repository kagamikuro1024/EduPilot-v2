# DEV handoff — US-P6-06 (`/gradebook`: sổ điểm thật, nhập điểm kiểu bảng tính, giải trình, lịch sử, công bố cột)
Nhánh `sprint/8-p6`. Spec: US-P6-06 AC1–AC12, SRS 4.6, TLR-12. Proposals: #8 (route xem thử / giải trình / lịch sử / công bố cột), #9 (dòng `want`), #12 (số thao tác OpenAPI: 232 → **238** với sáu route mới), #21 / #22 (ca cũ dùng `/gradebook` làm màn mô phỏng).

## Backend (`internal/grade`, `internal/httpapi/gradehttp`)
- `gradebook.go`: `Gradebook` (trang theo con trỏ `{f,i}`, cột từ công thức đã xác nhận, `entries`, điểm cộng / trừ, kết quả `Compute` của từng hàng, cảnh báo `PENALTY_NOT_IN_SCHEME` / `BONUS_NOT_IN_SCHEME` + `warning_students`), `PutEntry` (`ParseScore` ≤ 2 chữ số thập phân không làm tròn, khoá sổ `FOR SHARE`, so `version`, audit fail-closed, trả `row_result`), `PublishItem`, `Explain`, `History` (từ `audit_log`), `Preview` (tóm tắt toàn lớp ở trang đầu).
- Route: `GET …/gradebook`, `GET …/gradebook/preview`, `PUT …/grade-entries` (Teacher), `POST …/grade-items/{itemId}/publish` (Teacher), `GET …/gradebook/students/{uid}/explain|history` (Staff). 409 `VERSION_CONFLICT` kèm `details.current{score,version,updated_by{name},updated_at}`; 409 `GRADEBOOK_LOCKED`; 409 `SCHEME_NOT_CONFIRMED` (preview / explain trước lần xác nhận đầu); 422 `SCORE_OUT_OF_RANGE` / `SCORE_PRECISION`.
- `gradebook.sql` (sqlc), `openapi.yaml` (6 path + schema), `contract_test.go` (`want` + 238), `scenarios_grade_test.go` (mọi status khai báo).
- Kiểm: `TestPutEntry{OK,OutOfRange,Precision,Clear,ExamItemRejected,ReturnsRowResult,Under300ms,ConflictBody,Locked,AuditRow,AuditFailClosed}`, `TestEntryHistoryFromAudit`, `TestPublish*`, `TestPublishedColumnShownInGradebook`, `TestGradebookWarningsPenaltyNotInScheme`, `TestGradebookNoSchemeHasNoColumns`, `TestGradebookNoNPlusOne`, `TestGradebookPageP95`, `TestGradebookExplainNoSeqScan`, `TestPreviewUsesCompute`, `TestRBACMatrix`, `TestGradebookRBAC`, `TestPreviewRBAC`.

## Frontend
- `shared/ui/DataTable` + `DataTableVirtual`: `Column.cell` + `cellNav` (TLR-12): ô nhập trong bảng ảo hoá, Enter → xuống, Tab / Shift+Tab → ngang, Esc hoàn tác, mũi tên; chữ gõ dở nằm ngoài DOM (cuộn đi, cuộn về còn); `rowAttrs` cho bảng ảo hoá; tiêu đề cột nổi trên ô cố định (`z-index`).
- `shared/domain/GradeCalculation`: giải trình dạng phép tính, `Căn cứ · tr. N` mở đoạn trích tại chỗ, "do giảng viên đặt". Dùng ở `Gradebook` (Drawer `Giải trình`) và tab Điểm của `/students/[id]` (AC11).
- `Gradebook.tsx` (thay mock, `mock/gradebook.ts` đã xoá từ P6-04): một `Panel`; cột Sinh viên cố định; cột theo item (menu cột `Công bố cho sinh viên` / `Gỡ công bố`; cột bài thi chỉ đọc); Điểm cộng / trừ, Quá trình, Điểm học phần, Điểm chữ, Trạng thái; menu dòng `Giải trình` / `Lịch sử`. Lưu lạc quan từng ô, cột tính hiện `…` trong lúc chờ; hàng đợi từng ô; mất mạng giữ chữ + nhãn "Chưa lưu" + tự gửi lại; lỗi 422 ngay tại ô; 409 → popover "{tên} vừa nhập {điểm} · {n phút trước}" với `Giữ của tôi` / `Dùng của họ`. Công thức chưa xác nhận: banner + `Chốt điểm` khoá (lý do ở chính nút). Một dòng ghi chú cố định "Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường." TA chỉ đọc.
- `gradeApi.ts`: `useGradebook` (infinite query, 100 dòng / trang), `putEntry`, `publishItem`, `useExplain`, `useHistory`. Mã lỗi mới trong `ApiError` + `vi.ts`.
- Playwright `gradebook.spec.ts`: `gradebook layout`, `official note`, `cell edit keys`, `cell edit instant`, `cell conflict choose`, `explain student B`, `virtualized 1000`, `type then scroll keeps text`, `gradebook states`, `offline cell`, `gradebook rbac`, `publish column`, `penalty not in scheme notice`, `gradebook banner`.
- Ca cũ đổi theo proposal: `class-join.spec.ts` 137 / 148 → `/analytics` (#21); `tokens.spec.ts:39` giữ `/gradebook`, thêm `gradebookApi` (#22); `panels.spec` không đổi (mock `gradebookApi` trong `staffMock.ts`); `visual.spec.ts` thêm `gradebookApi` cho route `/gradebook` (#22).

## Ảnh mốc đổi (sinh trong `mcr.microsoft.com/playwright:v1.63.0-noble`, `visual.spec.ts` 14/14 pass sau đó)
| Ảnh | Lý do | Trước → sau |
| --- | --- | --- |
| `gradebook-1440.png` | `/gradebook` thành màn thật (bảng ảo hoá, cột theo công thức, ô nhập) | bảng mô phỏng BT01–BT03 + tiêu đề / mô tả → bảng sổ điểm thật dữ liệu giả cố định |
| `gradebook-390.png` | như trên | như trên, bề rộng 390 |

## Nằm ngoài story này
- Trạng thái `LOCKED` ở `GET gradebook`: chỉ dấu `locked{version}` + `Compute` sống; kết quả theo bản chốt và `drift` ở US-P6-08. Nút `Chốt điểm` chỉ có biến thể khoá; luồng chốt, `⋯` Nhập / Xuất / Mở khoá ở US-P6-07 / 08.
- Đo p95 lớp 1.000 và độ trễ trên stack: US-S8-SEED.
