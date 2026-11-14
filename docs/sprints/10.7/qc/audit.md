# QC 10.7 — audit khung trang «sau» (móc `data-ep-*`)

**310** lượt chụp (dữ liệu 108, rỗng 94, đang tải 54, lỗi 54); lỗi chụp 0. Số đo từ DOM (`frontend/e2e/qc-ui-audit-real.spec.ts` v2), tiêu chí theo `techlead.md` mục 1.2. Ảnh có hai đường gióng xanh ở mép cột mong đợi. Lưới ảnh: `audit.html` (xếp theo kiểu trang). So với hiện trạng: `audit-truoc.md`.

- Dữ liệu = lớp 761987; rỗng = lớp QCEMPTY (sv31, teacher, ta). Đang tải = mọi `/api/v1/**` (trừ `auth`, `me`) giữ 5 s, chụp ở 0,8 s. Lỗi = 500. Đang tải / lỗi chỉ một vai mỗi route.
- Không đo: C1 (`contrast.spec.ts`), C2 diện tích đỏ (xem mục cuối nếu có PIL), P5 giữa các `Split`.

## Tổng hợp theo mã (số lượt lệch / số vai-route)

| Mã | Phép đo | Lệch | Vai + route (tối đa 8) |
| --- | --- | --- | --- |
| W1 | kiểu trang: đúng 1 `data-ep-page` = work|read | **0** | – |
| W2 | bề rộng cột 1440: work 1176; read 800 căn giữa (428→1228) | **0** | – |
| W3 | mép panel cấp cao nhất = mép cột (trừ Split) | **0** | – |
| W4 | mép trái h1 = mép trái cột; mép phải cụm nút = mép phải cột | **0** | – |
| W5 | 375: cột 12→363 (rộng 351), không tràn ngang | **0** | – |
| H1 | đúng 1 h1 trong header | **0** | – |
| H2 | 0 thẻ `p` thừa ở đầu trang | **0** | – |
| H3 | meta ≤ 1 dòng, không chứa lớp đang chọn | **12** | ta/analytics, teacher/analytics, teacher/observability |
| H4 | nút cùng hàng tiêu đề (|Δ tâm dọc| ≤ 4, 1440) | **0** | – |
| H5 | ≤ 1 primary, ≤ 4 nút ở đầu trang | **0** | – |
| H6 | 0 nút primary ngoài đầu trang khi ≤ 1 vùng | **14** | student/chat, student/practice/history, ta/, teacher/ |
| H7 | chiều cao đầu trang mỗi nhóm (độ rộng, cao h1, số dòng meta, có hàng nút, có hàng quay lại +36/+44; TL 12.1, 14.5) đúng 1 giá trị (±2) | **12** | student/practice/history, ta/gradebook/scheme, teacher/gradebook/scheme |
| H8 | 375: hàng nút đầu trang ≤ 48 px | **0** | – |
| P1 | đệm panel 24 (16 ở 375) | **0** | – |
| P2 | panel cao theo nội dung (chênh ≤ 2; trừ Split fill) | **13** | admin/observability, student/me, ta/analytics, ta/insights, teacher/analytics, teacher/insights |
| P3 | 0 panel lồng | **0** | – |
| P4 | h2 có `data-ep-section-title`, 0 `p` ở đầu mục | **0** | – |
| P5 | nhịp dọc đầu trang→thân 40 (32@375), mục→mục 48, h2→panel 16 | **2** | admin/observability |
| E1 | khối rỗng: 1 tiêu đề + ≤ 1 nút, 0 svg, 0 chữ khác | **0** | – |
| E2 | khối rỗng căn giữa |Δ| ≤ 2 px | **0** | – |
| E3 | tiêu đề rỗng ≤ 40 ký tự, 1 dòng, không dấu chấm cuối | **0** | – |
| E4 | nút rỗng không trùng nút đầu trang / tab / thanh công cụ | **0** | – |
| E5 | nút rỗng `secondary` | **2** | student/practice/history |
| E6 | vật chứa khối rỗng ≤ 320 px (1440) | **3** | admin/observability, ta/documents, teacher/documents |
| E7 | route ở lớp rỗng mà 0 `data-ep-empty` (xem ảnh) | **6** | student/, student/chat, teacher/observability |
| L1 | đang tải: h1 cùng toạ độ như có dữ liệu (±2) | **0** | – |
| L2 | đang tải có khung xương `aria-busy` (≥ 1) | **6** | teacher/analytics, teacher/observability, teacher/settings |
| L3 | CLS ≤ 0,1 (có dữ liệu) | **0** | – |
| X1 | lỗi: đúng 1 `role=alert`, rộng = cột, có «Thử lại» | **8** | student/chat, teacher/analytics, teacher/observability, teacher/settings |
| X2 | lỗi: không mã trần (4xx/5xx/undefined/Error:) | **0** | – |
| X3 | lỗi: giữ h1 + tab như có dữ liệu | **7** | student/practice, teacher/attendance, teacher/gradebook, teacher/gradebook/scheme |
| D1 | cỡ chữ ⊆ {28,24,20,16,15,14,12} | **8** | ta/class/settings, teacher/class/settings |
| D2 | cao điều khiển 36|30 (1440), ≥ 44 (375) | **11** | student/chat, ta/class/settings, teacher/class/settings, teacher/gradebook |
| D3 | cao hàng bảng 48|44 | **29** | admin/admin/courses, admin/admin/users, admin/observability, ta/attendance, ta/class/members, ta/documents, ta/gradebook, ta/grading … |
| T1 | 0 icon giải thích (info / circle-help) ngoài thông báo | **0** | – |
| T2 | chuỗi > 30 ký tự lặp < 3 lần | **4** | ta/documents, teacher/documents |
| C3 | vàng: ≤ 3 nền, 0 chữ | **0** | – |

