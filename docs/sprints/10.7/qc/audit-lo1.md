# QC 10.7 — audit khung trang «lo1» (móc `data-ep-*`)

**36** lượt chụp (dữ liệu 12, rỗng 12, đang tải 6, lỗi 6); lỗi chụp 0. Số đo từ DOM (`frontend/e2e/qc-ui-audit-real.spec.ts` v2), tiêu chí theo `techlead.md` mục 1.2. Ảnh có hai đường gióng xanh ở mép cột mong đợi. Lưới ảnh: `audit.html` (xếp theo kiểu trang). So với hiện trạng: `audit-truoc.md`.

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
| H3 | meta ≤ 1 dòng, không chứa lớp đang chọn | **0** | – |
| H4 | nút cùng hàng tiêu đề (|Δ tâm dọc| ≤ 4, 1440) | **0** | – |
| H5 | ≤ 1 primary, ≤ 4 nút ở đầu trang | **0** | – |
| H6 | 0 nút primary ngoài đầu trang khi ≤ 1 vùng | **1** | ta/inbox |
| H7 | chiều cao đầu trang mỗi nhóm (độ rộng, cao h1, số dòng meta, có hàng nút; TL mục 12, PM 15/11) đúng 1 giá trị (±2) | **0** | – |
| H8 | 375: hàng nút đầu trang ≤ 48 px | **0** | – |
| P1 | đệm panel 24 (16 ở 375) | **0** | – |
| P2 | panel cao theo nội dung (chênh ≤ 2; trừ Split fill) | **2** | ta/insights, teacher/insights |
| P3 | 0 panel lồng | **0** | – |
| P4 | h2 có `data-ep-section-title`, 0 `p` ở đầu mục | **8** | ta/questions, teacher/questions |
| P5 | nhịp dọc đầu trang→thân 40 (32@375), mục→mục 48, h2→panel 16 | **3** | ta/insights, teacher/insights |
| E1 | khối rỗng: 1 tiêu đề + ≤ 1 nút, 0 svg, 0 chữ khác | **0** | – |
| E2 | khối rỗng căn giữa |Δ| ≤ 2 px | **0** | – |
| E3 | tiêu đề rỗng ≤ 40 ký tự, 1 dòng, không dấu chấm cuối | **0** | – |
| E4 | nút rỗng không trùng nút đầu trang / tab / thanh công cụ | **0** | – |
| E5 | nút rỗng `secondary` | **0** | – |
| E6 | vật chứa khối rỗng ≤ 320 px (1440) | **0** | – |
| E7 | route ở lớp rỗng mà 0 `data-ep-empty` (xem ảnh) | **0** | – |
| L1 | đang tải: h1 cùng toạ độ như có dữ liệu (±2) | **1** | teacher/inbox |
| L2 | đang tải có khung xương `aria-busy` (≥ 1) | **1** | teacher/inbox |
| L3 | CLS ≤ 0,1 (có dữ liệu) | **0** | – |
| X1 | lỗi: đúng 1 `role=alert`, rộng = cột, có «Thử lại» | **2** | teacher/questions |
| X2 | lỗi: không mã trần (4xx/5xx/undefined/Error:) | **0** | – |
| X3 | lỗi: giữ h1 + tab như có dữ liệu | **2** | teacher/insights, teacher/questions |
| D1 | cỡ chữ ⊆ {28,24,20,16,15,14,12} | **0** | – |
| D2 | cao điều khiển 36|30 (1440), ≥ 44 (375) | **5** | ta/inbox, ta/questions, teacher/questions |
| D3 | cao hàng bảng 48|44 | **1** | ta/questions |
| T1 | 0 icon giải thích (info / circle-help) ngoài thông báo | **0** | – |
| T2 | chuỗi > 30 ký tự lặp < 3 lần | **0** | – |
| C3 | vàng: ≤ 3 nền, 0 chữ | **0** | – |

**Tổng lệch: 26** (mục chưa 0 = chưa đạt).

## Chữ ký khung (kiểu, h1.left, cột.left, cột.width, có meta, đệm panel, hình khối rỗng)

