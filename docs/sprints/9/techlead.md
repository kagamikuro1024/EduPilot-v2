# Sprint 9: câu hỏi kỹ thuật cho Tech Lead

`## TL-<số> (<ai>, <ngày>): <tiêu đề>`, Tech Lead trả lời ngay dưới.

## TL-1 (PM, 2026-10-31): proposals #10, kiểm LCP cho `/assignments/[id]` (US-P7-03 AC9)
Dev nói `lcp.spec.ts -g /assignments` không đạt được nếu không đổi hợp đồng `shared/shell/intro`: route động, `h1` là tiêu đề bài. Dev đề xuất bỏ khỏi AC9, chỉ giữ 375 px + a11y. Tech Lead xem mã `lcp.spec.ts` / Lighthouse CI và chọn cách **vẫn đo được** LCP màn nộp bài của sinh viên (route seed cố định, phần tử LCP khác, hoặc thêm vào danh sách Lighthouse CI), hoặc giải thích vì sao thật sự không đo được. Không nới ngưỡng. Trả lời ngay dưới, báo PM 2 dòng.

**TL trả lời:**

**Kết luận.** LCP của `/assignments/[id]` **đo được** bằng Lighthouse CI lượt `devtools` (cùng cách, cùng ngưỡng 2.500 ms với 8 route hiện có). `lcp.spec.ts` là công cụ sai cho route này. Nhưng số đo thật hiện là **≈ 6,1 s**: AC9 đỏ vì hiệu năng thật, không phải vì không đo được. Bỏ `-g '/assignments'` khỏi dòng `Kiểm:` là đúng (phương án A). **Bỏ hẳn phần đo LCP thì không đúng**: thay bằng lệnh LHCI ở dưới.

**Vì sao `lcp.spec.ts` không dùng được cho route này.** Test đó kiểm một bất biến của US-PU-06: chữ lớn nhất màn đầu là chữ do `PreShell` vẽ từ máy chủ, và sau phiên không có ứng viên LCP nào mới. Bất biến này chỉ đúng khi mọi khối chữ dựng từ dữ liệu đều **nhỏ hơn** chữ máy chủ.
- Trên `/assignments/[id]`, phần tử LCP thật là **đoạn đầu của đề bài Markdown**. Đo được: `section > div.Markdown-module__md > p`, cả 3 lượt.
- Diện tích đoạn này phụ thuộc dữ liệu nên không có chuỗi giữ chỗ nào chặn được.
- Phiên ở bộ nhớ trình duyệt, cookie refresh `Path` = `/auth/*` của API (`authhttp/handler.go:22`, `:144-145`). Máy chủ Next không có phiên nên không vẽ trước dữ liệu được.

**Cách đo (dev làm, không đổi ngưỡng).**
1. `lighthouserc.json` và `lighthouserc.devtools.json`: thêm `http://localhost:3310/assignments/00000000-0000-7000-8000-0000000000a3`. Ở `ci.yml`, thêm vào nửa `devtools B` (hoặc tách `devtools C` nếu job vượt 25 phút).
2. `lighthouse-auth.cjs`: tra vai theo tiền tố (`/assignments/` → `student`).
3. `lighthouse-api.mjs`: trả `GET /courses/{cid}/assignments/{aid}/mine` theo đúng `MineOut` (`assignmentApi.ts`), đề bài lấy nguyên văn `fixtures/de-bai-bt03.md` và 4 tiêu chí của `rubric-bt03.json`, `submission: null`. Như vậy đo đúng màn thật chứ không đo trang lỗi (tiền lệ `/me`, `/settings/llm`).
4. US-P7-03 AC9 `Kiểm:` đổi thành `pnpm -C frontend exec lhci autorun --config=lighthouserc.devtools.json --collect.url=http://localhost:3310/assignments/00000000-0000-7000-8000-0000000000a3` → `largest-contentful-paint` trung vị ≤ 2500 (mức `error`, đã có trong `assertMatrix` theo mẫu URL).
5. `FEAT-ui-foundation` US-PU-06 AC4 đếm số URL trong tệp cấu hình (`… | length`), nên đếm phải tăng theo. Đổi qua `proposals.md` như lần thêm `/me`.

**Số đo (Tech Lead tự chạy).** Bản `origin/sprint/9-p7` @ `27edb226`, `pnpm -C frontend build:gate`. Dùng `lighthouserc.devtools.json` (devtools, 3 lượt, không tự đặt `throttling`), gateway giả có thêm `/mine` như mục 3. Tệp tạm ở `/tmp/tl-lh/`, không commit.

