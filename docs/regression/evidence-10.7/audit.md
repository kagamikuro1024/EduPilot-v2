# QC 10.7 — audit khung trang «sau» (móc `data-ep-*`)

**300** lượt chụp (dữ liệu 108, rỗng 87, đang tải 51, lỗi 54); lỗi chụp 10. Số đo từ DOM (`frontend/e2e/qc-ui-audit-real.spec.ts` v2), tiêu chí theo `techlead.md` mục 1.2. Ảnh có hai đường gióng xanh ở mép cột mong đợi. Lưới ảnh: `audit.html` (xếp theo kiểu trang). So với hiện trạng: `audit-truoc.md`.

- Dữ liệu = lớp 761987; rỗng = lớp QCEMPTY (sv31, teacher, ta). Đang tải = mọi `/api/v1/**` (trừ `auth`, `me`) giữ 5 s, chụp ở 0,8 s. Lỗi = 500. Đang tải / lỗi chỉ một vai mỗi route.
- Không đo: C1 (`contrast.spec.ts`), C2 diện tích đỏ (xem mục cuối nếu có PIL), P5 giữa các `Split`.

## Tổng hợp theo mã (số lượt lệch / số vai-route)

| Mã | Phép đo | Lệch | Vai + route (tối đa 8) |
| --- | --- | --- | --- |
| W1 | kiểu trang: đúng 1 `data-ep-page` = work|read | **8** | student/chat, student/threads, ta/analytics, ta/calendar, ta/class/members, ta/class/settings, ta/inbox, ta/settings |
| W2 | bề rộng cột 1440: work 1176; read 800 căn giữa (428→1228) | **0** | – |
| W3 | mép panel cấp cao nhất = mép cột (trừ Split) | **0** | – |
| W4 | mép trái h1 = mép trái cột; mép phải cụm nút = mép phải cột | **0** | – |
| W5 | 375: cột 12→363 (rộng 351), không tràn ngang | **0** | – |
| H1 | đúng 1 h1 trong header | **3** | student/chat, ta/calendar, ta/inbox |
| H2 | 0 thẻ `p` thừa ở đầu trang | **0** | – |
| H3 | meta ≤ 1 dòng, không chứa lớp đang chọn | **4** | teacher/observability |
| H4 | nút cùng hàng tiêu đề (|Δ tâm dọc| ≤ 4, 1440) | **0** | – |
| H5 | ≤ 1 primary, ≤ 4 nút ở đầu trang | **0** | – |
| H6 | 0 nút primary ngoài đầu trang khi ≤ 1 vùng | **0** | – |
| H7 | chiều cao đầu trang mỗi nhóm (độ rộng, cao h1, số dòng meta, có hàng nút, có hàng quay lại +36/+44; TL 12.1, 14.5) đúng 1 giá trị (±2) | **2** | teacher/gradebook |
| H8 | 375: hàng nút đầu trang ≤ 48 px | **0** | – |
| P1 | đệm panel 24 (16 ở 375) | **0** | – |
| P2 | panel cao theo nội dung (chênh ≤ 2 + dung sai đo dưới pixel 0,5; trừ Split fill) | **0** | – |
| P3 | 0 panel lồng | **0** | – |
| P4 | h2 có `data-ep-section-title`, 0 `p` ở đầu mục | **0** | – |
| P5 | nhịp dọc đầu trang→thân 40 (32@375), mục→mục 48, h2→panel 16 | **6** | admin/observability, student/threads, ta/analytics, ta/class/members, ta/class/settings, ta/settings |
| E1 | khối rỗng: 1 tiêu đề + ≤ 1 nút, 0 svg, 0 chữ khác | **0** | – |
| E2 | khối rỗng căn giữa |Δ| ≤ 2 px | **0** | – |
| E3 | tiêu đề rỗng ≤ 40 ký tự, 1 dòng, không dấu chấm cuối | **0** | – |
| E4 | nút rỗng không trùng nút đầu trang / tab / thanh công cụ | **0** | – |
| E5 | nút rỗng `secondary` | **0** | – |
| E6 | vật chứa khối rỗng ≤ 320 px (1440) | **0** | – |
| E7 | route ở lớp rỗng mà 0 `data-ep-empty` (xem ảnh) | **4** | student/chat, teacher/observability |
| L1 | đang tải: h1 cùng toạ độ như có dữ liệu (±2) | **1** | teacher/inbox |
| L2 | đang tải có khung xương `aria-busy` (≥ 1) | **1** | teacher/inbox |
| L3 | CLS ≤ 0,1 (có dữ liệu) | **3** | student/chat, student/practice/history, ta/documents |
| X1 | lỗi: ≤ 1 alert mỗi mục, trang không mục đúng 1, tổng ≤ số GET hỏng khác nhau, mỗi alert rộng = cột + «Thử lại» (TL 14.7) | **0** | – |
| X2 | lỗi: không mã trần (4xx/5xx/undefined/Error:) | **0** | – |
| X3 | lỗi và đang tải: giữ h1, nhãn tab, kiểu trang, nút đầu trang (cùng số, kiểu, thứ tự); meta và số đếm được mất (TL 14.2, 14.6) | **7** | student/practice, teacher/gradebook, teacher/inbox |
| D1 | cỡ chữ ⊆ {28,24,20,16,15,14,12} | **0** | – |
| D2 | cao điều khiển 36|30 (1440), ≥ 44 (375) | **0** | – |
| D3 | cao hàng bảng: ≥ 720 px 48|44 (±1); < 720 px là sàn (TL 14.8) | **0** | – |
| T1 | 0 icon giải thích (info / circle-help) ngoài thông báo | **0** | – |
| T2 | chuỗi > 30 ký tự lặp < 3 lần | **6** | admin/settings/llm, teacher/settings/llm |
| C3 | vàng: ≤ 3 nền, 0 chữ | **0** | – |