**Tổng lệch: 137** (mục chưa 0 = chưa đạt).

## Chữ ký khung (kiểu, h1.left, cột.left, cột.width, có meta, đệm panel, hình khối rỗng)

### 1440 px — **11** chữ ký (đạt ≤ 4: CHƯA)

| Số lượt | Chữ ký | Ví dụ (tối đa 4) |
| --- | --- | --- |
| 21 | ('read', 428, 428, 800, False, 24, '-') | student/practice[data], student/practice/history[data], student/exams[data], student/library[data] |
| 20 | ('work', 240, 240, 1176, False, 24, '-') | student/threads[data], student/calendar[data], ta/gradebook[data], ta/grading[data] |
| 17 | ('work', 240, 240, 1176, True, 24, '-') | ta/[data], ta/students[data], ta/gradebook/scheme[data], ta/questions[data] |
| 17 | ('work', 240, 240, 1176, False, 24, 'title') | student/threads[empty], student/calendar[empty], ta/inbox[empty], ta/gradebook[empty] |
| 8 | ('read', 428, 428, 800, True, 24, '-') | student/[data], ta/insights[data], ta/analytics[data], teacher/insights[data] |
| 8 | ('read', 428, 428, 800, False, 24, 'title') | student/practice[empty], student/exams[empty], student/library[empty], student/me[empty] |
| 4 | ('work', 240, 240, 1176, False, 0, '-') | student/chat[data], ta/inbox[data], teacher/inbox[data], student/chat[empty] |
| 3 | ('work', 240, 240, 1176, False, 24, 'title+nút') | teacher/gradebook[empty], ta/attendance[empty], teacher/attendance[empty] |
| 1 | ('read', 428, 428, 800, True, 24, 'title') | admin/[data] |
| 1 | ('work', 240, 240, 1176, True, 24, 'title') | admin/observability[data] |
| 1 | ('read', 428, 428, 800, False, 24, 'title+nút') | student/practice/history[empty] |

### 375 px — **11** chữ ký (đạt ≤ 2: CHƯA)

