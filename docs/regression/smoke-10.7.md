# Smoke hồi quy sprint 10.7 — kết quả
Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2 (SmkH) · Ngày: 2026-11-15
Tổng TC cũ: 86 · Đạt: 79 · Trượt: 7 · Không chạy (K): 0 · Chưa kết luận (N): 0   (P+F+K+N = 86)

Sprint 10.7 không có tệp `tc-*.md`; tập kiểm tự định nghĩa từ bộ kiểm của chính sprint (audit khung `qc-ui-audit-real`, thanh dính, trang giới thiệu 24 AC, tương phản, ui-antipatterns, e2e dev): 37 mã audit + 5 điểm dock + 24 AC landing + 1 tương phản + 2 antipatterns + 12 spec dev + 5 điểm riêng = 86. Route động không đo.

Cách chạy:
- Audit: `qc-ui-audit-real.spec.ts` (`playwright.qc-audit.config.ts`), 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi; lượt 1 (`audit/sau`) có 45 lệch khi chạy cùng bộ e2e nặng; chạy lại riêng từng vai không tải nền: `audit/sau2` (ta), `audit/sau3` (student, teacher, admin); kết luận theo lượt sau.
- `qc-dock-real`, `qc-docs-real`, `qc-landing-real`, `qc-settings-sv-real` (`real-r107.log`); `qc-r107-landing-extra.spec.ts` (AC12, AC16, AC22); copy-lint 0 FAIL.
- E2e dev ở bản dựng gate: `dev-e2e-107.log` (215 xanh, panels/class-join lượt 1 đỏ do thiếu `docs/sprints/1.5` trong bản sao; chạy lại riêng 59/59 và 77/77 xanh).
- `ui-antipatterns.sh` + `--selftest`.

## Tổng hợp
| Nhóm | Tổng | P | F | N |
|---|---|---|---|---|
| Audit khung giao diện (37 mã W/H/P/E/L/X/D/T/C) | 37 | 31 | 6 | 0 |
| Thanh dính đáy P-107-14 | 5 | 5 | 0 | 0 |
| Trang giới thiệu US-UX91-08 AC1–24 | 24 | 23 | 0 | 1 |
| Tương phản | 1 | 1 | 0 | 0 |
| ui-antipatterns | 2 | 2 | 0 | 0 |
| Bộ e2e dev (12 spec) | 12 | 12 | 0 | 0 |
| Điểm riêng | 5 | 4 | 1 | 0 |

