# Cách một mình bạn xây EduPilot v2 cùng Claude Code

Bạn làm một mình, nên bạn giữ ba vai cùng lúc: **chủ sản phẩm** (quyết định làm gì), **reviewer** (đọc diff), **tester** (bấm thử).
Claude Code là người thi công. Quy trình dưới đây tồn tại để bù cho việc không có người thứ hai soát lỗi.

## 1. Cài bộ tài liệu vào repo (làm một lần)

```bash
cd TA_Agent
# chép toàn bộ nội dung bộ kit vào gốc repo: CLAUDE.md, docs/, .claude/, scripts/, frontend/ (token + logo; không ghi đè file nào đang có)
git checkout -b chore/edupilot-v2-docs
git add CLAUDE.md docs .claude scripts frontend/src/shared/styles frontend/public/brand && git commit -m "docs: EduPilot v2 PRD, kiến trúc, phase, quy trình"
```

Mở Claude Code tại gốc repo. `CLAUDE.md` được nạp tự động mỗi phiên. Ba lệnh riêng của dự án nằm ở `.claude/skills/`:

| Lệnh | Khi nào | Việc nó làm |
| --- | --- | --- |
| `/phase P3 L2` | Đầu phiên | Đọc tiến độ + phase file, làm đúng MỘT lát việc, trình kế hoạch trước rồi mới code |
| `/gate P3` | Khi xong các lát của phase | Chạy từng lệnh nghiệm thu, báo PASS/FAIL trung thực, không tự sửa |
| `/handoff` | Cuối phiên | Ghi `PROGRESS.md`, tick việc xong, ghi chú luận văn, commit |

Nếu phiên bản Claude Code của bạn chưa nhận `.claude/skills/`, chép ba file `SKILL.md` thành `.claude/commands/phase.md`, `gate.md`, `handoff.md` (định dạng cũ, vẫn chạy). Tài liệu chính thức: https://code.claude.com/docs

## 2. Nhịp một phiên làm việc (60–120 phút)

1. `/phase <MÃ> <LÁT>` → Claude Code đọc ngữ cảnh và **trình kế hoạch**. Với thay đổi lớn, bật plan mode (`/plan`) để nó không sửa file khi đang lập kế hoạch.
2. Bạn đọc kế hoạch. Hỏi lại ba câu: *Có đụng file nào ngoài phạm vi lát này không? Migration có sửa file cũ không? Test nào sẽ chứng minh việc này xong?* Sửa kế hoạch rồi mới duyệt.
3. Claude Code thi công, commit từng bước.
4. Bạn đọc diff (mục 3) và bấm thử.
5. `/handoff`. Rồi `/clear` trước khi sang lát khác.

**Một lát = một phiên = một ngữ cảnh sạch.** Đừng kéo một phiên qua nhiều lát: ngữ cảnh đầy thì chất lượng giảm và nó bắt đầu quên nguyên tắc. Thấy phiên dài, dùng `/context` xem cái gì đang chiếm chỗ; ưu tiên `/handoff` + `/clear` hơn là `/compact`.
Trí nhớ giữa các phiên nằm ở `docs/PROGRESS.md` và git, không nằm trong cuộc hội thoại.

## 3. Bạn là reviewer duy nhất: đọc gì, bỏ qua gì

Không ai đọc được hết mọi dòng AI viết. Dồn sức vào chỗ sai là chết:

| Luôn đọc từng dòng | Đọc lướt | Tin test |
| --- | --- | --- |
| Mọi file migration | Handler CRUD | Code sqlc sinh ra |
| `internal/auth`, `CourseAccessGuard`, mọi chỗ kiểm quyền | Component frontend | DTO, mapper |
| `internal/grade` (tính điểm) | Template mail | Cấu hình lint |
| `internal/privacy` (mask/unmask, phân loại kênh) | Seed | |
| `platform/crypto` (AES-GCM) | | |
| Prompt chấm bài và prompt trích công thức điểm | | |
| `internal/llm/scheduler`, `platform/outbox`, middleware idempotency | | |
| Luồng tài khoản F1: token, cookie, thu hồi phiên, nối roster | | |
| Mọi truy vấn trả dữ liệu cá nhân: có lọc `course_id` + `user_id` từ phiên không? | | |
| `shared/styles/tokens.css` và mọi thay đổi vào primitive ở `shared/ui/` sau phase PU | | |
| Mọi thay đổi vào test, contract test, CSV đối chiếu | | |

Mỗi lần xong lát, chạy `git diff --stat main...` và tự hỏi: *có file nào không nên bị đụng không?* Đó là dấu hiệu lan phạm vi.
Cuối các phase đụng auth / PII / điểm (PG, P3, P6, P7): nếu phiên bản của bạn có `/security-review` thì chạy nó trên nhánh trước khi merge.

