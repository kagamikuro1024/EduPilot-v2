# Sprint 11: kênh Tech Lead

## TL-1 (US-P10-10, đợt 4): kế hoạch chạy E1–E6 trên Mac

PM hỏi (2026-11-14): kế hoạch `make eval` E1–E6 chạy trên Mac, LLM qua 9router local (`127.0.0.1:20128`, `ag/gemini-3.8-flash`) và khoá Gemini; E4 ≥ 2 provider; E3 cần chủ dự án chấm tay → phương án tạm. Khớp `docs/specs/FEAT-p10-eval` (APPROVED).

**TL trả lời:**

### Kết luận
1. Một stack đo `-p ep-eval`, seed T0, nhúng bằng `gemini-embed` thật (pha `seed` của `deploy/vps-llm.mjs`). Hai provider sinh chữ: **P1** = 9router `openai_compatible` `ag/gemini-3.8-flash`; **P2** = `gemini` trực tiếp, model do `PHASE=probe` chọn. Mỗi provider chạy một lượt rồi đổi tuyến, nên mọi số E4 đến từ cùng một stack và cùng một seed.
2. Mọi phần cần người (chấm E3, duyệt E5, soát nhãn E1/E2) dồn vào **một gói cho chủ dự án**. `make eval` vẫn chạy AI ngay, lưu kết quả thô, và ghi `WAITING_HUMAN` cho chỉ số cần chuẩn người. Khi chủ dự án điền xong, `make eval-report` tính lại mà **không gọi LLM**.
3. E2 không cần chạy lại LLM theo từng ngưỡng: `confidence` tính thuần sau khi sinh xong (`internal/agent/confidence.go` `Metadata`), ngưỡng chỉ so `confidence < threshold`. Một lượt hỏi mỗi provider là đủ dựng cả đường cong.
4. Điều kiện trước: sprint 10 và 9.5 đã merge vào nhánh chạy. Nhánh `sprint/11-p10-eval` hiện **chưa có** `questions/generate`, `gen-stats` (E5) và bản sửa schema Gemini TL-2h của 9.5 (E4 với P2).

### Điều kiện trước (kiểm 2026-11-14)
| Điều | Trạng thái | Cách kiểm / làm |
| --- | --- | --- |
| 9router | **Không chạy** (`curl 127.0.0.1:20128` → 000) | Chủ dự án bật `9router -n -t` (mặc định bind `0.0.0.0`). Kiểm `GET /v1/models` có `ag/gemini-3.8-flash` |
| Container → máy Mac | Đạt: `host.docker.internal` → `192.168.5.2` từ container Colima | `base_url` của P1 là `http://host.docker.internal:20128/v1`, không dùng `127.0.0.1` |
| Khoá 9router | 1 khoá bật trong `~/.9router/db/data.sqlite` (bảng `apiKeys`) | Script đọc kiểu `sqlite3 "file:…?immutable=1"` trong tiến trình như bản VPS cũ (`origin/main:deploy/vps-llm.mjs:54`). Không in, không ghi tệp |
| `deploy/vps-llm.mjs` | Bản 9.5 chỉ có Gemini | Thêm `PROVIDER=9router\|gemini` cho pha `live`. Nhánh 9router lấy lại từ bản cũ, đổi `base_url` |
| Stack | `ep-research` đang chạy (đợt 0 của 9.5) | Đo E sau khi đợt 0 xong (một stack tại một thời điểm), hoặc dùng lại chính `ep-research` nếu đã seed nhúng thật |