## Kết quả từng ca
| Ca | KQ | Ghi chú |
|---|---|---|
| AUDIT-W1 | P | 0 lệch (audit lại 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi) |
| AUDIT-W2 | P | 0 lệch (audit lại 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi) |
| AUDIT-W3 | P | 0 lệch (audit lại 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi) |
| AUDIT-W4 | P | 0 lệch (audit lại 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi) |
| AUDIT-W5 | P | 0 lệch (audit lại 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi) |
| AUDIT-H1 | P | 0 lệch (audit lại 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi) |
| AUDIT-H2 | P | 0 lệch (audit lại 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi) |
| AUDIT-H3 | F | 4 lượt lệch: teacher/observability · DEF-R10.7-01 |
| AUDIT-H4 | P | 0 lệch (audit lại 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi) |
| AUDIT-H5 | P | 0 lệch (audit lại 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi) |
| AUDIT-H6 | P | 0 lệch (audit lại 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi) |
| AUDIT-H7 | F | 3 lượt lệch: ta/gradebook; teacher/gradebook · DEF-R10.7-02 |
| AUDIT-H8 | P | 0 lệch (audit lại 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi) |
| AUDIT-P1 | P | 0 lệch (audit lại 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi) |
| AUDIT-P2 | P | 0 lệch (audit lại 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi) |
| AUDIT-P3 | P | 0 lệch (audit lại 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi) |
| AUDIT-P4 | P | 0 lệch (audit lại 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi) |
| AUDIT-P5 | F | 1 lượt lệch: admin/observability · DEF-R10.7-01 |
| AUDIT-E1 | P | 0 lệch (audit lại 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi) |
| AUDIT-E2 | P | 0 lệch (audit lại 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi); student/chat 1 lệch cũ «không áp dụng» theo bản gốc |
| AUDIT-E3 | P | 0 lệch (audit lại 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi) |
| AUDIT-E4 | P | 0 lệch (audit lại 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi) |
| AUDIT-E5 | P | 0 lệch (audit lại 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi) |
| AUDIT-E6 | P | 0 lệch (audit lại 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi) |
| AUDIT-E7 | F | 2 lượt lệch: teacher/observability · DEF-R10.7-01 |
| AUDIT-L1 | P | 0 lệch (audit lại 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi) |
| AUDIT-L2 | P | 0 lệch (audit lại 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi) |
| AUDIT-L3 | P | 0 lệch (audit lại 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi) |
| AUDIT-X1 | P | 0 lệch (audit lại 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi) |
| AUDIT-X2 | P | 0 lệch (audit lại 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi) |
| AUDIT-X3 | F | 6 lượt lệch: student/practice, teacher/gradebook · DEF-R10.7-02/03 |
| AUDIT-D1 | P | 0 lệch (audit lại 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi) |
| AUDIT-D2 | P | 0 lệch (audit lại 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi) |
| AUDIT-D3 | P | 0 lệch (audit lại 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi) |
| AUDIT-T1 | P | 0 lệch (audit lại 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi) |
| AUDIT-T2 | F | 6 lượt lệch: admin/settings/llm, teacher/settings/llm · DEF-R10.7-04 |
| AUDIT-C3 | P | 0 lệch (audit lại 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi) |
| DOCK-1 dock có, nút chính trong dock, đầu trang không nút chính | P | P14-att375-dock:dock=true; nút chính trong dock=[{"t":"Đã lưu · 29 có mặt · ; P14-rev375-dock:dock=true; nút trong dock=[{"t":"Duyệt bài","h |
| DOCK-2 cố định, cách đáy = cao thanh dưới | P | P14-att375-sticky:position=fixed; cách đáy khung nhìn=56px = cao thanh điều hư; P14-rev375-sticky:position=fixed; cách đáy=56px (thanh dưới  |
| DOCK-3 nút ≥ 44 px | P | P14-att375-touch:cao nút 44; P14-rev375-touch:cao 44 |
| DOCK-4 sau cuộn không đổi | P | P14-att375-scroll:sau cuộn cách đáy 56px (thanh dưới 57px); P14-rev375-scroll:sau cuộn 56px (thanh dưới 57px) |
| DOCK-5 chữ cuối không bị che; 1440 một nút chính | P | P14-att375-nocover:cuối trang: đỉnh dock 695; số dòng chữ nằm sau dock 0; P14-att1440:đầu trang primary=["Đã lưu"]; dock=[] (không hai nút c |
| LANDING-AC1 | P | AC1:h1="Vận hành lớp học có trợ lý AI" (8 từ); mục: top=1vh, bai; AC1b:dòng dẫn nguyên văn |
| LANDING-AC2 | P | AC2:[{"t":"Đăng nhập","pause":false},{"t":"Xem video","pause":fa |
| LANDING-AC3 | P | AC3:demo aria-label=Màn duyệt bài: AI chấm nháp tiêu chí «Gi; im |
| LANDING-AC4 | P | AC4:nút 32×32; aria-pressed false→true; data-paused=?; AC4b:sau tải lại aria-pressed=true (nhớ trong phiên) |
| LANDING-AC5 | P | AC5:request video trước bấm=0; mở=true; focus trong hộp=true; Es |
| LANDING-AC6 | P | AC6:mục cao 1800px (2.00 màn); sticky=1; tên=Chat riêng có nguồn; AC6b:ảnh#0 top=358 ; ảnh#1 top=358 ; ảnh#2 top=358 |
| LANDING-AC7 | P | AC7:1024 px: sticky=0; AC7-390:sticky=0; ảnh=[{"w":347,"h":216,"vis":true},{"w":340,"h":212 |
| LANDING-AC8 | P | AC8:trước cuộn opacity=1,1,1,1,1,1; sau 2,6 s=1,1,1 |
| LANDING-AC9 | P | AC9:thiếu nguyên văn: []; 146: giữa=91 cuối=146; hình=[{"w":626, |
| LANDING-AC10 | P | AC10:tab=2; chọn true,false→false,true; mục mỗi vai=0,0; ảnh=gian; AC10-390:mục vai cao 844px (1.00 màn @844) |
| LANDING-AC11 | P | AC11:h1 72px; h2 48,48,48,48; p nhỏ nhất 16; AC11-390:h1 38px; h2 30,30,30,30 |
| LANDING-AC12 | P | AC12:nghỉ=none rê=matrix(1, 0, 0, 1, 0, -1) bấm=matrix(0.985, 0,  |
| LANDING-AC13 | P | AC13-lcp:LCP={"tag":"H1","id":"","t":72} (cần là chữ của h1; ghi chú: |
| LANDING-AC14 | P | AC14:m-on=0; animation chạy=0; sticky=0; số 146 hiện=146; nút tạm |
| LANDING-AC15 | P | AC15:không script: mục hiện=true; khối chữ opacity 0=0; danh sách |
| LANDING-AC16 | P | AC16-1440:axe 0; thứ bậc true; ảnh thiếu alt=0; AC16-390:axe 0; thứ bậc true; ảnh thiếu alt=0; AC16-kb:dialog mở 1→0; tạm dừng false→true; f |
| LANDING-AC17 | P | AC17:rò=false mô phỏng=true tên=true github=true |
| LANDING-AC18 | P | landing.spec dev (17 ca) xanh trên bản dựng gate |
| LANDING-AC19 | P | AC19:lỗi console: |
| LANDING-AC20 | P | landing.spec dev (17 ca) xanh trên bản dựng gate |
| LANDING-AC21 | P | landing.spec dev (17 ca) xanh trên bản dựng gate |
| LANDING-AC22 | P | màn đầu có một dữ kiện «146» và không «1.000 sinh viên» (landing-extra2.log; đối chiếu landing.spec dev) |
| LANDING-AC23 | P | RERUN 2026-11-16: lần trước lỗi chứng chỉ Caddy (ERR_CERT_AUTHORITY_INVALID) = môi trường; chạy lại trên FE http://localhost:3700 (`consistency.mjs --widths 1440,390 --rhythm-widths 1440,390 --page software`): 1440 = 8 FAIL, 390 = 3 FAIL (tổng 11) đều thuộc danh sách miễn trừ AC23 (scroll-reveal, scroll-linked, section-air, empty-area, copy-headings) — khớp báo cáo dev; copy-lint 0 FAIL (evidence-10.7/landing-consistency-r2.log) |
| LANDING-AC24 | P | AC24-1:mục số đo có 1 ảnh; AC24-390:mục số đo cao 1688px; khoảng trống liền giữa danh sách và ản |
| COLOR-contrast.spec (ma trận chữ × nền, vàng, sidebar, status) | P | dev-e2e-107.log: 4/4 |
| ANTI-ui-antipatterns.sh | P | 32 phép ✓, exit 0 |
| ANTI-ui-antipatterns --selftest | P | 32/32 phép bắt được |
| DEV-frame.spec (4 vai × mọi route) | P | dev-e2e-107.log |
| DEV-panels.spec | P | lượt 1 25 trượt do thiếu docs/sprints/1.5 trong bản sao; chạy lại 59/59 pass (dev-e2e-panels-2.log) |
| DEV-a11y.spec (axe) | P | dev-e2e-107.log |
| DEV-documents.spec | P | dev-e2e-107.log |
| DEV-landing.spec | P | dev-e2e-107.log |
| DEV-lcp.spec | P | dev-e2e-107.log |
| DEV-private-chat.spec | P | dev-e2e-107.log |
| DEV-decision-observability.spec | P | dev-e2e-107.log |
| DEV-class-join.spec | P | lượt 1 7 trượt cùng nguyên nhân; chạy lại 77/77 pass |
| DEV-settings-error.spec | P | dev-e2e-107.log |
| DEV-insights-course.spec | P | dev-e2e-107.log |
| DEV-me.spec | P | dev-e2e-107.log |
| MISC-documents: công tắc khoá aria-disabled, không icon, chạm mở lý do, Esc | P | 375: 3 công tắc aria-disabled, chạm mở / không đổi / chạm ra đóng; 1440 không có công tắc khoá ở lớp 761987 hiện tại (dữ liệu); dev documents.spec xanh |
| MISC-settings sinh viên lỗi API (qc-settings-sv-real) | P | spec xanh 1440 + 375; 4 GET hỏng riêng lẻ, mỗi khối có Thử lại (QCINFO trong real-r107.log) |
| MISC-observability: EmptyState «Chưa có số liệu» ở lớp rỗng | F | audit E7 + decision-observability.spec · DEF-R10.7-01 |
| MISC-/insights lớp rỗng: không mượn báo cáo lớp khác | P | insights-course.spec xanh (4/4) |
| MISC-/analytics lớp rỗng: «Chưa có số liệu» | P | audit E7 không lệch ở analytics |

## Trượt
Bốn lỗi, xem `defects-10.7.md`: DEF-R10.7-01 (observability), -02 (gradebook), -03 (student/practice khi tải), -04 (settings/llm lặp chữ).

## Chưa kết luận (N)
- Không còn. LANDING-AC23 đã chạy lại → đạt (xem dòng ca).

## Chạy lại sau lô sửa 9c346b86 (audit `sau4`, 2026-11-16, ep-r dựng lại từ 9c346b86)
Audit 4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi (`evidence-10.7/audit/sau4`, `audit-sau4.md`): 16 lệch (lượt `sau3`: ~24). Đã hết: DEF-R10.7-03 (practice mất tab khi tải), DEF-R10.7-04 (T2 lặp «Máy chủ riêng»), DEF-R10.7-01 phần meta lặp tên lớp + P5 nhịp 24≠48.
Còn lệch: **H7 gradebook** đầu trang 66,4 ≠ 63 (1440) và 134,4 ≠ 131 (375) — DEF-R10.7-02 chưa hết; **X3 gradebook** loading/error có «Chốt điểm» còn trạng thái dữ liệu thì không (lớp 761987 đã Chốt nên khung dữ liệu không có nút này — cần lớp chưa chốt để kết luận); **X3 teacher/insights** loading/error không có nút đầu trang («Tạo báo cáo mới», «Thêm hành động») trong khi trạng thái dữ liệu có — MỚI; **D3 gradebook** cao hàng [48, 66] — MỚI; **E6 teacher/observability[rỗng]** vật chứa 346,4 > 320 px; **E2 student/chat** lệch 190 (có sẵn từ sau3).
Các mục này chưa đổi trạng thái F của DEF-R10.7-01/-02 (còn mở một phần); -03 và -04 đã sửa.

Đo lại sổ điểm trên lớp CHƯA chốt, ít dữ liệu (QCEMPTY, GV `teacher`, `audit/sau5`, `audit-sau5.md`): H7 và D3 KHÔNG lệch (hai lệch trên 761987 phụ thuộc dữ liệu: lớp đã Chốt, nhiều cột/hàng); X3 VẪN lệch (1440): đang tải / lỗi hiện [«Xem công thức», «Thêm hành động», «Chốt điểm»] còn trạng thái dữ liệu chỉ có [«Xem công thức», «Chốt điểm»] — nút menu «Thêm hành động» ở khung chờ thừa so với khung dữ liệu của lớp ít dữ liệu. ⇒ X3 gradebook là lệch thật (giao dev2, P-11-51).

## Chạy lại sau P-11-51 (80e90d1b + 4 commit; ep-r dựng lại, 2026-11-16)
- **Lớp 761987 (đã chốt), audit đầy đủ `sau7`** (4 vai × 1440/375 × dữ liệu/rỗng/đang tải/lỗi): **4 lệch**, đều X3 gradebook (teacher, loading/error × 1440/375): khung chờ có [«Xem công thức», «Chốt điểm»] còn trạng thái dữ liệu của lớp đã chốt có [«Xem công thức», «Thêm hành động»]. HẾT: H7 và D3 gradebook, X3 insights, E6 observability rỗng, E2 student/chat.
- **Lớp QCEMPTY (chưa chốt, ít dữ liệu), audit `sau8`** (ta/teacher × gradebook/insights/observability × dữ liệu/đang tải/lỗi): **4 lệch**, đều X3 teacher/insights (loading/error × 1440/375): khung chờ có [«Tạo báo cáo mới», «Thêm hành động»] còn trạng thái dữ liệu của lớp chưa có báo cáo không có nút đầu trang. Gradebook trên QCEMPTY: 0 lệch.
- Nhận xét: các lệch còn lại phụ thuộc dữ liệu lớp (khung chờ không biết lớp đã chốt / đã có báo cáo) — mỗi lớp lệch ở một trang khác nhau; không còn lệch bố cục (H7, D3, E2, E6).