## 4. Bạn là tester duy nhất: cổng nghiệm thu hai lớp

1. **Máy chạy:** `/gate <MÃ>` (gồm cả `scripts/ui-antipatterns.sh`). Tất cả PASS mới đi tiếp.
2. **Bạn bấm:** làm đủ danh sách "Bạn tự kiểm" trong phase file **và cổng UX** ở `UX.md` mục 6 — trên điện thoại thật, với mạng 3G chậm trong DevTools. Danh sách đó ngắn có chủ ý; đừng bỏ qua. Riêng P6 (điểm): **tự tính tay 3 sinh viên**.

Merge vào `main` chỉ khi cả hai lớp đều qua. Gắn tag `v2-<phase>` sau mỗi lần merge để luôn có điểm quay lại.

## 5. Khi Claude Code đi sai

| Dấu hiệu | Cách xử lý |
| --- | --- |
| Sửa test / contract test / CSV cho xanh | Dừng ngay. Hoàn tác thay đổi đó. Nhắc mục Cấm trong `CLAUDE.md`. Yêu cầu tìm nguyên nhân gốc |
| Cổng đỏ sau 2 lần sửa | Đừng để nó thử lần 3 kiểu mò. Yêu cầu viết test tái hiện nhỏ nhất, hoặc bạn thu hẹp lát việc |
| Diff lan sang module khác | Hoàn tác phần lan, ghi vào Nợ trong `PROGRESS.md` |
| Đề xuất thêm hạ tầng (Kafka, K8s, microservice, vector DB riêng) | Từ chối. Hỏi: "nút cổ chai nào ở tải T1 mà cái này gỡ?" — xem `SYSTEM_DESIGN.md` mục 1.3 và 4 |
| Màn mới trông như dashboard SaaS: tường thẻ số liệu, card lồng card, mọi thứ bo tròn đổ bóng, đỏ làm nền | Dán `DESIGN.md` §21 vào, yêu cầu làm lại theo hợp đồng route ở §14. Đừng sửa lặt vặt |
| Tự chế biến thể nút / chip / bảng cho riêng một route | Từ chối; mở rộng primitive ở `shared/ui/` nếu thật sự thiếu |
| Sa vào đánh bóng giao diện vô tận | `AGENT_PROMPT.md` giới hạn đúng HAI lượt QA. Hết hai lượt thì dừng |
| Viết `fetch` trần / spinner riêng / bảng riêng / confirm riêng | Yêu cầu dùng `frontend/src/shared/`; thiếu thì bổ sung vào đó |
| Danh sách không phân trang, truy vấn trong vòng lặp | Từ chối merge; luật 13 trong `CLAUDE.md` |
| Tự thêm thư viện lạ | Từ chối trừ khi có trong bảng ở `ARCHITECTURE.md`; nếu thật cần, thêm vào bảng trước |
| Đổi hợp đồng API cũ trong PG | Không bao giờ chấp nhận trong PG |
| Quên nguyên tắc giữa phiên dài | `/handoff`, `/clear`, mở phiên mới |
| Hỏng nặng | Quay về commit trước (commit nhỏ là bảo hiểm của bạn); nếu phiên bản có tính năng rewind/checkpoint thì dùng |

Khi tài liệu mâu thuẫn hoặc thiếu: **bạn** quyết định, ghi vào `DECISIONS.md`, rồi sửa PRD/phase. Đừng để Claude Code tự đoán yêu cầu nghiệp vụ.

## 6. Lịch: 23,5 tuần tới vạch bảo vệ, 25,5 tuần tới vạch thí điểm thật

Hai vạch đích khác nhau (xem `PRODUCTION_READINESS.md`): **bảo vệ đồ án** = xong P10; **trường dùng thật** = xong PR và trường đã đồng ý các mục pháp lý, vận hành. Lượt rà soát end-to-end thêm 1,5 tuần vào đường tới bảo vệ (P2 +0,5 tài khoản an toàn, P4 +0,5 SLA ticket + kiểm duyệt, P7 +0,5 nộp bài trong app + phúc khảo) và một phase PR 2 tuần.

