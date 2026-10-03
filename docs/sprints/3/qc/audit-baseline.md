# Audit baseline — nền giao diện trước PU (QC, sprint 3)

Điều kiện của FEAT-ui-foundation Q10 / US-PU-01 AC9 / US-PU-05 AC2 + AC10: dev **không gộp story PU đầu tiên** khi tệp này chưa có. Mọi số dưới đây **đo thật** (2026-10-03), không chép từ báo cáo cũ.

## Điều kiện đo
| Mục | Giá trị |
| --- | --- |
| Commit gốc | `307bfd2` ("sprint 3: kế hoạch PU + P1, D53…") — worktree riêng, `git log -1` đúng commit này |
| Bản dựng | `pnpm install --frozen-lockfile` → `pnpm -C frontend build` **rc=0**, 0 dòng cảnh báo / `Failed to load font`; `next start -p 3400` |
| Trình duyệt | Chrome for Testing headless riêng (`--headless=new`), chuột thật, `caffeinate -d -i` |
| Công cụ | `docs/sprints/1.5/qc/scripts/{audit.mjs, sweep.mjs, proto-curl.sh}` (bản của nhánh này; sửa nhỏ nêu ở "Sửa công cụ") |
| So với hiện tại | `git diff 307bfd2 HEAD -- frontend` chỉ có 3 tệp lịch (fix BUG-v5-01-6, `a4a8184`) — **chưa có story PU nào** |

## Số liệu nền (so sánh sau mỗi story PU)
Cách chạy: Eval (JS), `const m = await import('<abs>/docs/sprints/1.5/qc/scripts/audit.mjs'); await m.default(browser, { base:'http://localhost:3400', out, only:'student' })` (lặp `teacher`, `ta`, `admin`); phần "spec + không vai" chạy với `routesOnly:['/__khong-co__'], skipScenarios:true, skipEdge:true`. Mỗi lệnh phải xong ≤ 280 s nên tách theo vai.

| Lượt chạy `audit.mjs` | Số hàng | PASS | FAIL | Ghi chú |
| --- | --- | --- | --- | --- |
| `only:'student'` | 165 | 165 | **0** | 15 route; bề rộng 1440/390/375 + biên 1100/1024/720/719; 45 hàng `AUDIT` |
| `only:'teacher'` | 170 | 170 | **0** | 21 route; 44 hàng `AUDIT` |
| `only:'ta'` | 106 | 106 | **0** | 17 route; 36 hàng `AUDIT` |
| `only:'admin'` | 54 | 54 | **0** | 7 route; 14 hàng `AUDIT` |
| spec + không vai (`routesOnly` giả) | 185 | 185 | **0** | 00-AC7…13, 01-AC11/17/19/21/22/27, 02-AC9/10/14/15/18/19/21, 03-AC7, 04-AC7/8/11/12 + `/login` × 3 bề rộng |
| **Tổng** | **680** | **680** | **0** | `AUDIT-biên` (48 cặp vai×route×bề rộng) nằm trong các hàng trên |

**Điều kiện cổng (AC10):** sau mỗi story PU, từng lượt trên cho `FAIL 0` và **số hàng PASS ≥ số ở bảng** (route mới như `/dev/ui` chỉ làm tăng). Danh sách hàng nền: `audit-baseline.json` (role, route, bề rộng, phép đo, ok).

Các phép đo khác ở cùng commit:

