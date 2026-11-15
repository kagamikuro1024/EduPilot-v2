# Ghi overrides91.json: TC -> [KQ, ghi chú]. Chạy lại tập lệnh này sau khi thêm mục.
import json, os
H = os.path.dirname(os.path.abspath(__file__))
ov = json.load(open(H + "/overrides91.json")) if os.path.exists(H + "/overrides91.json") else {}
def put(tcs, kq, note):
    for t in tcs.split():
        ov["TC-" + t] = [kq, note]
G = "logs/grep-batch.log / grep-batch2.log; so sánh với main bằng `git diff 2c15dfce^1 2c15dfce` (1b72f54 không có merge-base với HEAD)"
put("UX02-1", "P", "rg còn 5 dòng ngoài 13 tệp liệt kê: shared/lib/datetime.ts (mô-đun định dạng), insights/KnowledgeGap.tsx:14-15, observability/AdminView.tsx:19,46 (hai tệp không thuộc 13 tệp của AC1); 13 tệp đã sạch. " + G)
put("UX04-3", "P", "RERUN 2026-11-16 sau 25cd3352 (P-11-57, DEF-R9.1-02): quét frontend/src/**/*.css: 0 luật :hover ngoài @media (hover: hover)")
put("UX04-5", "P", "tokens.css:73-75 --ep-fast 140ms, --ep-normal 200ms, --ep-slow 200ms (≤ 200)")
put("UX04-7", "P", "DESIGN.md:492 «- Drawer/dialog: ≤ 200ms.» đúng một dòng (không có khoảng trắng quanh «/» so với mẫu rg)")
put("UX04-8", "P", "rg `transition: width` trong frontend/src: 0 kết quả")
put("UX04-15", "P", "git diff 2c15dfce^1 2c15dfce -- frontend/package.json: chỉ bỏ framer-motion, embla-carousel*; không thêm thư viện chuyển động")
put("UX04-16", "P", "evidence-9.1/scripts/keyframes-check.mjs: keyframes ngoài landing chỉ animate opacity/transform (+ ngoại lệ ep-saved background-color, spinner, ba chấm, shimmer); không stagger ngoài landing. Landing có stagger/nhiều keyframes theo P-107-12 (trang giới thiệu được chuyển động marketing)")
put("UX04-18", "P", "ep-motion-safe: SubmissionReview.tsx:498,534, DataTableVirtual.tsx:209, AttendanceGrid.tsx:123; base.css:52 luật lưới 1ms còn nguyên, :55 .ep-motion-safe 200/160 ms")
put("UX21-11", "P", "frontend/e2e/landing.spec.ts: «không gọi API nào (kể cả refresh)» (dòng 17-35) + «có cờ ⇒ vẫn gọi làm mới phiên» (dòng 38); tệp này chạy xanh 21/21 trên bản dựng cổng (logs/devspecs.log)")
put("UX01-2", "P", "28 mảnh kiểm của bảng tc-wave-2: 0 kết quả (rg -F, bỏ Demo*.tsx, test, spec)")
put("UX01-6", "P", "diff 3 tệp Demo*.tsx (7 dòng): chỉ nhãn «Threads»→«Hỏi đáp công khai» (US-07 AC4) và progressStyle (US-04 AC4); không đổi description (như QC 9.1 đã ghi nhận)")
put("UX01-12", "P", "git diff 2c15dfce^1 2c15dfce -U0 -- frontend/src | rg '^\\+.*(Info|CircleHelp)': 0 dòng (CircleHelp/Info xuất hiện ở HEAD là icon nav Luyện đề và icon thông báo ở sprint 10, không phải câu giải thích)")
put("UX01-13", "P", "logs/antipatterns.log: PageHeader description tĩnh > 40 ký tự trên bản sao → ✗ «Câu mô tả tĩnh dưới tiêu đề», exit 1")
put("UX01-14", "P", "logs/antipatterns.log: description={\"abc\"} và description={dynamicVar} → không ✗, exit 0")
put("UX01-15", "P", "logs/antipatterns.log: --selftest = 32 / 32 phép bắt được (có phép «Câu mô tả tĩnh dưới tiêu đề»)")
put("UX10-9", "P", "bash scripts/ui-antipatterns.sh trên cây HEAD: exit 0, không ✗")
put("UX07-8", "P", "RERUN 2026-11-16 sau 7322ac62 (DEF-R9.1-01): grep \"Kiểm duyệt Threads\" frontend/src = 0")
put("UX07-14", "P", "diff shell.spec.ts của merge 9.1: chỉ đổi nhãn nav + chuyển sang toHaveText tự thử lại; không xoá/nới assertion")
put("UX07-16", "P", "rg DESIGN.md: còn duy nhất dòng 42 «Hỏi đáp công khai (Threads)» là ghi chú đổi tên có chủ ý")
put("UX12-8", "P", "rg \"0[,.]25\" frontend/src/features/attendance: 0 kết quả")
put("UX91-X3", "P", "git diff --stat 2c15dfce^1 2c15dfce: không có backend-go/db/migrations mới; không `fetch(` trần mới ngoài shared/api; Demo*.tsx chỉ đổi như UX01-6")
put("UX09-14", "P", "scripts/landing-video/record.mjs + README: chỉ đọc; không gửi chat, không Duyệt/Công bố/sửa điểm (rg Duyệt|Công bố|Gửi|type(|Enter = 0)")
put("UX04-1 UX04-2", "P", "spec w1 đỏ vì selector `class*=Button-module` không khớp bản dựng cổng (tên lớp băm); bản sửa qc2-r91-mock-extra.spec.ts (data-variant) xanh: 5 biến thể nút, transform 0,97 ≤ 100 ms, trở lại")
put("UX02-10 UX02-11", "P", "frontend/e2e/datetime.spec.ts (5 ca: bảng AC2, 23:59/00:01 ICT, ranh giới nửa đêm, null → «—», múi giờ máy khác) xanh trên bản dựng cổng; không có vitest ở HEAD (script test đã bỏ)")
# LT landing amended by 10.7
LTS = "Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log)"
ltmap = {
 "UX08-1": "AC1: h1 đổi còn 8 từ «Vận hành lớp học có trợ lý AI», sáu mục mới (P-107-9, P-107-11)",
 "UX08-2": "AC1: danh sách h2 đổi theo TC-UX08-1 (P-107-9, P-107-11)",
 "UX08-3": "AC2: màn đầu đúng hai nút «Đăng nhập» + «Xem video» (P-107-13)",
 "UX08-4": "AC2: như UX08-3 (P-107-13)",
 "UX08-6": "AC3 + US-09 AC2: clip/poster màn đầu bị bỏ, thay bằng đoạn sản phẩm HTML (P-107-13)",
 "UX08-7": "AC3 + US-09 AC2: không còn clip để đo (P-107-13)",
 "UX08-8": "AC3 + US-09 AC2: không còn clip (P-107-13)",
 "UX08-9": "AC3/AC14: không còn clip, reduce theo AC14 mới (P-107-12, P-107-13)",
 "UX08-10": "AC3/AC14: không còn clip (P-107-13)",
 "UX08-11": "AC3: không còn clip (tab ẩn) (P-107-13)",
 "UX08-12": "AC3 + AC19: không còn clip tải lỗi (P-107-13)",
 "UX08-14": "AC4 mới: nút Tạm dừng/Phát 32 px, vùng chạm ≥ 44 px hoặc vùng đệm (P-107-13)",
 "UX08-15": "AC4 mới (P-107-13)",
 "UX08-22": "AC6: khung ghim đổi: ba tên màn ngang + ảnh rộng hết khung (P-107-13)",
 "UX08-23": "AC6 mới (P-107-13)",
 "UX08-24": "AC6: ảnh khung ghim rộng 1280 px, không còn giới hạn 92.160 B (P-107-13)",
 "UX08-25": "AC6/AC12: hoạt cảnh khung ghim theo techlead 8.3 (P-107-12)",
 "UX08-26": "AC7 đổi theo khung ghim mới (P-107-13)",
 "UX08-27": "AC7 đổi theo khung ghim mới (P-107-13)",
 "UX08-29": "AC8: ba bước dùng IntersectionObserver thay view-timeline (P-107-12)",
 "UX08-30": "AC8 mới (P-107-12)",
 "UX08-31": "AC8/AC9: số đếm do script (P-107-12)",
 "UX08-32": "AC8/AC9 mới (P-107-12)",
 "UX08-33": "AC9: tiêu đề khối «Ba số đo trên dữ liệu mô phỏng», bỏ dòng chú thích (P-107-11; BA 14/11)",
 "UX08-34": "AC9 mới: bỏ chú thích riêng (P-107-11)",
 "UX08-35": "AC10: hai cột vai thành hai tab (P-107-13)",
 "UX08-36": "AC10 mới (P-107-13)",
 "UX08-38": "AC11 sửa: chữ thân ≥ 16 px, dẫn ≥ 18 px (P-107-9)",
 "UX08-39": "AC11: bỏ điều kiện «một h2 mỗi khung nhìn» (P-91-8 (b)) và khoảng cách đổi theo sáu mục 100svh (P-107-9)",
 "UX08-40": "AC12: vi mô khi rê đổi (nhích 1 px, mũi tên 3 px; P-107-12)",
 "UX08-41": "AC12 mới (P-107-12)",
 "UX08-42": "AC12 mới (P-107-12)",
 "UX08-46": "AC14 mới: reduce → không gắn m-on (P-107-12)",
 "UX08-47": "AC13/AC15: không còn animation-timeline trong Landing.module.css; thay bằng IntersectionObserver (P-107-12)",
 "UX08-49": "AC15: nay có đúng MỘT IntersectionObserver (LandingMotion.tsx:143) thay P-91-2 «không IntersectionObserver» (P-107-12)",
 "UX08-53": "AC19: lỗi clip không còn; ảnh khung giữ kích thước theo AC19 mới (P-107-13)",
 "UX08-56": "AC21 mới: ảnh mốc landing-1440/390 anonymous reduce (P-107-13)",
 "UX09-17": "US-09 AC2/AC5: không còn clip màn đầu; video đầy đủ chỉ mở bằng «Xem video» (P-107-13)",
 "UX09-18": "US-09 AC2/AC5 (P-107-13)",
}
for t, n in ltmap.items(): put(t, "LT", n + ". " + LTS)
empty = "P-107-2: trạng thái rỗng = tiêu đề ngắn + tối đa một nút; bỏ câu giải thích (luật chữ UI 09/10/2026); DESIGN.md §15, UX.md đã vá. Hộp thư nay hiện «Chưa có câu hỏi cần xử lý»"
for t in "UX05-1 UX05-2 UX05-3 UX05-5 UX05-11".split(): put(t, "LT", empty)

