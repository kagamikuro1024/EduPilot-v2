# Tiến độ

Claude Code: ĐỌC file này đầu mỗi phiên, CẬP NHẬT cuối mỗi phiên. Giữ ngắn; chi tiết nằm trong git log.

## Đang ở đâu
- Repo: `origin` = `github.com/kagamikuro1024/EduPilot-v2` (**public** từ 2026-10-09; lịch sử đã làm sạch secret của Project III bằng `git filter-repo`). Repo cũ `TA_Agent_v2` (private, lịch sử chưa làm sạch) = remote `old-private`, **không push thêm**; nhánh cục bộ `chore/edupilot-v2-docs`, `legacy-main` thuộc lịch sử cũ, **không bao giờ push lên `origin`**. `main` = sprint 1 + 1.5 + 2 + 3 + 4 + 5 + 5.5 + 6. Repo Project III `TA_Agent` = remote `old-origin`, không push thêm.
- Lộ trình: **11 sprint + 1.5 + 5.5 chen giữa** (`docs/sprints/ROADMAP.md`). Chạy cuốn chiếu. Mỗi sprint merge xong: `writer` viết phần quyển đồ án, PM + BA review.
- Sprint gần nhất: **6 — P3 Hai kênh + PII, P8 Tài liệu + Lịch** — xong, cổng P3 ĐẠT, cổng P8 ĐẠT, `docs/sprints/6/report.md`. Kế tiếp: quyển đồ án phần sprint 6, rồi sprint 7 (P4 + P5).
- Workflow giữ HF Space cũ thức đã dời vào `legacy/.github/` ở repo mới (repo cũ vẫn tự chạy bản của nó).

## Bảng phase

| Phase | Trạng thái | Bắt đầu | Xong | Cổng nghiệm thu | Ghi chú |
| --- | --- | --- | --- | --- | --- |
| P0 Chuẩn bị | Xong | 2026-10-01 | 2026-10-01 | PASS (`sprints/1/qc/gate-P0.md`) | Viết mới toàn bộ (D45), chỉ Go (D46) |
| PG Nền Go | Xong | 2026-10-01 | 2026-10-03 | PASS (`sprints/2/qc/report-GATE-PG.md`) | Gateway không trạng thái, `--scale gateway=2`, image 9,4 MB |
| PU Nền giao diện | Xong | 2026-10-02 | 2026-10-03 | PASS, LCP có điều kiện (`sprints/3/qc/report-GATE-PU.md`) | Ảnh mốc, axe, Lighthouse CI, `ci/ui-drift` |
| P1 LLM Gateway | Xong | 2026-10-02 | 2026-10-03 | PASS (`sprints/3/qc/report-GATE-P1.md`) | openai-go/v3, Scheduler ba làn, cầu dao; Anthropic chưa ghi replay |
| P2 Lớp học | Xong | 2026-10-03 | 2026-10-04 | ĐẠT CÓ ĐIỀU KIỆN (`sprints/4/qc/report-GATE-P2.md`) | Phiên cookie xoay vòng, nối chỉ bằng email, "Hôm nay", seed API thật; 0 lỗ hổng |
| PE Thi hằng tuần | Xong | 2026-10-08 | 2026-10-10 | ĐẠT CÓ ĐIỀU KIỆN (`sprints/5/qc/report-GATE-PE.md`) | Migration `00006_weekly_exam`; sandbox `go-judge`, hàng chấm kiểu outbox; tự công bố khi đóng; `TestNoAnswerLeak` |
| 5.5 UI panel nổi | Xong | 2026-10-09 | 2026-10-10 | ĐẠT CÓ ĐIỀU KIỆN (`sprints/5.5/qc/report-GATE-UI.md`) | D59 (a): canvas xám ấm + `Panel`; `ui-antipatterns.sh` 22 phép; axe 0 vi phạm; TBT +≤ 6 ms |
| P3 Hai kênh + PII | Xong | 2026-10-10 | 2026-10-11 | ĐẠT (`sprints/6/qc/report-GATE-P3.md`) | Che danh tính một chỗ trong `internal/llm`; E1 recall 0,960, chặn nhầm 0; sự kiện SSE đầu p95 85 ms |
| P4 Escalation + Mail | Chưa | | | | |
| P5 CRM + 360 | Chưa | | | | |
| P6 Sổ điểm | Chưa | | | | |
| P7 Chấm bài | Chưa | | | | |
| P8 Tài liệu + Lịch | Xong | 2026-10-10 | 2026-10-11 | ĐẠT (`sprints/6/qc/report-GATE-P8.md`) | `docling-serve` (profile `ingest`), tìm lai RRF + `tsv`, lịch UNION, ICS token băm |
| P9 Luyện đề | Chưa | | | | |
| P10 Đánh giá | Chưa | | | | → vạch bảo vệ |
| PR Sẵn sàng thí điểm | Chưa | | | | → vạch thí điểm thật |