| Số lượt | Chữ ký | Ví dụ (tối đa 4) |
| --- | --- | --- |
| 21 | ('read', 12, 12, 351, False, 16, '-') | student/practice[data], student/practice/history[data], student/exams[data], student/library[data] |
| 20 | ('work', 12, 12, 351, False, 16, '-') | student/threads[data], student/calendar[data], ta/gradebook[data], ta/grading[data] |
| 17 | ('work', 12, 12, 351, True, 16, '-') | ta/[data], ta/students[data], ta/gradebook/scheme[data], ta/questions[data] |
| 17 | ('work', 12, 12, 351, False, 16, 'title') | student/threads[empty], student/calendar[empty], ta/inbox[empty], ta/gradebook[empty] |
| 8 | ('read', 12, 12, 351, True, 16, '-') | student/[data], ta/insights[data], ta/analytics[data], teacher/insights[data] |
| 8 | ('read', 12, 12, 351, False, 16, 'title') | student/practice[empty], student/exams[empty], student/library[empty], student/me[empty] |
| 4 | ('work', 12, 12, 351, False, 0, '-') | student/chat[data], ta/inbox[data], teacher/inbox[data], student/chat[empty] |
| 3 | ('work', 12, 12, 351, False, 16, 'title+nút') | teacher/gradebook[empty], ta/attendance[empty], teacher/attendance[empty] |
| 1 | ('read', 12, 12, 351, True, 16, 'title') | admin/[data] |
| 1 | ('work', 12, 12, 351, True, 16, 'title') | admin/observability[data] |
| 1 | ('read', 12, 12, 351, False, 16, 'title+nút') | student/practice/history[empty] |

## Chưa đo / không áp dụng (TL 14)

- **E7 chưa đo:** ta/analytics (empty@1440/empty@375), ta/students (empty@1440/empty@375), teacher/analytics (empty@1440/empty@375), teacher/students (empty@1440/empty@375) — lớp QCEMPTY vẫn có 1 sinh viên; `analytics` là màn mô phỏng (nợ sprint 11).
- **E7 không áp dụng (11 route):** student/settings, ta/ và teacher/ (Hôm nay), ta/ và teacher/ class/members, class/settings, ta/ và teacher/ settings, teacher/settings/llm, teacher/settings/integrations.

## Chi tiết lệch

### H6 — 0 nút primary ngoài đầu trang khi ≤ 1 vùng (14)

- student/chat[data@1440] Gửi
- student/chat[data@375] Gửi
- ta/[data@1440] Mở
- ta/[data@375] Mở
- teacher/[data@1440] Mở
- teacher/[data@375] Mở
- student/chat[empty@1440] Gửi
- student/practice/history[empty@1440] Luyện đề
- student/chat[empty@375] Gửi
- student/practice/history[empty@375] Luyện đề
- ta/[empty@1440] Mở
- ta/[empty@375] Mở
- teacher/[empty@1440] Mở
- teacher/[empty@375] Mở

### D2 — cao điều khiển 36|30 (1440), ≥ 44 (375) (11)

- student/chat[data@1440] [32]
- ta/class/settings[data@1440] [16]
- ta/class/settings[data@375] [16]
- teacher/gradebook[data@1440] [32]
- teacher/class/settings[data@1440] [16]
- teacher/class/settings[data@375] [16]
- student/chat[empty@1440] [32]
- ta/class/settings[empty@1440] [16]
- ta/class/settings[empty@375] [16]
- teacher/class/settings[empty@1440] [16]
- teacher/class/settings[empty@375] [16]

### P2 — panel cao theo nội dung (chênh ≤ 2; trừ Split fill) (13)

- student/me[data@1440] +17
- student/me[data@375] +17
- ta/analytics[data@1440] +17
- ta/insights[data@375] +2.1
- ta/analytics[data@375] +17
- teacher/analytics[data@1440] +17
- teacher/insights[data@375] +2.1
- teacher/analytics[data@375] +17
- admin/observability[data@1440] +2.1
- ta/analytics[empty@1440] +17
- ta/analytics[empty@375] +17
- teacher/analytics[empty@1440] +17
- teacher/analytics[empty@375] +17

### D3 — cao hàng bảng 48|44 (29)