### Từng phép đo
| E | Dữ liệu | Cách chạy (chỉ HTTP, AC2) | Lời gọi LLM | Phần chờ người |
| --- | --- | --- | --- | --- |
| E1 | `benchmarks/pii/e1_dataset.jsonl` (200) | `eval_pii.py` gọi `threads/precheck`, lần này **nhúng thật** (lần trước nhúng `fake`: bước ngữ nghĩa không góp phần, `eval_pii.py:309`). Rò rỉ payload lấy từ `real_llm.py mask` (9.5) | 200 lần nhúng, 0 sinh chữ | Nhãn do agent viết (`LABELING.md`, bảng soát QC đang trống), chưa khớp AC3 «nhãn do người gán» → P-11-3 |
| E2 | **Chưa có.** Golden set quy chế (≈ 20) + 100 câu học thuật lấy từ tài liệu seed. Mỗi mục có `answerable` và `must_contain` (sự kiện hoặc số phải có) | Hỏi qua **Threads** của lớp đo: `ThreadPost.confidence` là đường HTTP duy nhất có `confidence` cho mọi câu trả lời (`openapi.yaml` `ThreadPost`; chat riêng chỉ trả `low_confidence`). Chấm đúng/sai **tất định**: thiếu sự kiện nào trong `must_contain` là sai; câu `answerable=false` mà không `no_context` là bịa. Quét ngưỡng 0,30 → 0,90, bước 0,05 (13 điểm ≥ 7, AC4) | ≤ 2/câu × 120 ≈ 240 mỗi provider | Soát 20 % nhãn (≈ 24 mục) → P-11-4 |
| E3 | 60 bài BT03, sinh **một lần** bằng P1 theo 5 tầng (tốt / khá / trung bình / yếu / lạc đề), lưu `benchmarks/data/e3/*.md` kèm sha256. Thêm 7 bài fixture `FEAT-assignments`; bỏ 5 bài chèn lệnh khỏi QWK | `real_llm.py grade-setup` / `grade-run` mở rộng: P1 chạy 2 vòng, P2 chạy 1 vòng | 67 × 2 lượt × 3 vòng ≈ 400 | Chủ dự án chấm mù 60 bài (phương án tạm bên dưới) |
| E4 | E2 và E3 trên P1, P2 | Chất lượng: điểm E2 và E3. Độ trễ: p50/p95 phía client (đến sự kiện đầu, đến xong) + `GET /admin/llm/usage`. Chi phí / 100 yêu cầu: token × **giá niêm yết** Gemini ngày đo (`ai.google.dev/gemini-api/docs/pricing`); với P1 ghi thêm «chi phí biên 0, thuê bao 9router» | đã tính ở E2, E3 | – |
| E5 | 100 câu do `POST …/questions/generate` sinh từ tài liệu lớp đo (P1) | Chủ dự án duyệt bằng chính UI ngân hàng câu hỏi. Script đọc `GET …/questions/gen-stats` (sprint 10). **Không đếm** câu đã duyệt sẵn trong seed | ≈ 10–20 | Duyệt 100 câu. Chưa đủ thì ghi cỡ mẫu thật (AC7) |
| E6 | Stack `ep-eval`, seed T0 | `docker stats` sau 60 s nghỉ; khởi động lạnh `up → /readyz 200` × 3; kích thước image; p95 ≥ 5 endpoint bằng bộ lấy mẫu Python tuần tự (200 yêu cầu/endpoint sau khởi động ấm). Không chạy lại tải T1 (Q-E-20); số T1 trích từ 9.5 | 0 | – |

Hạn mức: P1 ≈ 650 lời gọi, P2 ≈ 400.
- P2 free tier tính RPD theo project, đặt lại lúc 15:00 giờ VN (9.5 TL-1 §3). Vì vậy chia P2 làm hai cửa sổ: E2 trước, E3 sau.
- `rpm_limit` đặt 5 cho P2 và 10 cho P1 `[SUY LUẬN: chưa biết hạn mức của 9router/ag]`.
- Gặp `429` có `Retry-After` > 60 s: dừng lượt và ghi lý do (AC6, AC10).
- Không bật LLM thật ở stack nào khác trong lúc đo.

### Thứ tự chạy
1. Dựng `ep-eval` (cổng lệch, chỉ bind `127.0.0.1`), `PHASE=seed`, `scripts/seed.mjs`.
2. Khi tuyến sinh chữ còn `fake`: chạy E6, E1, rồi E1/E2/E6 lần hai để đo sai số tái lập (AC11).
3. `PHASE=live PROVIDER=9router`: sinh bài E3 → E2 → E3 chấm 2 vòng → E5 sinh 100 câu.
4. `PHASE=probe`, rồi `PHASE=live PROVIDER=gemini`: E2, sau đó E3 1 vòng (có thể sang cửa sổ RPD hôm sau).
5. `make eval-report`: viết JSON, MD, biểu đồ. Ghi `WAITING_HUMAN` cho chỉ số E3 vs người, E5, và mục soát nhãn E1/E2. Mỗi báo cáo có mục «Giới hạn» (AC9) và ghi nhà cung cấp + model + ngày.
6. Giao gói cho chủ dự án. Điền xong thì chạy `make eval-report` lần nữa, không cần stack.

`Makefile` gốc (chưa có tệp này):
- `eval`: kiểm `GET /readyz`; lỗi thì in «Gateway chưa chạy», thoát 1 (AC12). Sau đó chạy E1…E6.
- `eval-report`: chỉ tính từ kết quả thô đã lưu.