### 1440 px — **5** chữ ký (đạt ≤ 4: CHƯA)

| Số lượt | Chữ ký | Ví dụ (tối đa 4) |
| --- | --- | --- |
| 6 | ('work', 240, 240, 1176, False, 24, 'title') | teacher/inbox[data], teacher/questions[data], ta/inbox[empty], ta/questions[empty] |
| 3 | ('read', 428, 428, 800, False, 24, 'title') | teacher/insights[data], ta/insights[empty], teacher/insights[empty] |
| 1 | ('work', 240, 240, 1176, False, 0, '-') | ta/inbox[data] |
| 1 | ('work', 240, 240, 1176, True, 24, '-') | ta/questions[data] |
| 1 | ('read', 428, 428, 800, True, 24, '-') | ta/insights[data] |

### 375 px — **5** chữ ký (đạt ≤ 2: CHƯA)

| Số lượt | Chữ ký | Ví dụ (tối đa 4) |
| --- | --- | --- |
| 4 | ('work', 12, 12, 351, False, 16, 'title') | ta/inbox[empty], ta/questions[empty], teacher/inbox[empty], teacher/questions[empty] |
| 2 | ('work', 12, 12, 351, False, 0, '-') | ta/inbox[data], teacher/inbox[data] |
| 2 | ('work', 12, 12, 351, True, 16, '-') | ta/questions[data], teacher/questions[data] |
| 2 | ('read', 12, 12, 351, True, 16, '-') | ta/insights[data], teacher/insights[data] |
| 2 | ('read', 12, 12, 351, False, 16, 'title') | ta/insights[empty], teacher/insights[empty] |

## Chi tiết lệch

### H6 — 0 nút primary ngoài đầu trang khi ≤ 1 vùng (1)

- ta/inbox[data@1440] Nhận

### D2 — cao điều khiển 36|30 (1440), ≥ 44 (375) (5)

- ta/inbox[data@1440] [28]
- ta/questions[data@1440] [39]
- teacher/questions[data@1440] [39]
- ta/questions[empty@1440] [39]
- teacher/questions[empty@1440] [39]

### P4 — h2 có `data-ep-section-title`, 0 `p` ở đầu mục (8)

- ta/questions[data@1440] h2 thiếu móc 1, p 0
- ta/questions[data@375] h2 thiếu móc 1, p 0
- teacher/questions[data@1440] h2 thiếu móc 1, p 0
- teacher/questions[data@375] h2 thiếu móc 1, p 0
- ta/questions[empty@1440] h2 thiếu móc 1, p 0
- ta/questions[empty@375] h2 thiếu móc 1, p 0
- teacher/questions[empty@1440] h2 thiếu móc 1, p 0
- teacher/questions[empty@375] h2 thiếu móc 1, p 0

### D3 — cao hàng bảng 48|44 (1)

- ta/questions[data@1440] [61, 82]

### P5 — nhịp dọc đầu trang→thân 40 (32@375), mục→mục 48, h2→panel 16 (3)

- ta/insights[data@1440] mục→mục [80]
- ta/insights[data@375] mục→mục [80]
- teacher/insights[data@375] mục→mục [80]

### P2 — panel cao theo nội dung (chênh ≤ 2; trừ Split fill) (2)

- ta/insights[data@375] +2.1
- teacher/insights[data@375] +2.1

### L1 — đang tải: h1 cùng toạ độ như có dữ liệu (±2) (1)

- teacher/inbox[loading@1440] không có h1 khi tải

### L2 — đang tải có khung xương `aria-busy` (≥ 1) (1)

- teacher/inbox[loading@1440] không có aria-busy

### X1 — lỗi: đúng 1 `role=alert`, rộng = cột, có «Thử lại» (2)

- teacher/questions[error@1440] alert×1 rộng 1126
- teacher/questions[error@375] alert×1 rộng 317

### X3 — lỗi: giữ h1 + tab như có dữ liệu (2)

- teacher/insights[error@1440] h1×1 tab [] vs []
- teacher/questions[error@375] h1×1 tab [] vs []