**Tổng lệch: 45** (mục chưa 0 = chưa đạt).

## Chữ ký khung (kiểu, h1.left, cột.left, cột.width, có meta, đệm panel, hình khối rỗng)

### 1440 px — **16** chữ ký (đạt ≤ 4: CHƯA)

| Số lượt | Chữ ký | Ví dụ (tối đa 4) |
| --- | --- | --- |
| 22 | ('work', 240, 240, 1176, False, 24, 'title') | student/threads[data], ta/gradebook[data], ta/gradebook/scheme[data], ta/grading[data] |
| 15 | ('work', 240, 240, 1176, True, 24, '-') | ta/[data], ta/students[data], teacher/students[data], teacher/attendance[data] |
| 13 | ('read', 428, 428, 800, False, 24, 'title') | student/practice[data], student/practice/history[data], student/exams[data], student/library[data] |
| 13 | ('work', 240, 240, 1176, False, 24, '-') | student/calendar[data], ta/class/members[data], teacher/grading[data], teacher/threads[data] |
| 13 | ('read', 428, 428, 800, False, 24, '-') | student/settings[data], ta/class/settings[data], ta/settings[data], teacher/exams[data] |
| 4 | ('read', 428, 428, 800, True, 24, 'title') | student/[data], ta/analytics[data], student/[empty], teacher/analytics[empty] |
| 4 | ('work', 240, 240, 1176, False, 24, 'title+nút') | ta/attendance[data], ta/attendance[empty], teacher/attendance[empty], teacher/gradebook[empty] |
| 4 | ('none', 500, -1, -1, False, 24, '-') | ta/analytics[empty], ta/class/members[empty], ta/class/settings[empty], ta/settings[empty] |
| 3 | ('none', -1, -1, -1, False, None, '-') | ta/inbox[data], student/chat[empty], ta/calendar[empty] |
| 3 | ('read', 428, 428, 800, False, 24, 'title+nút') | ta/insights[data], teacher/insights[data], teacher/insights[empty] |
| 1 | ('work', 240, 240, 1176, False, 0, 'title') | student/chat[data] |
| 1 | ('work', 240, 240, 1176, True, 24, 'title') | teacher/[data] |
| 1 | ('work', 240, 240, 1176, False, 0, '-') | teacher/inbox[data] |
| 1 | ('read', 428, 428, 800, True, 24, '-') | teacher/analytics[data] |
| 1 | ('read', 428, 428, 800, True, None, '-') | admin/[data] |
| 1 | ('read', 428, 428, 800, False, None, '-') | ta/inbox[empty] |

