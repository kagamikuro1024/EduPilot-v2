# Prompt cho `dev`: thi công sprint 7 (P4 + P5)

**Nơi làm:** worktree `/Users/kuro/Documents/TA_Agent_v2-s7`, nhánh `sprint/7-p4-p5`. Copy `.env.local` đã có sẵn trong worktree.

## Đọc
- `docs/team/DEV.md`, `docs/team/CONTEXT.md`.
- `docs/sprints/7/plan.md`: thứ tự story.
- Spec **APPROVED** của feature đang làm: `docs/specs/FEAT-escalation-notify/`, `docs/specs/FEAT-crm-360/`. Mỗi spec đọc đủ US, SRS, QUESTIONS (cột Trả lời) và mục "Quyết định PM" trong `TL-REVIEW.md`. Spec chưa APPROVED thì chưa làm story của nó.
- `docs/sprints/7/proposals.md`, phần "Quyết định PM".
- `CLAUDE.md`: riêng sprint này nhớ kỹ luật 2, 4, 10, 12, 13, 14 và các mục "Cấm tuyệt đối" về chat riêng, ghi chú quan sát, nhãn rủi ro, thời gian học trong `internal/grade`.

## Bắt đầu ngay: story 0 — US-CI-02 (TL-CI #3, #5)
Không cần spec sản phẩm. Nguồn: `docs/sprints/6/techlead.md` mục TL-CI, bảng đề xuất dòng 3 và 5 (6 tuỳ chọn, làm nếu rẻ); hiện trạng ở `docs/sprints/6/handoff/dev-US-P3-08.md` mục "CI nhanh hơn".
- **#3:** Playwright `--shard` (giữ `workers: 2`, `retries: 1`); Lighthouse chia matrix như TL đề xuất, cùng ngưỡng, cùng `numberOfRuns`.
- **#5:** tách `go-static` song song với `go-test` matrix; mọi dòng giữ `-race -tags testroutes`; dòng "phần còn lại" tính lúc chạy để gói mới không bị rơi.
- **Không được:** bỏ test, bỏ `-race`, bỏ bước đo thời gian, bỏ lượt Lighthouse, giảm `numberOfRuns`, tăng `workers`.
- **Chứng minh:** (a) tổng số test Go và số spec Playwright đã chạy bằng đúng trước khi tách (đếm từ log hai run); (b) thời gian wall trước / sau trên cùng SHA; (c) một lượt cố ý làm đỏ một test ở mỗi shard / mỗi dòng matrix thì CI đỏ (rồi hoàn tác). Nếu `main` có branch protection theo tên check, ghi tên check mới vào handoff để PM báo chủ dự án.
- Handoff `docs/sprints/7/handoff/dev-US-CI-02.md`, commit `US-CI-02: …`, push.

## Thứ tự sau đó (theo plan)
US-P4-01 → 02 → 03 → 04 → 05 → US-P5-01 → 02 → US-P4-06 → 07 → US-P5-03 → 04 → 05 → 06 → US-S7-SEED. PM báo khi spec APPROVED.

Mỗi story giao như sau:
- test Go và frontend xanh, CI xanh;
- handoff `docs/sprints/7/handoff/dev-<story>.md` ghi đã làm gì, file đổi, lệnh để QC chạy, nợ;
- commit `<story>: …`, push.

PM tự giao QC khi thấy handoff. Không chờ QC xong mới làm story tiếp.

## Luật riêng sprint này
- **Migration:** `00012_escalation_notify`, `00013_crm`, `00014_observation`. Không sửa migration cũ. Index phức hợp bắt đầu bằng `course_id`.
- **Outbox:** mọi thông báo / mail ghi cùng transaction với thay đổi nghiệp vụ; consumer idempotent.
- **SSE thông báo** phải chạy với `--scale gateway=2` (Redis pub/sub, không giữ trạng thái trong tiến trình).
- **Heartbeat** chỉ ghi Redis, không bao giờ ghi Postgres trực tiếp.
- **Không log** nội dung chat, ghi chú quan sát, tên / MSSV / email.
- **Mail** chỉ Mailpit ở local.
- **Stack:** colima vừa khởi động lại, chưa có stack nào chạy. Image docling đã xoá để lấy chỗ; chỉ kéo lại khi story cần nạp tri thức (US-P4-05, US-P4-06), báo PM trước.
- **RAM có hạn:** chỉ một stack; Playwright tối đa 2 worker; `docker volume prune -f` sau mỗi story.
- Phân vân kỹ thuật thì hỏi Tech Lead (`docs/sprints/7/techlead.md`). Spec sai thì ghi `docs/sprints/7/proposals.md`.