| Tuần | Phase | Mốc phải đạt cuối giai đoạn |
| --- | --- | --- |
| 1 (nửa đầu) | P0 (0,5) | CI xanh, có openapi + bộ dữ liệu đánh giá + mốc k6 |
| 1 (nửa sau)–4 (giữa) | PG (3) | Nền Go viết mới: contract test khớp `openapi.yaml` 100%, hạ tầng SSE + outbox + blob, chạy được `--scale gateway=2` |
| 4 (giữa)–5 | PU (1,5) | Token + app shell + primitive đủ 8 trạng thái; `/chat`, `/threads` dựng lại |
| 6–7 (giữa) | P1 (1,5) | Đổi provider trên UI; Scheduler + trần ngân sách qua test. **Đã hỏi thầy về D4** |
| 7 (giữa)–9 | P2 (2,5) | F1 + F2 đi trọn: xác minh email, mời giảng viên, mở lớp, mã tham gia; test mạo danh MSSV bị chặn; seed 2 lớp |
| 10–11 | P3 (2) | F3: E1 đạt; payload LLM sạch; stream chịu được tải lại trang |
| 12–13 (giữa) | P4 (1,5) | F4 + F5: escalation → mail; SLA ticket; kiểm duyệt Threads |
| 13 (giữa)–14 | P5 (1,5) | F7 + F8: tạo lịch buổi học; điểm danh trên điện thoại ≤ 60 s; hồ sơ 360 |
| 15–16 (giữa) | P6 (1,5) | F10: 30/30 khớp bảng tính tay; nhập XLSX; mở khoá có lý do |
| 16 (giữa)–19 (giữa) | P7 (3) | F9 + F11: tạo bài → nộp trong app → chấm nháp → công bố → phúc khảo. **Bắt đầu chấm tay 60 bài** |
| 19 (giữa)–20 (giữa) | P8 (1) | F6 + F13: thư viện, lịch, ICS, ingest chạy nền |
| 20 (giữa)–21 | P9 (1,5) | F12: thi thử, bài QUIZ có khoá chat, import Forms |
| 22–23,5 | P10 (2,5) | F14–F17; mỗi luồng có spec E2E; `make eval`, `make load-t1`, `make chaos`; visual QA hai lượt. **→ VẠCH BẢO VỆ** |
| 24–25,5 | PR (2) | F18; bảo mật, sao lưu + diễn tập khôi phục, giám sát, đồng ý + quyền dữ liệu, runbook, staging. **→ VẠCH THÍ ĐIỂM** |

**Đường thí điểm sớm** (nếu trường muốn dùng thật trong khi bạn còn đang làm): P0 → P5 rồi PR rút gọn ≈ 16 tuần, thí điểm giai đoạn 1 (hỏi đáp, escalation, điểm danh — chưa chấm tự động, chưa công bố điểm). P6–P10 làm tiếp song song; dữ liệu thật từ thí điểm đưa vào chương thực nghiệm. Chi tiết: `PRODUCTION_READINESS.md` mục 5.

**Ba điểm dừng xem lại tiến độ:**

- Giữa tuần 4: PG chưa xong → rủi ro lớn nhất. Cắt ngay hạng mục 1–4.
- Giữa tuần 13: chưa xong P4 → cắt 1–6.
- Giữa tuần 19: chưa xong P7 → cắt hết danh sách, rút P9 về chỉ trắc nghiệm.

**Thứ tự cắt khi trễ** (từ trên xuống; mỗi dòng ghi phần tiết kiệm ước tính):

0. Red Thread Transition và mọi chuyển động trang trí; bố cục riêng cho màn ≥ 1440 px (giữ ba mốc còn lại) — P10, PU (≈ 2 ngày)
1. Kịch bản lỗi nâng cao của mock-graph: `slow`, `partial_failure`, `expired_token` (giữ `happy`, `paged`, `throttle`, `no_consent`) — P7 L4 (≈ 1 ngày)
2. `GenerateQuestions` (giữ trích từ đề cũ), bỏ E5 — P9 L1 (≈ 3 ngày)
3. `FormsImportSource` (giữ QUIZ trong app) — P9 L3 (≈ 2 ngày)
4. Tầng NER cho PII — P3 L1 (≈ 2 ngày) — **đã cắt bởi D46, ≈ 2 ngày đã tiết kiệm**; giữ dòng để không đổi thứ tự cắt
4a. Tuỳ chọn nâng cao của mã tham gia: giới hạn tên miền email, sĩ số, ngày hết hạn (giữ mã + tạo lại + bật/tắt + yêu cầu duyệt); chia sẻ ngân hàng câu hỏi và công thức giữa các lớp (giữ chia sẻ tài liệu) — P2 (≈ 1,5 ngày)
4b. Toàn bộ `mock-graph` + nút "Đồng bộ từ Teams" (giữ adapter + unit test `httptest`) — P7 L4 (≈ 3 ngày). Chỉ cắt khi thật sự trễ, vì đây là thứ duy nhất cho phép demo luồng Teams
5. Hàng đợi ghi cục bộ khi mất mạng cho điểm danh (giữ cập nhật lạc quan + báo lỗi) — P5 (≈ 2 ngày)
6. `<CommandPalette>` và phím tắt nâng cao của `/inbox`, `/grading` (giữ điều hướng bàn phím cơ bản) — P2, P4, P7 (≈ 2 ngày)
7. Xuất PDF báo cáo lỗ hổng (giữ Markdown); cảnh báo ngưỡng ở M13-A — P10 (≈ 2 ngày)
8. Kịch bản k6 (d) và test hỗn loạn khởi động lại Redis (giữ giết worker) — P10 (≈ 2 ngày)