### E3: phương án tạm khi chưa có điểm người
- **Không** dùng điểm do LLM sinh làm chuẩn (AC5, Q-E-18: vòng tròn).
- AI chấm ngay cả 60 bài trên P1 và P2, lưu kết quả thô. Khi có điểm người, QWK / MAE / Pearson tính tức thì, không phải gọi lại LLM.
- Báo cáo tạm có thêm ba con số, **dán nhãn rõ "không phải độ đúng"**:
  - test–retest: QWK giữa 2 vòng của P1;
  - đồng thuận giữa provider: QWK giữa P1 và P2;
  - Spearman giữa điểm AI và **tầng thiết kế** lúc sinh bài. Đây là nhãn của người viết đề, không phải điểm chấm.
  
  Writer chỉ được trích các số này là «độ ổn định», không gọi là «độ chính xác».
- Gói chấm tay: CSV một dòng mỗi bài gồm mã bài (đã xáo thứ tự) và cột trống cho từng tiêu chí rubric BT03. **Không hiện điểm AI**, tránh neo theo AI. Kèm rubric và đáp án tham chiếu. Nếu có thầy hướng dẫn / TA chấm độc lập 20–30 bài thì ghi thêm QWK người–người làm mốc (P10 L3).
- Ước lượng công cho chủ dự án `[SUY LUẬN]`: chấm 60 bài khoảng 2–3 giờ; duyệt 100 câu E5 khoảng 40 phút; soát 50 nhãn E1 + 24 mục E2 khoảng 30 phút.

### Rủi ro
- **9router `ag/` hết hạn mức hoặc treo.** Ghi lý do theo AC6, thay P1 bằng model khác của 9router và ghi rõ model đã thay.
- **P2 chọn trúng `gemini-3.8-flash`.** Khi đó E4 chỉ còn khác đường đi, không khác model. Nếu probe cho qua, ưu tiên model khác P1 để phép so sánh có nghĩa.
- **Câu hỏi E2 bị gộp hoặc bị chặn.** Câu gần nhau có thể bị Threads gộp (`THREAD_SIMILAR_MIN=0.75`) hoặc bị tường lửa chặn. Câu `SKIPPED` / bị chặn được ghi riêng và không tính vào đường cong. Mỗi câu phải khác hẳn nhau.
- **Thiếu thư viện vẽ PNG.** AC1 cần PNG, nhưng `benchmarks/` hiện chỉ dùng thư viện chuẩn Python và `matplotlib` không có trong bảng thư viện → P-11-2.

**Đề xuất cho PM:** P-11-2, P-11-3, P-11-4.

**Cập nhật theo quyết định PM (`proposals.md` @2fb37bb7):**
- **P-11-2 (biểu đồ):** dùng SVG bằng thư viện chuẩn Python: một hàm nhỏ vẽ đường cong E2 (đường gấp khúc) và cột E4/E5 (`<rect>`). Hàm chỉ dùng màu cố định, không dùng font ngoài. `.md` nhúng `![](e2.svg)`. Không dùng `matplotlib`.
- **P-11-3/4:** báo cáo E1 vẫn có số nhưng ghi `labels:"WAITING_HUMAN"`. Golden set E2 do BA soạn theo định dạng ở bảng trên (`answerable`, `must_contain`).
- **9router (kiểm 14/11 sau khi bật lại):**
  - Gọi `GET /v1/models` bằng khoá trong sqlite trả `200`, có đúng `ag/gemini-3.8-flash`.
  - Từ container, `http://host.docker.internal:20128/v1/models` trả `200`.
  - 9router còn có model khác họ, ví dụ `ag/claude-sonnet-4-6` và `ag/gpt-oss-120b-medium`. Nếu P2 (Gemini trực tiếp) hết hạn mức hoặc probe chọn trúng 3.8-flash, dùng một trong hai làm P2 (vẫn là `openai_compatible`, chỉ đổi `model`). Nhờ vậy E4 so được hai model khác nhau. Ghi rõ model và lý do vào báo cáo theo AC6.

---

## TL-2 (US-P10-08, đợt 3): hướng sửa LCP 7 trang

PM hỏi (2026-11-14): hướng sửa LCP trên build production với dữ liệu T1, không nới 2.500 ms.

**TL trả lời:**