- ta/students[data@1440] [58, 59, 480]
- ta/gradebook[data@1440] [60, 61, 432]
- ta/grading[data@1440] [61, 62]
- ta/questions[data@1440] [48, 59, 62]
- ta/documents[data@1440] [89, 62]
- ta/class/members[data@1440] [56, 57]
- ta/students[data@375] [103, 104, 81, 84, 480]
- ta/gradebook[data@375] [105, 83, 528]
- teacher/students[data@1440] [58, 59, 480]
- teacher/gradebook[data@1440] [60, 61, 83, 432]
- teacher/grading[data@1440] [61, 62]
- teacher/questions[data@1440] [48, 59, 62]
- teacher/documents[data@1440] [89, 62]
- teacher/class/members[data@1440] [56, 57]
- teacher/observability[data@1440] [44, 61, 62]
- teacher/students[data@375] [103, 104, 81, 84, 480]
- teacher/gradebook[data@375] [105, 83, 528]
- admin/observability[data@1440] [61, 62, 44]
- admin/admin/courses[data@1440] [56, 57, 58]
- admin/admin/users[data@1440] [49]
- admin/observability[data@375] [106, 107, 84]
- ta/attendance[data@1440] [58, 59]
- teacher/attendance[data@1440] [58, 59]
- ta/students[empty@1440] [58]
- ta/class/members[empty@1440] [56, 57]
- ta/students[empty@375] [81]
- teacher/students[empty@1440] [58]
- teacher/class/members[empty@1440] [56, 57]
- teacher/students[empty@375] [81]

### T2 — chuỗi > 30 ký tự lặp < 3 lần (4)

- ta/documents[data@1440] Không dùng cho AI của sinh viên×4
- ta/documents[data@375] Không dùng cho AI của sinh viên×4
- teacher/documents[data@1440] Không dùng cho AI của sinh viên×4
- teacher/documents[data@375] Không dùng cho AI của sinh viên×4

### H3 — meta ≤ 1 dòng, không chứa lớp đang chọn (12)

- ta/analytics[data@1440] meta 1 dòng lặp lớp
- ta/analytics[data@375] meta 1 dòng lặp lớp
- teacher/analytics[data@1440] meta 1 dòng lặp lớp
- teacher/observability[data@1440] meta 1 dòng lặp lớp
- teacher/analytics[data@375] meta 1 dòng lặp lớp
- teacher/observability[data@375] meta 1 dòng lặp lớp
- ta/analytics[empty@1440] meta 1 dòng lặp lớp
- ta/analytics[empty@375] meta 1 dòng lặp lớp
- teacher/analytics[empty@1440] meta 1 dòng lặp lớp
- teacher/observability[empty@1440] meta 1 dòng lặp lớp
- teacher/analytics[empty@375] meta 1 dòng lặp lớp
- teacher/observability[empty@375] meta 1 dòng lặp lớp

### D1 — cỡ chữ ⊆ {28,24,20,16,15,14,12} (8)

- ta/class/settings[data@1440] [52]
- ta/class/settings[data@375] [32.2]
- teacher/class/settings[data@1440] [52]
- teacher/class/settings[data@375] [32.2]
- ta/class/settings[empty@1440] [52]
- ta/class/settings[empty@375] [32.2]
- teacher/class/settings[empty@1440] [52]
- teacher/class/settings[empty@375] [32.2]

### P5 — nhịp dọc đầu trang→thân 40 (32@375), mục→mục 48, h2→panel 16 (2)

- admin/observability[data@1440] mục→mục [60.8, 48]
- admin/observability[data@375] mục→mục [127.9, 48]

### E6 — vật chứa khối rỗng ≤ 320 px (1440) (3)

- admin/observability[data@1440] 377.1
- ta/documents[empty@1440] 454.4
- teacher/documents[empty@1440] 454.4

### E7 — route ở lớp rỗng mà 0 `data-ep-empty` (xem ảnh) (6)

- student/[empty@1440] tự dựng (TL 14.3)
- student/chat[empty@1440] tự dựng (TL 14.3)
- student/[empty@375] tự dựng (TL 14.3)
- student/chat[empty@375] tự dựng (TL 14.3)
- teacher/observability[empty@1440] tự dựng (TL 14.3)
- teacher/observability[empty@375] tự dựng (TL 14.3)