**KHÔNG BAO GIỜ cắt** (đây chính là "cái nền"): tài khoản an toàn + quy tắc nối danh sách lớp theo email đã xác minh (P2), phân quyền xem prompt (P10), phúc khảo (P7), gateway không trạng thái + object storage (PG), token + app shell + primitive đủ trạng thái (PU), LLM Scheduler (P1), chuẩn API + outbox (P2, P4), contract test (PG), test tải hỗn hợp chat + chấm bài (P10), `make eval` (P10).

Nếu quỹ thời gian thật là 14–15 tuần thì cắt danh sách trên là KHÔNG đủ (chỉ bớt ≈ 3,5 tuần). Khi đó phải chọn một trong ba, và nên bàn với thầy hướng dẫn: (a) bỏ hẳn M4 luyện đề + form trắc nghiệm (−1,5 tuần) và M13-B/C (−1 tuần); (b) rút phạm vi PG xuống mức tối thiểu để chạy được P1–P3 (mất contract test đầy đủ và một phần nền không trạng thái); (c) xin thêm thời gian. Ghi vào `DECISIONS.md`.

**Lịch trên chưa tính D45 và D46.** D45 (viết mới toàn bộ, không kế thừa mã Project III) thêm ≈ 3–5 tuần; D46 (chỉ Go, bỏ service Python AI) bỏ đi phần việc dựng và kiểm thử một service riêng. Số tuần trong bảng và thứ tự cắt ở trên giữ nguyên cho tới khi **chủ dự án** chốt lại ước lượng ròng.

## 7. Việc chỉ bạn làm được (Claude Code không thay được)

| Việc | Hạn | Vì sao |
| --- | --- | --- |
| API key ≥ 2 provider LLM | Trước P1 | Không ai cấp hộ |
| Hỏi thầy hướng dẫn về trục nghiên cứu (D4) | Trước tuần 4 | Quyết định chương thực nghiệm làm sâu cái gì |
| Hộp thư thử + app password | Trước P7 | IMAP/SMTP thật |
| Quy chế môn học thật | Trước P6 | Demo trích công thức thuyết phục hơn file tự soạn |
| **Chấm tay ≥ 60 bài theo rubric** | P7 → trước P10 | Không có thì E3 vô nghĩa. Mất khoảng 8–10 giờ, rải ra mỗi ngày 5 bài |
| Duyệt 100 câu AI sinh, ghi tỷ lệ | P9 | Số liệu E5 |
| VPS / máy demo | Trước P10 | |
| Đọc diff các vùng nhạy cảm | Mọi phase | Bạn là reviewer duy nhất |
| Làm việc với trường về pháp lý dữ liệu cá nhân, hạ tầng, mail, LLM được phép, vị thế của điểm | Trước PR, càng sớm càng tốt | `PRODUCTION_READINESS.md` mục 1–2; không có sự đồng ý của trường thì không có thí điểm |
| Tìm 2 người ngoài dự án dùng thử không hướng dẫn | PR | Bạn không còn nhìn thấy chỗ khó hiểu của chính mình |

## 8. Viết luận văn song song

`/handoff` ghi 3–6 gạch đầu dòng vào `docs/thesis-notes/<phase>.md` sau mỗi phiên. Cuối mỗi phase, bạn bổ sung: ảnh màn hình, số đo, một đoạn "vì sao làm thế này".
Đến tuần 17, chương thiết kế và chương thực nghiệm gần như đã có sẵn nguyên liệu. Đừng để dồn việc viết vào hai tuần cuối.

Khung chương gợi ý: 1 Giới thiệu · 2 Khảo sát và yêu cầu (từ `PRD.md`) · 3 Thiết kế (từ `ARCHITECTURE.md`, nhấn vào port Go, tường lửa PII + che danh tính, chấm tự luận, công thức điểm từ quy chế) · 4 Triển khai · 5 Thực nghiệm E1–E6 · 6 Kết luận.
Ghi rõ một bảng "kế thừa từ Project III của nhóm / làm mới trong ĐATN" — hội đồng sẽ hỏi.