| Cổng | Kết quả nền |
| --- | --- |
| `pnpm -C frontend lint` | rc=0 |
| `bash scripts/ui-antipatterns.sh` | rc=0; **11** dòng ✓ (màu cứng, xám Tailwind, bo 16–24, bóng, cỡ chữ, `fetch` trần, spinner toàn trang, hiệu ứng cấm, gamification, khung vỏ đặc, dialog bảo vệ) |
| `proto-curl.sh all` (`F=http://localhost:3400`) | **497 PASS / 0 FAIL** (gồm `tc_00_matrix`: 128 hàng ma trận vai × route) |
| `sweep.mjs only:'student'` | 42 hàng (14 route × 3 bề rộng): **0 `FORBIDDEN`** (từ kỹ thuật, `[[SV_`), 0 cuộn ngang, 0 HTTP lỗi, 0 lỗi console, mọi trang đúng 1 `h1`, 0 phần tử bị cắt |
| Danh sách trắng `ui-allow` | **10** chỗ (khớp bảng SRS 8.2) — liệt kê dưới |

Lưu ý khi đọc `sweep.mjs`: chỉ số `nSmall` đếm liên kết "Bỏ qua điều hướng" (159 × 40, chỉ hiện khi có focus) → 1 ở 390/375 và nhiều ở 1440 — **không phải lỗi**; `audit.mjs` (phép `TOUCH`) đã loại nó. Cổng chính của sweep là `FORBIDDEN = 0`.

### `ui-allow` hiện có (10 chỗ)
| Tệp:dòng | Lý do |
| --- | --- |
| `features/chat/ChatScreen.module.css:33` | vạch chọn `inset 2px` |
| `features/gradebook/GradeScheme.module.css:63` | cùng lớp nổi với Popover |
| `features/gradebook/Gradebook.module.css:35` | vòng focus là token |
| `features/gradebook/Gradebook.module.css:58` | cùng lớp nổi với Popover |
| `shared/ui/ActionList.module.css:10` | vạch "đang chọn" |
| `shared/ui/CommandPalette.module.css:5` | tắt vòng focus (khung ngoài đã có) |
| `shared/ui/DataTable.module.css:33` | hàng có vòng focus riêng |
| `shared/ui/DataTable.module.css:34` | chỉ báo focus của hàng |
| `shared/shell/AppShell.module.css:150` | `main` nhận focus bằng chương trình |
| `shared/styles/tokens.css:78` | định nghĩa vòng focus toàn cục |

## Ảnh "trước" (US-PU-05 AC2)
12 ảnh PNG tại `docs/sprints/3/qc/shots/before/`, mẫu `<route>-<w>.png`, từ cùng bản dựng, `ep_demo_state` sạch, đồng hồ giả lập 29/10 09:20, font đã tải:

| Route | Vai (cookie) | Tệp |
| --- | --- | --- |
| `/` | SV B (`sv-2`) | `home-1440.png`, `home-390.png` |
| `/chat` | SV B | `chat-1440.png`, `chat-390.png` |
| `/threads` | SV B | `threads-1440.png`, `threads-390.png` |
| `/inbox` | GV | `inbox-1440.png`, `inbox-390.png` |
| `/gradebook` | GV | `gradebook-1440.png`, `gradebook-390.png` |
| `/settings/llm` | Admin | `settings-llm-1440.png`, `settings-llm-390.png` |

1440 × 900 và 390 × 844 (cảm ứng); `/dev/ui` không có ảnh trước (route mới). `ls docs/sprints/3/qc/shots/before | wc -l` → `12`.

## Sửa công cụ QC khi lập baseline
- `audit.mjs`: phép `HEADER đặc (00-AC12)` không áp dụng cho `/login` (không có `header`) → bỏ hàng thay vì FAIL (trước đó 3 hàng FAIL giả ở 1440/390/375).
- `sweep.mjs`: chạy lại được với `browser.open`/`tab.run` hiện hành (tham số không còn nhân bản `VIEWPORTS` chứa hàm; trả `results` từ `tab.run`; `timeout` 280000).

## Việc QC làm tiếp
Sau mỗi story PU dev bàn giao: chạy lại bảng trên + `proto-curl.sh all` + sweep → ghi vào `audit-log.md` (số hàng, FAIL, chênh so với nền); lệch ảnh `visual.spec.ts` đối chiếu `shots/before/`.