### E5 — nút rỗng `secondary` (2)

- student/practice/history[empty@1440] primary
- student/practice/history[empty@375] primary

### H7 — chiều cao đầu trang mỗi nhóm (độ rộng, cao h1, số dòng meta, có hàng nút, có hàng quay lại +36/+44; TL 12.1, 14.5) đúng 1 giá trị (±2) (12)

- ta/gradebook/scheme[data@1440] 102.4 ≠ 63 (w1440, h1 35px, meta 1 dòng, hàng nút=False, hàng quay lại=False)
- student/practice/history[data@1440] 72 ≠ 36 (w1440, h1 35px, meta 0 dòng, hàng nút=False, hàng quay lại=False)
- student/practice/history[empty@1440] 72 ≠ 36 (w1440, h1 35px, meta 0 dòng, hàng nút=False, hàng quay lại=False)
- ta/gradebook/scheme[empty@1440] 72 ≠ 36 (w1440, h1 35px, meta 0 dòng, hàng nút=False, hàng quay lại=False)
- teacher/gradebook/scheme[empty@1440] 72 ≠ 36 (w1440, h1 35px, meta 0 dòng, hàng nút=True, hàng quay lại=False)
- ta/gradebook/scheme[data@375] 110.4 ≠ 63 (w375, h1 36px, meta 1 dòng, hàng nút=False, hàng quay lại=False)
- student/practice/history[data@375] 80 ≠ 36 (w375, h1 36px, meta 0 dòng, hàng nút=False, hàng quay lại=False)
- student/practice/history[empty@375] 80 ≠ 36 (w375, h1 36px, meta 0 dòng, hàng nút=False, hàng quay lại=False)
- ta/gradebook/scheme[empty@375] 80 ≠ 36 (w375, h1 36px, meta 0 dòng, hàng nút=False, hàng quay lại=False)
- teacher/gradebook/scheme[empty@375] 148 ≠ 104 (w375, h1 36px, meta 0 dòng, hàng nút=True, hàng quay lại=False)
- teacher/gradebook/scheme[data@1440] 102.4 ≠ 63 (w1440, h1 35px, meta 1 dòng, hàng nút=True, hàng quay lại=False)
- teacher/gradebook/scheme[data@375] 178.4 ≠ 131 (w375, h1 36px, meta 1 dòng, hàng nút=True, hàng quay lại=False)

### L2 — đang tải có khung xương `aria-busy` (≥ 1) (6)

- teacher/settings[loading@1440] không có aria-busy
- teacher/analytics[loading@1440] không có aria-busy
- teacher/observability[loading@1440] không có aria-busy
- teacher/settings[loading@375] không có aria-busy
- teacher/analytics[loading@375] không có aria-busy
- teacher/observability[loading@375] không có aria-busy

### X1 — lỗi: đúng 1 `role=alert`, rộng = cột, có «Thử lại» (8)

- teacher/settings[error@1440] alert×0
- teacher/analytics[error@1440] alert×0
- teacher/observability[error@1440] alert×0
- student/chat[error@1440] alert×0
- teacher/settings[error@375] alert×0
- teacher/analytics[error@375] alert×0
- teacher/observability[error@375] alert×0
- student/chat[error@375] alert×0

### X3 — lỗi: giữ h1 + tab như có dữ liệu (7)

- teacher/attendance[error@1440] nút [] vs ['Lưu điểm danh']
- teacher/gradebook[error@1440] nút [] vs ['Chốt điểm', 'Thêm hành động', 'Xem công thức']
- teacher/gradebook/scheme[error@1440] nút [] vs ['Tải quy chế']
- student/practice[error@1440] tab [] vs ['Theo chủ đề13', 'Thi thử']
- teacher/gradebook[error@375] nút [] vs ['Chốt điểm', 'Thêm']
- teacher/gradebook/scheme[error@375] nút [] vs ['Tải quy chế']
- student/practice[error@375] tab [] vs ['Theo chủ đề13', 'Thi thử']