| URL | LCP trung vị | FCP | Phần tử LCP |
| --- | --- | --- | --- |
| `/assignments/…a3` | **6.148 ms** (6.173 / 6.148 / 6.120) — `✘ expected <=2500` | ≈ 4,1 s | `<p>` đầu của đề bài (Render Delay ≈ 6,1 s) |
| `/me` (đối chứng) | 1.461 ms | ≈ 0,9 s | `h1` của `PreShell` |

Thác yêu cầu của `/assignments` (ms, lượt 1):
- HTML xong 893.
- Các chunk JS xong ≈ 3.484.
- `auth/refresh` (tiền kiểm + gọi) 3.417 → 3.993.
- `me/courses` 4.077 → 4.655.
- **Sau đó** mới tải chunk động `StudentAssignment` + CSS: 4.662 → 5.505. Nguyên nhân: `dynamic(…, { loading: () => null })` ở `AssignmentScreen.tsx:6`, cộng thêm chờ `realCourseId`.
- **Sau đó** mới gọi `…/mine`: 5.532 → 6.114.
- LCP 6,1 s.

**Phần dev làm được ngay, trong phạm vi P7, không đổi hợp đồng:**
- (a) Liên kết tới bài tập (Hôm nay, lịch, `/me`, chuông) mang `?course={id}`. Màn gọi `…/mine` ngay khi có phiên, song song với `/me/courses`. Không có `?course` thì giữ đường cũ.
- (b) Nhập `StudentAssignment` tĩnh (hoặc `import()` ngay khi `AssignmentScreen` gắn).

Hai việc này bớt ≈ 0,85 s (chunk + CSS) + ≈ 0,6 s (một vòng). Ước còn **≈ 4,6–4,8 s** [SUY LUẬN, tính từ thác ở trên], **vẫn đỏ**.

**Vì sao ≤ 2,5 s không đạt được nếu chỉ tối ưu phía máy khách.** Ở cấu hình này, riêng JS xong đã ≈ 3,4 s. Mọi chữ dựng từ API (cần phiên ở bộ nhớ) không thể vẽ trước JS + `refresh` + một lần gọi dữ liệu ≈ 4,6 s. Các route đang xanh đều xanh nhờ chữ máy chủ là ứng viên LCP. Muốn ≤ 2,5 s chỉ có hai đường, cả hai vượt khuôn Tech Lead → **PM**:
1. **Mở rộng hợp đồng `intro` theo tiền tố route (phương án B của dev, sửa `FEAT-ui-foundation`)**, kèm quyết định giao diện để không khối chữ dữ liệu nào lớn hơn chữ máy chủ. Ví dụ đề bài hiện gọn vài dòng đầu và có `Xem toàn bộ đề`. Việc này đụng `DESIGN.md` §14 và luật chữ UI, phải qua BA / chủ dự án.
2. **Vẽ dữ liệu phía máy chủ**: Next cần phiên khi dựng HTML. Đây là đổi kiến trúc xác thực (`Path` cookie refresh, D-quyết định về token chỉ ở bộ nhớ). Tech Lead **không** đề xuất ở P7.

Nếu PM không chọn 1 hoặc 2 thì AC9 phần LCP giữ nguyên ngưỡng nhưng **đỏ có ghi nợ** (số đo ở trên), không đổi thành "bỏ đo".

**Đề xuất dòng `proposals.md`:** `#10 Tech Lead (TL-1): bỏ -g '/assignments' khỏi lcp.spec (bất biến PreShell không áp cho trang có phần tử LCP là dữ liệu), THAY bằng LHCI devtools cho /assignments/<id cố định> + /mine giả trong lighthouse-api.mjs (ngưỡng 2,5 s không đổi). Số đo hiện 6,1 s (FAIL): dev làm (a) ?course= + gọi /mine song song, (b) bỏ dynamic import → ước 4,6–4,8 s; đạt 2,5 s cần PM chọn: intro theo tiền tố + đề bài gọn ở màn đầu (FEAT-ui-foundation + DESIGN), hoặc ghi nợ.`

## TL-2 (PM, 2026-10-31): proposals #13, whitelist `AssignmentQueue` trong `TestNoQueryLinksByStudentCode`
Dev thêm truy vấn hàng chờ chấm có ô tìm theo MSSV, nên test bảo mật đỏ; dev đề xuất cho truy vấn này vào whitelist. Tech Lead xem truy vấn trên `origin/sprint/9-p7` (`c487e4fa`) và trả lời:
- Ô tìm này có nối dữ liệu / danh tính theo MSSV tự khai không, hay chỉ là bộ lọc hiển thị cho nhân viên trong phạm vi lớp?
- Nên lọc trên `enrollments.student_code_snapshot` (ROSTER / ADMIN), trên `users.student_code`, hay viết lại để khỏi cần whitelist?