### Kết luận
- LCP của cả 6 route trượt đều là **chữ** và đều thuộc pha **Render Delay**. TTFB chỉ khoảng 15 ms, không có Load Delay. Nghĩa là không có ảnh chậm và máy chủ cũng không chậm. Chữ lớn nhất chỉ xuất hiện **sau** chuỗi `HTML → JS (3 đợt) → /auth/refresh → /me/courses → chunk Real* lười → dữ liệu`. Với throttle devtools (562,5 ms mỗi yêu cầu, 1,47 Mbps, CPU 4×), mỗi mắt xích tốn khoảng 0,6 s.
- `/` và `/chat` đạt (1,55 s) vì ứng viên LCP là chữ PreShell do máy chủ vẽ (`shared/shell/intro.ts`, US-PU-06 AC8). Sáu route còn lại vi phạm đúng luật AC8 đó trên dữ liệu T1: sau khi có phiên, xuất hiện một khối chữ lớn hơn h1 của PreShell.
- **Chỉ cắt chuỗi phía client thì không đủ cho LCP phụ thuộc dữ liệu** `[SUY LUẬN có số]`:
  - riêng tải JS đã mất khoảng 215 KB gzip ở 1,47 Mbps, cộng bootup khoảng 0,7 s (audit `bootup-time`), tức React chỉ sẵn sàng ở khoảng 2,6–2,9 s, kể cả khi `refresh` và dữ liệu đã chạy song song;
  - vậy chữ phụ thuộc dữ liệu vẽ sớm nhất khoảng 2,7–3,2 s.
  
  Muốn đạt ≤ 2,5 s chỉ có hai đường: **(a)** khối chữ lớn nhất vẫn là chữ của PreShell (luật AC8 có sẵn), hoặc **(b)** máy chủ vẽ luôn dữ liệu đầu trang (SSR). Đường (b) đụng auth: phải làm mới phiên phía máy chủ. → P-11-6.
- Chọn mặc định: làm **U1–U4** dưới đây (bắt buộc, cũng là tăng tốc thật cho người dùng), rồi **(a)** theo từng route. Nếu route nào vẫn trượt thì đi nhánh lỗi AC6.

### Bằng chứng
Nguồn: `benchmarks/.lighthouse-raw/devtools/*.json` của lần đo 9.5 TL-2f (commit `868cce5b`), worktree `TA_Agent_v2-s95`. Bảng dưới lấy từ audit `largest-contentful-paint-element` và `network-requests`.

| Route | LCP (ms) | FCP (ms) | Phần tử LCP | Mắt xích thừa thấy trong waterfall |
| --- | --- | --- | --- | --- |
| `/threads` | 4.896 | 880 | `<p>` của hộp thoại thời gian học («EduPilot ghi thời gian bạn dùng…», `StudyTimeTracker` trong `shared/shell/TrackingNotice.tsx`) | Hộp thoại chỉ hiện sau `/me/courses`. Ngoài ra `/threads/<id>?_rsc` được prefetch 16 lần (8 link × 2) |
| `/inbox` | 6.717 | 869 | `span.clamp` của dòng ticket («Phishing là gì…») | Chunk `RealInbox` lười 5,41 → 6,07 s, rồi `tickets` 6,10 → 6,70 s |
| `/gradebook` | 5.887 | 884 | Thông báo «Có điểm cộng ghi tay không được tính theo công thức» | `gradebook` 33 KB (4,85 → 5,78 s) tranh băng thông với prefetch `/?_rsc` 32 KB |
| `/me` | 6.256 | 886 | `p.headline` «Điểm quá trình hiện tại của bạn là 8,75 trên 10.» (lớn hơn h1 của PreShell; chú thích ở `intro.ts:22` đã cũ) | Chunk `RealMe` lười 5,30 → 5,99 s, rồi 5 truy vấn `me/*` 6,02 → 6,62 s |
| `/settings/llm` | 4.786 | 880 | Chữ trạng thái rỗng «Thêm nhà cung cấp có mô hình trò chuyện, rồi quay lại đây…» | `providers` / `routes` 4,14 → 4,72 s |
| `/assignments/[id]` | 4.639 | **3.994** | Hộp thoại thời gian học | Route động **không có PreShell** nên đến khi có phiên mới vẽ chữ đầu tiên (FCP 4 s) |

