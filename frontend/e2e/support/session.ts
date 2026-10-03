import type { BrowserContext } from "@playwright/test";

export type DemoRole = "student" | "ta" | "teacher" | "admin";

/** Đặt phiên MÔ PHỎNG (cookie ep_demo_*) trước khi mở trang — như bộ đổi vai của prototype 1.5. */
export async function asDemo(context: BrowserContext, role: DemoRole, opts: { person?: string } = {}) {
  const url = "http://localhost:3310";
  await context.addCookies([
    { name: "ep_demo_role", value: role, url },
    ...(role === "student" ? [{ name: "ep_demo_person", value: opts.person ?? "sv-2", url }] : []),
  ]);
}