## Luồng end-to-end (tick khi có spec E2E xanh cho đường chính + một nhánh lỗi)

F1 ☐ · F2 ☐ · F3 ☑ (sprint 6) · F4 ☐ · F5 ☐ · F6 ☑ (sprint 6) · F7 ☐ · F8 ☐ · F9 ☐ · F10 ☐ · F11 ☐ · F12 ☐ · F13 ☑ (sprint 6) · F14 ☐ · F15 ☐ · F16 ☐ · F17 ☐ · F18 ☐ · F19 ☑ (sprint 5, `playwright-cli` + e2e)

## Phiên gần nhất
- Ngày: 2026-10-11
- Đã làm: sprint 6 — 11 story PASS, cổng P3 + P8 ĐẠT (góp ý #1–#17, QC Q1–Q3, dev D1–D3, TL-REVIEW 28 TLR, TL-CI). Chủ dự án chốt: tên người đăng thread công khai, khoá giờ thi = chat + đăng thread, migration `tsv` + ICS token băm, nút "Nhờ giảng viên" ẩn tới sprint 7. Thêm vai `writer` (quyển đồ án ở `~/Documents/EduPilot-thesis/`, đã viết tới sprint 5.5).
- Đang dở: không.
- Bước kế tiếp cụ thể: `writer` viết phần sprint 6, PM + BA review; PM lập kế hoạch sprint 7 (P4 + P5), gồm TL-CI #3 / #5.

## Nợ (việc thấy cần nhưng ngoài phạm vi phase)
- Image object storage lâu dài (đang `pgsty/minio` fork) → lát blob của PG chốt + ghi D mới (proposals #7).
- `typescript` ghim `^5` (typescript-eslint chưa hỗ trợ TS 7).
- GitHub Actions ghim theo major, chưa theo SHA.
- Lịch WORKFLOW §6 chưa điều chỉnh theo D45/D46 — chủ dự án quyết.
- 9 câu hỏi sản phẩm của kịch bản demo (`specs/FEAT-demo-script/QUESTIONS.md`) — chặn P1/P2/P4/P7.
- `thesis-notes/legacy-perf.md` thân bài còn tiếng Anh.
- ~~**Lighthouse (PM #15 / #14):** `total-blocking-time` của 6 route người dùng đang `warn`…~~ **ĐÃ TRẢ (US-PU-06, sprint 5, #14):** TBT 6 route `error` ở `lighthouserc.json` (mô phỏng, ngưỡng 200 giữ nguyên); LCP `error` 2.500 ms ở `lighthouserc.devtools.json` (throttle thật; mô phỏng không chấm LCP vì sàn ≈ 254 KB JS ở 1,6 Mbps). Hồ sơ: `docs/sprints/5/handoff/dev-US-PU-06.md`, `docs/sprints/5/techlead.md` TL-1 / TL-2. Còn lại: `/dev/ui` giữ `warn`; TBT hai đỉnh (≈ 90 / ≈ 175) do một tác vụ chạy mã khung (`EvaluateScript` ≈ 230 ms ở 12×) rơi trong hay ngoài cửa sổ FCP → TTI.
- **Nợ của PE (US-PE-09, AC10):** (1) máy chấm code dùng chung host với gateway/worker → tách máy sandbox — PR; (2) precompiled header cho `<bits/stdc++.h>` (cắt thời gian biên dịch ~1 s/lần) — chưa làm; (3) checker tuỳ chỉnh (hiện chỉ `EXACT` / `TOKENS` / `FLOAT_EPS`) — chưa làm; (4) xoá lần `RUN` cũ và `exam_events` theo hạn giữ — PR (retention); (5) nối điểm bài thi vào sổ điểm — P6; (6) chat riêng tôn trọng khoá trong giờ thi (PE mới có cổng thử `/_test/exam/chat-gate`; chat riêng thật chưa có) — P3; (7) amd64 seccomp của judge chưa thử (dev chạy `-no-seccomp` trên Colima arm64); (8) `BlobReader` của judge chưa nối (test lớn > 64 KiB đọc qua worker); (9) đo p95 cuộn 1.000 dòng ở bảng điểm và `docker compose logs | grep CANARY` chưa chạy; (10) ảnh mốc `visual` chưa tái tạo cho màn PE; (11) `@real` e2e chỉ chạy qua `check-exam-seed.mjs demo` (không có lượt UI thật).
- **Nợ của 5.5:** `Dialog` đang đóng nằm trong DOM của Panel → đưa ra portal (Q-QC-UI04-2); chữ nút chính Admin (Q-QC-UI06-1); `audit-login.mjs` bốn vai + `lhci` ở máy QC chưa chạy; "Lớp cần chú ý" (P5) và "Tiếp tục học" (P3 / P9) hiện khi phase đó có hình dữ liệu (#2).
- **Nợ của sprint 6:**
  - Chưa ép được với provider giả: `OVERLOADED` / quá hạn che (TC-49/50 P3-02, TC-26 P3-05); chưa đo lại 3G chậm.
  - Nhiều test còn dùng `pgxpool.New` mặc định thay vì `RuntimePool`, nên lỗi PgBouncer có thể ẩn.
  - Vai DB gateway / worker là superuser; cần `NOSUPERUSER NOBYPASSRLS` (QC Q2, PR).
  - Báo upstream lỗi `docling_parse` arm64.
  - TL-CI #3, #5 (sprint 7).
  - Thu hồi chia sẻ tài liệu (TLR-5 P8); "nạp lại tri thức đã xác nhận" (P4); tài liệu mới trên "Hôm nay".
  - E1 nên có người ngoài đội soạn thêm mẫu.
- ~~**PRD / DECISIONS chữ cũ:** `PRD.md` §2 còn sandbox ngoài phạm vi; D3~~ **Đã vá ở sprint 6** (proposals #1, #3).
- Caddy: upstream tĩnh + `health_uri` khi số bản gateway cố định (P10/PR, sprint 2 #3).
- **Nguồn việc của "Hôm nay" mà phase sau phải đăng ký (US-P2-11, SRS 4.7 bảng bậc):** P4 `TICKET` (10), thread; P5 điểm danh, `STUDENT_ATTENTION` (95) + "Lớp cần chú ý"; P6 `GRADE_SCHEME_UNCONFIRMED` (70); P7 hạn nộp, `GRADING_REVIEW` (30), `APPEAL` (20), `UNMATCHED_SUBMISSION` (60); P9 `QUESTION_REVIEW` (90), QUIZ; P10 insight. **Đã đăng ký ở sprint 6:** `AI_CONFIRM` (50, P3, proposals #5) và `continue[]` của sinh viên (P3). Mỗi phase gọi `today.Aggregator.Register(provider)` ở `today.NewService` và thêm Kind vào `allowed` (bộ lọc theo vai).

## Ánh xạ migration
| Số goose | Tên | Phase |
| --- | --- | --- |
| 00001 | pg_platform | PG |
| 00002 | llm | P1 |
| 00003 | course_foundation | P2 |
| 00004 | auth_hardening | P2 |
| 00005 | vn_fold | P2 |
| 00006 | weekly_exam | PE (US-PE-01) |
| 00007 | chat_threads | P3 (US-P3-01) |
| 00008 | privacy | P3 (US-P3-01) |
| 00010 | chunk_search | P8 (US-P8-01, cột `content_chunks.tsv`) |
| 00011 | calendar | P8 (US-P8-03; số 00009 bỏ trống, proposals #13) |

## Việc chỉ chủ dự án làm được
- [x] API key ≥ 2 provider LLM (OpenAI + Gemini, 2026-10-03; nên xoay khoá vì đã dán trong chat, đặt trần chi phí)
- [ ] Hỏi thầy hướng dẫn về D4 (trước tuần 4)
- [ ] Hộp thư thử + app password cho IMAP/SMTP (trước P7)
- [ ] Quy chế môn học thật (trước P6, nếu có)
- [ ] Chấm tay ≥ 60 bài theo rubric (bắt đầu ở P7, xong trước P10)
- [ ] Duyệt 100 câu hỏi AI sinh, ghi tỷ lệ (P9)
- [ ] VPS hoặc máy demo (trước P10)
- [ ] Gặp trường về pháp lý dữ liệu cá nhân, hạ tầng, mail, LLM được phép (trước PR; bắt đầu hỏi từ sớm)
- [ ] 2 người ngoài dự án dùng thử (PR)
- [ ] Tải zip template ĐATN "Định hướng ứng dụng (Tiếng Việt)" từ Overleaf về `~/Downloads`; cài MacTeX (`brew install --cask mactex-no-gui`, cần mật khẩu máy) — cho `writer`
- [ ] Quyết lịch sau D45 (lùi vạch bảo vệ hay cắt) — trước sprint 2