### 375 px — **14** chữ ký (đạt ≤ 2: CHƯA)

| Số lượt | Chữ ký | Ví dụ (tối đa 4) |
| --- | --- | --- |
| 22 | ('work', 12, 12, 351, False, 16, 'title') | ta/inbox[data], ta/gradebook[data], ta/gradebook/scheme[data], ta/questions[data] |
| 19 | ('read', 12, 12, 351, False, 16, '-') | student/practice[data], student/practice/history[data], student/exams[data], student/library[data] |
| 17 | ('work', 12, 12, 351, False, 16, '-') | student/chat[data], student/threads[data], student/calendar[data], ta/grading[data] |
| 14 | ('work', 12, 12, 351, True, 16, '-') | ta/[data], ta/students[data], teacher/students[data], teacher/attendance[data] |
| 4 | ('work', 12, 12, 351, False, 16, 'title+nút') | ta/attendance[data], ta/attendance[empty], teacher/attendance[empty], teacher/gradebook[empty] |
| 4 | ('read', 12, 12, 351, False, 16, 'title+nút') | ta/insights[data], teacher/insights[data], ta/insights[empty], teacher/insights[empty] |
| 4 | ('read', 12, 12, 351, True, 16, 'title') | ta/analytics[data], student/[empty], ta/analytics[empty], teacher/analytics[empty] |
| 3 | ('read', 12, 12, 351, False, 16, 'title') | ta/exams[data], ta/exams[empty], teacher/exams[empty] |
| 2 | ('read', 12, 12, 351, True, 16, '-') | student/[data], teacher/analytics[data] |
| 2 | ('work', 12, 12, 351, True, 16, 'title') | teacher/[data], ta/[empty] |
| 1 | ('work', 12, 12, 351, False, 0, '-') | teacher/inbox[data] |
| 1 | ('read', 12, 12, 351, True, None, '-') | admin/[data] |
| 1 | ('none', 12, -1, -1, False, 16, '-') | student/threads[empty] |
| 1 | ('read', 12, 12, 351, False, None, '-') | student/me[empty] |

## Chưa đo / không áp dụng (TL 14)

- **E7 chưa đo:** ta/analytics (empty@1440), ta/students (empty@1440/empty@375), teacher/students (empty@1440/empty@375) — lớp QCEMPTY vẫn có 1 sinh viên; `analytics` là màn mô phỏng (nợ sprint 11).
- **E7 không áp dụng (11 route):** student/settings, ta/ và teacher/ (Hôm nay), ta/ và teacher/ class/members, class/settings, ta/ và teacher/ settings, teacher/settings/llm, teacher/settings/integrations.

- **D3 mục > 112 px ở 375 (liệt kê để xem ảnh, không tính lệch; TL 14.8):** –

## Chi tiết lệch

### L3 — CLS ≤ 0,1 (có dữ liệu) (3)

- student/chat[data@1440] CLS 0.16164903376910256
- student/practice/history[data@375] CLS 0.3585884227353248
- ta/documents[data@375] CLS 0.11575646558043551

### W1 — kiểu trang: đúng 1 `data-ep-page` = work|read (8)

- ta/inbox[data@1440] (0, none)
- student/chat[empty@1440] (0, none)
- student/threads[empty@375] (0, none)
- ta/calendar[empty@1440] (0, none)
- ta/analytics[empty@1440] (0, none)
- ta/class/members[empty@1440] (0, none)
- ta/class/settings[empty@1440] (0, none)
- ta/settings[empty@1440] (0, none)

### H1 — đúng 1 h1 trong header (3)

- ta/inbox[data@1440] h1×0 trong header=False
- student/chat[empty@1440] h1×0 trong header=False
- ta/calendar[empty@1440] h1×0 trong header=False

### H3 — meta ≤ 1 dòng, không chứa lớp đang chọn (4)