Chung cho mọi route:
- HTML **592 KB thô / 84 KB gzip**: 176 KB CSS nằm trong `<style>`, cộng 412 KB RSC flight. **36/36 mẫu cắt từ CSS đều có mặt lại trong flight**, tức CSS bị chép hai lần (đo trên `/inbox` của stack demo).
- Đây là lỗi đang mở của `experimental.inlineCss`: vercel/next.js #95141, #98079. Mỗi payload prefetch `/?_rsc` cũng mang theo khoảng 30 KB này.
- Sáu tệp phông `woff2` (~55 KB) được preload ưu tiên High, tải cùng lúc với JS.
- `/auth/refresh` chỉ bắt đầu khi JS chạy xong (≈ 3,5 s).
- Mọi route prefetch RSC của các link trên thanh bên: 8–12 yêu cầu, 40–80 KB, rơi đúng vào cửa sổ dữ liệu.

### Hướng sửa
**U1–U4: chung, không đổi giao diện** (ước tính từ waterfall `[SUY LUẬN]`, phải đo lại):

| # | Sửa | Tệp | Tiết kiệm | Rủi ro / kiểm |
| --- | --- | --- | --- | --- |
| U1 | Gọi `POST /auth/refresh` **sớm, đúng một lần**: script nội tuyến trong `<head>` của root layout bắt đầu `fetch` và giữ promise trên `window`; `refreshSession()` (`shared/data/authSession.ts:69`) dùng lại promise này rồi xoá. Giữ luật `/` + `isReturning()` | `app/layout.tsx`, `authSession.ts` | ≈ 0,6 s (đè lên thời gian tải JS) | **Cao:** gateway thu hồi phiên khi token refresh bị dùng lại (`internal/auth/session.go:372`, `REFRESH_REUSE`, không có thời gian ân hạn). Bắt buộc single-flight. Thêm e2e đếm đúng 1 `auth/refresh` mỗi lần tải. Token vẫn chỉ nằm trong bộ nhớ, không đổi mô hình auth |
| U2 | Tải chunk `Real*` **song song** với phiên: gọi `import("./RealX")` ngay khi module màn được nạp; `dynamic()` dùng lại đúng promise đó | `features/{me,inbox,threads}/…Screen.tsx` | ≈ 0,6–0,7 s | Ngân sách JS mỗi route không đổi vì chunk vẫn tách riêng |
| U3 | Không chờ `/me/courses` mới gọi dữ liệu: dùng lớp đã nhớ (`myCourses.ts`, localStorage `COURSE_STORAGE_KEY`) để `prefetchQuery` truy vấn đầu trang song song. Gặp `403`/`404` thì quay về đường cũ (CourseAccessGuard vẫn chặn ở máy chủ) | `shared/session/myCourses.ts`, `session.tsx` | ≈ 0,6 s | Lần đầu chưa nhớ lớp thì không lợi gì. Lighthouse luôn mở phiên mới nhưng localStorage giữ qua các lượt nếu cùng hồ sơ trình duyệt `[cần kiểm]` |
| U4 | `prefetch={false}` cho link thanh bên và link trong danh sách (thread, ticket). Muốn giữ cảm giác nhanh thì gọi `router.prefetch` khi rê chuột hoặc focus | primitive link của shell + danh sách | Giải phóng 40–150 KB giữa lúc tải dữ liệu, ≈ 0,3–0,8 s | Bấm điều hướng lần đầu chậm thêm 1 lượt RSC |

Không tắt `inlineCss`: CSS sẽ thành một yêu cầu chặn vẽ (≈ +0,6 s FCP), làm LCP của `/` và `/chat` (đang ăn theo PreShell) xấu đi hơn 10 %, trái AC4. U4 đã cắt phần lớn chi phí CSS chép đôi.

**(a) Từng route: khối lớn nhất sau phiên không được vượt PreShell** (luật AC8 có sẵn):
- `/settings/llm`, `/gradebook`: chữ gây trượt là câu giải thích / thông báo dài. Thay bằng trạng thái 1–2 từ + nút, đúng luật cứng «chữ trên UI» ở `CLAUDE.md` (ví dụ «Chưa có nhà cung cấp» + `Thêm nhà cung cấp`; «Ghi tay» gắn cạnh cột liên quan). Đổi chữ trên UI → BA/DESIGN xác nhận (P-11-6).
- `/me`: chuyển headline thành dòng `nhãn + giá trị` («Điểm quá trình  8,75 / 10»), cỡ chữ ≤ h1. Sửa luôn chú thích `intro.ts:22`.
- `/inbox`: dòng ticket kẹp 2 dòng ở 412 px nên lớn hơn h1 «Hộp thư hỗ trợ». Đường (a) chỉ đạt nếu dòng danh sách còn 1 dòng trên mobile, mà khi đó diện tích gần bằng h1, rất mong manh. Route này có khả năng **phải dùng (b) hoặc FAIL theo AC6**.
- `/threads`, `/assignments/[id]`: hộp thoại thời gian học là thứ chỉ hiện **một lần cho mỗi tài khoản** (SRS crm-360 7.5). Lighthouse đo tài khoản `t1sv0001` chưa bấm «Đã hiểu», tức luôn ở trạng thái lần đầu. Đây là quyết định giao thức đo → PM (P-11-5). Hai cách: đo ở trạng thái người dùng quay lại (bấm «Đã hiểu» một lần trước lượt đo, báo riêng số đo lần đầu), hoặc đổi UX hộp thoại.
- `/assignments/[id]` còn thiếu PreShell cho route động (FCP 4 s): thêm một mục PreShell theo tiền tố đường dẫn. Dù vậy, tiêu đề và mô tả bài là dữ liệu, nên route này cũng như `/inbox`: có khả năng phải dùng (b) hoặc FAIL.

