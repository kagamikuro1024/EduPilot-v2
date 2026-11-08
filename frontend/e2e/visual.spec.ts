import { expect, test, type BrowserContext, type Page } from "@playwright/test";
import { BASE_URL } from "./support/env";
import { acquireFakeApi, releaseFakeApi } from "./support/fake-api-lock";
import { gradebookApi } from "./support/gradebook-fixtures";
import { base, reset, script } from "./support/llm-fixtures";
import { asDemo, asJwt, type DemoRole } from "./support/session";

// US-PU-05 AC1: 7 route đại diện × 2 bề rộng = 14 ảnh mốc (SRS 8.3). Đồng hồ đóng băng 29/10/2026 09:20 giờ VN, animation tắt,
// font đã nạp. Cập nhật ảnh mốc CHỈ bằng `--update-snapshots` kèm giải thích trong handoff.
// Ảnh mốc sinh trên macOS arm64; nếu CI Linux lệch do khử răng cưa, sinh lại trong CI và giải thích (xem handoff).
test.describe.configure({ retries: 0 });
test.use({ reducedMotion: "reduce", colorScheme: "light" });
test.beforeEach(async ({}, info) => {
  test.skip(info.project.name !== "desktop", "tự đặt bề rộng");
});

const FROZEN = new Date("2026-10-29T09:20:00+07:00");
const WIDTHS = [
  { w: 1440, h: 900 },
  { w: 390, h: 844 },
] as const;
const ROUTES: Array<{ name: string; path: string; role?: DemoRole; admin?: boolean; staffToday?: boolean }> = [
  { name: "home", path: "/", role: "student" },
  { name: "chat", path: "/chat", role: "student" },
  { name: "threads", path: "/threads", role: "student" },
  { name: "inbox", path: "/inbox", role: "teacher" },
  { name: "hom-nay-gv", path: "/", staffToday: true }, // US-UX91-10 AC12: Hôm nay của giảng viên, việc chia nhóm + cột phải
  { name: "gradebook", path: "/gradebook", role: "teacher" },
  { name: "settings-llm", path: "/settings/llm", role: "admin", admin: true }, // Admin bằng token dev (dán ở cổng); màn THẬT với dữ liệu giả cố định
  { name: "dev-ui", path: "/dev/ui" },
];

/** Gateway giả cho "Hôm nay" của giảng viên: một lớp, đủ bốn kiểu việc (overdue, high, normal, EXAM_SIMILARITY) + lớp cần chú ý + buổi sắp tới. */
async function staffTodayApi(page: Page) {
  const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };
  const course = { id: "00000000-0000-7000-8000-00000000c001", class_code: "761987" };
  const ok = (r: import("@playwright/test").Route, body: unknown) => r.fulfill({ status: 200, contentType: "application/json", headers: cors, body: JSON.stringify(body) });
  const act = (kind: string, title: string, reason: string, urgency: string, href: string) => ({ id: `${kind}:${title}`, kind, title, reason, urgency, href, course });
  await page.route("**/api/v1/me/courses**", (r) => ok(r, { items: [{ course: { ...course, subject_code: "INT1006", name: "An ninh mạng", semester: "2026-2027-HK1", status: "ACTIVE" }, role_in_course: "TEACHER", enrollment_status: "ACTIVE" }], next_cursor: null }));
  await page.route("**/api/v1/courses/*/today**", (r) =>
    ok(r, {
      count: 5,
      actions: [
        act("JOIN_REQUEST", "3 yêu cầu vào lớp đang chờ duyệt", "Cũ nhất đã chờ 2 ngày.", "overdue", "/class/members?tab=pending"),
        act("GRADING_REVIEW", "2 bài AI cần xem kỹ", "Độ tin cậy thấp ở tiêu chí 2.", "high", "/grading"),
        act("PUBLISH_PENDING", "Công bố điểm Bài tập 02", "Đã duyệt đủ 18 bài.", "normal", "/grading"),
        act("APPEAL", "1 yêu cầu xem lại điểm", "Sinh viên gửi hôm qua.", "normal", "/grading"),
        act("EXAM_SIMILARITY", "Hai bài thi có đáp án giống nhau", "Xem khi rảnh.", "normal", "/exams"),
      ],
      attention: [{ student: { id: "00000000-0000-7000-8000-00000000b001", full_name: "Trần Thu Uyên" }, course, reason: "vắng 3/4 buổi; học ít" }],
      attention_total: 1,
      upcoming: [{ at: "2026-10-30T02:00:00Z", title: "Buổi 11 · An ninh mạng", place: "P.302", course }],
    }),
  );
}

async function settle(page: Page) {
  await page.evaluate(() => document.fonts.ready);
  await page.waitForLoadState("networkidle");
  await page.waitForTimeout(300);
}

for (const r of ROUTES) {
  for (const { w, h } of WIDTHS) {
    test(`${r.name} @${w}`, async ({ page, context }) => {
      if (r.admin) await acquireFakeApi(); // màn thật dùng máy chủ giả dùng chung
      try {
        await shoot(page, context, r, w, h);
      } finally {
        if (r.admin) releaseFakeApi();
      }
    });
    async function shoot(page: Page, context: BrowserContext, r: (typeof ROUTES)[number], w: number, h: number) {
      await page.setViewportSize({ width: w, height: h });
      await page.clock.setFixedTime(FROZEN);
      if (r.admin) {
        await reset(page);
        await script(page, base()); // dữ liệu cố định theo hợp đồng thật (support/llm-fixtures.ts)
      }
      if (r.admin) await asJwt(page, "ADMIN", { email: "admin@ptit.edu.vn", fullName: "Đỗ Hoàng Nam", expIn: 366 * 86400 }); // đồng hồ đóng băng 29/10: hạn phải dài hơn khoảng lệch với giờ thật; phiên thật giả lập (refresh → JWT ADMIN)
      else if (r.staffToday) {
        await asJwt(page, "TEACHER", { expIn: 366 * 86400 });
        await staffTodayApi(page);
      } else if (r.role) await asDemo(context, r.role, { expIn: 366 * 86400 }); // đồng hồ đóng băng 29/10: token mặc định (15 phút) đã hết hạn nên phiên rơi về Sinh viên — ảnh inbox / gradebook từng là trang chặn quyền
      if (r.path === "/gradebook") await gradebookApi(page); // #22: màn thật; dữ liệu cố định theo hợp đồng thật (support/gradebook-fixtures.ts)
      await page.goto(r.path);
      await page.locator("main").first().waitFor();
      await settle(page);
      await expect(page).toHaveScreenshot(`${r.name}-${w}.png`, { maxDiffPixelRatio: 0.005, threshold: 0.2, animations: "disabled", caret: "hide" });
    }
  }
}