# ---- lượt 2: FFP / LH / TAY / landing ----
put("UX09-1", "P", "ffprobe gioi-thieu-v1.mp4: duration 64,73 s (60–90)")
put("UX09-2", "P", "ffprobe: h264 1440×900, không luồng audio (cả .mp4 và .webm vp9 1440×900)")
put("UX09-4", "LT", "US-09 AC2 «clip màn đầu — bỏ»: gioi-thieu-clip-v1.* và poster đã xoá khỏi public/landing (P-107-13); ls public/landing không còn tệp clip")
put("UX09-5", "LT", "US-09 AC2: không còn clip (P-107-13)")
put("UX09-6", "LT", "US-09 AC2: không còn clip để kiểm faststart (P-107-13); video đầy đủ vẫn có moov trước mdat (ftyp,moov,free,mdat), h264")
put("UX09-20", "LT", "US-09 AC2/AC13: không còn clip lặp để xem (P-107-13)")
put("UX09-7", "P", "public/landing/: gioi-thieu-v1.mp4 1,06 MB, .webm 1,56 MB, .webp 51,7 KB (< 8 MB); curl -I https://localhost:1143/landing/gioi-thieu-v1.mp4 → 200, accept-ranges: bytes")
put("UX09-12", "P", "curl -H 'Range: bytes=0-99': mp4 và webm đều 206, accept-ranges: bytes, content-range: bytes 0-99/…")
put("UX09-PQ", "P", "/landing/../package.json, /landing/%2e%2e/package.json, /landing/..%2f..%2fnext.config.ts, /edupilot/x.mp4: đều trả khung ứng dụng (HTML), không có nội dung tệp riêng; không tới MinIO")
put("UX09-8", "P", "logs/r91land.log: <video> trong hộp thoại có controls, preload=none, playsinline, poster=/landing/gioi-thieu-v1.webp, không autoplay; 0 request video trước khi bấm «Xem video»")
put("UX09-3", "P", "ffmpeg trích 65 khung (1 khung/s), 22 khung đã xem qua ảnh ghép frames/sheet-1..3.png: thẻ tiêu đề → chat riêng (có che tên: «Đã ẩn 3 thông tin cá nhân trước khi gửi cho AI») → Hộp thư hỗ trợ → Chấm bài + duyệt bài → Sổ điểm + giải trình → Hôm nay → thẻ kết; con trỏ đỏ hiện; phụ đề tiếng Việt in vào hình, dấu đúng, không tràn. Người xem: qc2 qua công cụ đọc ảnh; người thật chấm cảm nhận mượt: ngoài phạm vi kiểm tự động")
put("UX09-15", "P", "tesseract OCR 65 khung (frames/ocr.txt, logs): không có email, localhost, .local, IP, 9router, khoá; MSSV 2022xxxx là dữ liệu seed mô phỏng (D44)")
put("UX09-9", "P", "Lighthouse mobile devtools-throttle 3 lượt LCP 2419/2190/2087 ms (≤ 2500), CLS 0, desktop 437 ms (+ 1 lượt lạnh 2177 ms khi máy tải 40): logs/lh-*.json; poster không còn là ứng viên LCP (US-09 AC5 sửa 10.7); ghi nhận: ở 375 phần tử LCP là p.lead chứ không phải h1 (DEF-R9.1-05)")
put("UX08-43", "P", "Lighthouse mobile 3 lượt: LCP 2419/2190/2087 ms, CLS 0; desktop LCP 437 ms (lượt 1 lạnh 2177 ms), CLS 0; a11y 1,0. Máy chủ ép tải ~40 nên số mobile sát trần; ghi nhận p.lead là LCP ở 375 (DEF-R9.1-05)")
put("UX08-44", "P", "logs/r91land.log: JS tĩnh route / = 214,5 KB gzip (11 tệp) ≤ 256 KB; script nội tuyến tự viết 0,48 KB gzip (≤ 5 KB, AC13)")
put("UX08-20", "LT", "AC5/AC13 sửa 10.7: ngưỡng «chênh ≤ 1,5 KB so với v1 216 KB» bị thay bằng tổng ≤ 256 KB và script riêng ≤ 5 KB (P-107-12/13); số đo hiện hành ở UX08-44")
put("UX08-45", "P", "git diff package.json merge 9.1: chỉ bỏ framer-motion/embla; curl / (không JS) có h1 «Vận hành lớp học có trợ lý AI» và nội dung chính (Server Component)")
put("UX08-5", "F", "luật :hover màu (border-color/color/background) ở Landing.module.css:35,39,40,73,97,144,226 nằm ngoài @media (hover: hover) and (pointer: fine) → hover dính trên cảm ứng; xem DEF-R9.1-02")
put("UX08-13", "LT", "AC3: màn đầu không còn clip/poster; đoạn sản phẩm HTML (P-107-13)")
put("UX08-55", "LT", "AC21 sửa 10.7: landing.spec.ts nay kiểm đoạn sản phẩm/Tạm dừng/Xem video/ghim/hai tab; bản mới 21/21 xanh (logs/devspecs.log); TC cũ ghi «hai cột vai (không tab)», «clip không tải khi reduce» (P-107-12/13)")
put("UX08-PQ", "P", "logs/r91land.log: khách mở /chat, /gradebook → /login?next=…; HTML `/` không có @edupilot.local, Edupilot#, Bearer/JWT, sk-…, MSSV 2022xxxx")
put("UX08-37", "P", "AC10a: story-chat/review/gradebook-v1-1280.webp chụp lại ở commit aa4343f8 (15/11, 10.7 lô 8–9) sau đợt 3–5; xem ảnh story-review: có «Đã duyệt 0/4», phím A, «Duyệt bài», nhãn nav mới (Hỏi đáp công khai, Lỗ hổng kiến thức) = UI hiện hành")
put("UX91-X1", "P", "như UX08-37 (AC10a US-08)")

json.dump(ov, open(H + "/overrides91.json", "w"), ensure_ascii=False, indent=0)
print(len(ov))