**Giữ lành bộ kiểm:**
- `e2e/lcp.spec.ts` đang dùng mock nhỏ (`mockMe`, `mockLlmApi`) nên không bắt được các khối lớn của T1. Mock phải mô phỏng trường hợp xấu nhất của T1: ticket dài, có điểm cộng ghi tay, chưa có nhà cung cấp, có headline `/me`. Thêm `/assignments/[id]` vào `CASES`.
- `benchmarks/lighthouse/summarize.mjs` ghi thêm `lcp_element` cho mỗi route để thấy hồi quy.
- Báo cáo nên ghi thêm thời điểm dữ liệu đầu trang vẽ xong, bên cạnh LCP. Nhờ vậy luận văn không trình bày LCP-theo-PreShell như thể dữ liệu đã hiện.

### Đoạn minh hoạ U1 (không phải bản để chép)
```tsx
// app/layout.tsx, trong <head>: chạy trước mọi chunk; cùng điều kiện isReturning() với AuthProvider
<script dangerouslySetInnerHTML={{ __html:
  `if(location.pathname!=="/"||localStorage.getItem("${RETURNING_KEY}")){window.__epRefresh=fetch("/api/v1/auth/refresh",{method:"POST",credentials:"include"})}` }} />
// authSession.ts
const early = (window as { __epRefresh?: Promise<Response> }).__epRefresh;
delete (window as { __epRefresh?: Promise<Response> }).__epRefresh; // dùng đúng MỘT lần: REFRESH_REUSE thu hồi phiên
res = early ? await early : await fetch(`${API_BASE}/auth/refresh`, { method: "POST", credentials: "include" });
```

### Rủi ro
- U1 làm sai single-flight → người dùng bị đăng xuất ngẫu nhiên. Có e2e đếm 1 lần và test hai tab.
- Sau U1–U4 và (a), `/inbox` và `/assignments/[id]` có thể vẫn trên 2,5 s. Theo AC6: ghi số đo vào `proposals.md` và story FAIL route đó, hoặc chủ dự án chọn (b).
- Số trong bảng U chỉ là ước tính từ một lần đo. Phải chạy lại `benchmarks/lighthouse/run.sh` (3 lượt, trung vị) trên `ep-*`, không dùng stack demo.

**Đề xuất cho PM:** P-11-5, P-11-6.

**Cập nhật theo quyết định PM (`proposals.md` @2fb37bb7):**
- **P-11-6:** chỉ đi đường (a): U1–U4 cộng sửa từng route. Không SSR, không đổi auth. Route nào sau lượt sửa cuối vẫn trên 2.500 ms thì ghi vào Nợ (route, số đo, phần tử LCP, nguyên nhân), không nới ngưỡng.
- **P-11-5:** đo ở trạng thái người dùng quay lại.
  - `benchmarks/lighthouse/auth-ep.cjs`: sau khi đăng nhập vai student, gọi `POST /api/v1/me/tracking-notice/ack` bằng access token lấy từ phản hồi login. Lệnh này idempotent và không đổi dữ liệu T1 nào khác.
  - Số đo lần đầu chạy riêng một lượt với `LH_FIRST_VISIT=1`, bỏ bước ack, trên một tài khoản T1 chưa từng ack (ví dụ `t1sv0002`). Ghi vào báo cáo dưới nhãn «lần đầu (hộp thoại thời gian học)» và không chấm ngưỡng.
  - Mặc định `run.sh` và CI đo trạng thái quay lại.