Kết luận: chấp nhận whitelist hay sửa mã. Trả lời ngay dưới, báo PM 2 dòng.

**TL trả lời:**

**Kết luận: chấp nhận whitelist `AssignmentQueue` (#13). Không cần viết lại truy vấn; có một sửa nhỏ không bắt buộc.**

Đọc `store/queries/review.sql:3-32` và `assignment/review.go:100-124` ở `origin/sprint/9-p7` @ `1d5919c6`:

1. **Không nối danh tính theo MSSV.**
   - Hàng chờ chỉ lấy bài `match_status = 'MATCHED'`. Sinh viên của bài đã được xác định từ trước bằng `s.student_id`, do bộ khớp đặt hoặc giảng viên gán tay.
   - Mọi phép nối đều theo khoá: `u.id = s.student_id`, `e.course_id = s.course_id AND e.user_id = s.student_id`.
   - MSSV chỉ xuất hiện ở vị từ `q` (dòng 28): một bộ **lọc hiển thị** trên các hàng đã thuộc lớp `course_id` của nhân viên (route qua `CourseAccessGuard` Staff). Nó không gán bài, không ghi gì, không mở thêm dữ liệu nào ngoài những hàng nhân viên vốn đã được xem khi không có `q`.
   - Đúng loại với `ListMembers` (#10 sprint 4) và `CrmStudentsList` / `CrmStudentsCount` (#30 sprint 7).
2. **Lọc trên cột nào.** Dùng `enrollments.student_code_snapshot`, **không** dùng `users.student_code`: như vậy cột hiển thị và cột lọc là một (cột `student_code` của chính truy vấn này), và khớp `ListMembers`, `CrmStudentsList`. Không cần giới hạn `joined_via IN ('ROSTER','ADMIN')` ở đây: với bộ lọc hiển thị, MSSV tự khai của dòng `CODE` chỉ làm hàng **hiện ra** khi gõ số đó. Bài đã thuộc sinh viên đó theo `student_id`, không chuyển sang ai. Ràng buộc `joined_via` (TL-REVIEW TLR-1) chỉ cần ở chỗ **khớp**, và mã đã làm đúng: `submission.sql:58-59` trả `joined_via`, bộ khớp Go chỉ tin dòng ROSTER / ADMIN.
3. **Đầu vào an toàn.** `q` đã được thoát `\`, `%`, `_` ở `review.go:110-112`. Ký tự thoát mặc định của `ILIKE` là `\`, nên không gõ ký tự đại diện được. Tham số đi qua sqlc nên không có tiêm SQL.

**Sửa nhỏ (nên làm cùng story, không đổi spec):** tên đang so bằng `ILIKE` thô, nên gõ "kha" không ra "Khá". `ListMembers` và `CrmStudentsList` đều tìm tên qua `vn_fold(…)`. Đổi vế tên thành `vn_fold(u.full_name) like vn_fold(sqlc.narg(q)) escape '\'` (giữ `%…%`); vế MSSV giữ `ilike`. Đây là sửa nhất quán UX, không phải điều kiện của whitelist.

**Rủi ro còn lại của cơ chế whitelist (gợi ý, không chặn #13):** whitelist miễn **cả truy vấn** theo tên. Nếu sau này ai thêm `JOIN … ON e.student_code_snapshot = …` vào `AssignmentQueue` thì test vẫn xanh. Muốn chặt hơn: đổi `TestNoQueryLinksByStudentCode` để vị từ có `student_code` dạng `like` / `ilike` không cần whitelist, còn so bằng (`=`, `IN`, `ON … student_code`) thì luôn đỏ trừ `EnrollmentConflictByStudentCode` / `RosterStudentCodeConflict`. Test thuộc P2 nên phải qua `proposals.md` riêng nếu PM muốn.

**Dòng cho `proposals.md` #13:** `Tech Lead TL-2: ACCEPT whitelist — AssignmentQueue chỉ lọc hiển thị trên hàng MATCHED đã gán theo student_id, nối theo id, lọc trên student_code_snapshot (cùng cột hiển thị, như ListMembers/CrmStudentsList); q đã thoát ký tự đại diện. Nên: tìm tên qua vn_fold cho nhất quán.`
