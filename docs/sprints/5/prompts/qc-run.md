# Prompt cho `qc` — Chạy test sprint 5 (US-PU-06 + PE)

**Worktree riêng của QC** từ `origin/sprint/5-pe` (như các sprint trước; không dùng worktree của dev).
**Công cụ nghiệm thu bằng tay:** `playwright-cli` (chủ dự án yêu cầu 2026-10-08) — cài toàn cục `npm i -g @playwright/cli@latest`, xem `playwright-cli --help` / https://playwright.dev/agent-cli/. Dùng cho kiểm thử thăm dò, đi luồng F19, chụp ảnh bằng chứng, xem snapshot DOM. Không chạy `playwright-cli install` trong repo (nó tạo `.playwright/` và sửa `.gitignore`); trình duyệt tự tải lần đầu. Đặt `PLAYWRIGHT_CLI_SESSION=qc`, đóng phiên trình duyệt sau mỗi lượt. Không thay test tự động: TC có lệnh Playwright / Go vẫn chạy như cũ.


## Việc
1. **Trước khi dev giao story đầu:**
   - Ghi các câu hỏi Q-QC sprint 5 còn treo vào `docs/specs/FEAT-weekly-exam/QUESTIONS.md` (mục Q-QC) rồi báo PM.
   - Viết `docs/sprints/5/qc/tc-US-PU-06.md` từ dòng US-PU-06 của `docs/sprints/5/plan.md`:
     - LCP ≤ 2,5 s và TBT ≤ 200 ms ở mức `error` trên CI cho 7 route;
     - TBT trung vị đo bằng 12× trên máy ≤ 170 ms (cách đo: `docs/sprints/4/techlead.md` TL-1);
     - không hồi quy ảnh mốc, axe, đăng nhập và phiên.
2. **Khi dev giao story** (có `docs/sprints/5/handoff/dev-<story>.md`):
   - chạy `tc-<story>.md`;
   - viết báo cáo `docs/sprints/5/qc/report-<story>.md`;
   - commit, push, báo PM ≤ 5 dòng.
   - Dev giao nhiều story một lúc thì chạy theo thứ tự.
3. **Story cuối (US-PE-09) xong:** chạy `gate-PE.md`, ghi `report-GATE-PE.md`.

## Luật
- QC tự chạy, không tin số của dev.
- Sandbox tấn công bằng mã QC tự viết.
- `visual.spec` chạy trong image `mcr.microsoft.com/playwright` đúng phiên bản.
- **RAM có hạn** (máy 18 GB, colima 4 GiB):
  - mỗi lúc chỉ một stack của QC (`EP_PORT_OFFSET=100`);
  - Playwright tối đa 2 worker;
  - xong một story thì tắt stack và Chrome, rồi chạy `docker volume prune -f`.
- Không mở subagent; không đụng cổng 3100; không `pkill` node chung.