- teacher/observability[data@1440] meta 1 dòng lặp lớp
- teacher/observability[data@375] meta 1 dòng lặp lớp
- teacher/observability[empty@1440] meta 1 dòng lặp lớp
- teacher/observability[empty@375] meta 1 dòng lặp lớp

### T2 — chuỗi > 30 ký tự lặp < 3 lần (6)

- teacher/settings/llm[data@1440] Máy chủ riêng (tương thích OpenAI)×7
- teacher/settings/llm[data@375] Máy chủ riêng (tương thích OpenAI)×7
- admin/settings/llm[data@1440] Máy chủ riêng (tương thích OpenAI)×7
- admin/settings/llm[data@375] Máy chủ riêng (tương thích OpenAI)×7
- teacher/settings/llm[empty@1440] Máy chủ riêng (tương thích OpenAI)×7
- teacher/settings/llm[empty@375] Máy chủ riêng (tương thích OpenAI)×7

### P5 — nhịp dọc đầu trang→thân 40 (32@375), mục→mục 48, h2→panel 16 (6)

- admin/observability[data@1440] mục→mục [24, 48, 48, 48]
- student/threads[empty@375] đầu trang→thân 48
- ta/analytics[empty@1440] đầu trang→thân 56
- ta/class/members[empty@1440] đầu trang→thân 56
- ta/class/settings[empty@1440] đầu trang→thân 56
- ta/settings[empty@1440] đầu trang→thân 56

### E7 — route ở lớp rỗng mà 0 `data-ep-empty` (xem ảnh) (4)

- student/chat[empty@1440] tự dựng (TL 14.3)
- student/chat[empty@375] tự dựng (TL 14.3)
- teacher/observability[empty@1440] tự dựng (TL 14.3)
- teacher/observability[empty@375] tự dựng (TL 14.3)

### H7 — chiều cao đầu trang mỗi nhóm (độ rộng, cao h1, số dòng meta, có hàng nút, có hàng quay lại +36/+44; TL 12.1, 14.5) đúng 1 giá trị (±2) (2)

- teacher/gradebook[data@1440] 66.4 ≠ 63 (w1440, h1 35px, meta 1 dòng, hàng nút=True, hàng quay lại=False)
- teacher/gradebook[data@375] 134.4 ≠ 131 (w375, h1 36px, meta 1 dòng, hàng nút=True, hàng quay lại=False)

### X3 — lỗi và đang tải: giữ h1, nhãn tab, kiểu trang, nút đầu trang (cùng số, kiểu, thứ tự); meta và số đếm được mất (TL 14.2, 14.6) (7)

- teacher/gradebook[loading@1440] nút ['Xem công thức', 'Thêm hành động', 'Chốt điểm'] vs ['Xem công thức', 'Thêm hành động']
- student/practice[loading@1440] tab ['Theo chủ đề0', 'Thi thử'] vs []
- teacher/inbox[loading@375] mất h1; kiểu none vs work
- teacher/gradebook[loading@375] nút [] vs ['Xem công thức', 'Thêm']
- teacher/gradebook[error@1440] nút ['Xem công thức', 'Thêm hành động', 'Chốt điểm'] vs ['Xem công thức', 'Thêm hành động']
- student/practice[error@1440] tab ['Theo chủ đề0', 'Thi thử'] vs []
- teacher/gradebook[error@375] nút ['Thêm', 'Chốt điểm'] vs ['Xem công thức', 'Thêm']

### L1 — đang tải: h1 cùng toạ độ như có dữ liệu (±2) (1)

- teacher/inbox[loading@375] không có h1 khi tải

### L2 — đang tải có khung xương `aria-busy` (≥ 1) (1)

- teacher/inbox[loading@375] không có aria-busy

## Lượt chụp lỗi

- student/practice[empty@375]: không có <main>
- student/practice/history[empty@375]: không có <main>
- student/exams[empty@375]: không có <main>
- student/library[empty@375]: không có <main>
- student/calendar[empty@375]: không có <main>
- student/settings[empty@375]: không có <main>
- ta/insights[empty@1440]: không có <main>
- teacher/grading[loading@1440]: không có <main>
- teacher/insights[loading@1440]: không có <main>
- admin/admin/users[loading@1440]: không có <main>
