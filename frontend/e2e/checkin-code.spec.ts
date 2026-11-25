import { expect, test } from "@playwright/test";
import { SUB, checkinApi, json } from "./support/checkin-fixtures";

// US-AS-09 AC10: không có #token → ô «Mã điểm danh».
test("checkin code: ô «Mã điểm danh» (cú pháp ở placeholder) + «Lớp 761988» + nút Điểm danh; gửi mã CHUẨN HOÁ (O→0, I/L→1), ký chuỗi code:", async ({ page, context }) => {
  const log = await checkinApi(page, context);
  await page.goto("/checkin");
  const field = page.getByLabel("Mã điểm danh");
  await expect(field).toHaveAttribute("placeholder", "ABC 123");
  await expect(field).toHaveAttribute("autocapitalize", "characters");
  await expect(field).toHaveAttribute("autocomplete", "off");
  await expect(page.getByText("Lớp 761988", { exact: true })).toBeVisible();
  const go = page.getByRole("button", { name: "Điểm danh", exact: true }).first();
  await expect(go).toBeDisabled();
  await field.fill("abo 1l3");
  await field.press("Enter");
  await expect(page.locator("[data-part=done]")).toHaveText("Đã điểm danh 07:03");
  const b = log.posts[0].body as { code: string; token?: string; device: { public_key: string; signature: string } };
  expect(b.code).toBe("AB0113");
  expect(b.token).toBeUndefined();
  const valid = await page.evaluate(
    async ({ pub, sig, msg }) => {
      const bin = (s: string) => Uint8Array.from(atob(s), (c) => c.charCodeAt(0));
      const key = await crypto.subtle.importKey("spki", bin(pub), { name: "ECDSA", namedCurve: "P-256" }, false, ["verify"]);
      return crypto.subtle.verify({ name: "ECDSA", hash: "SHA-256" }, key, bin(sig), new TextEncoder().encode(msg));
    },
    { pub: b.device.public_key, sig: b.device.signature, msg: `edupilot-checkin-v1|code:AB0113|${SUB}` },
  );
  expect(valid).toBe(true);
});

test("checkin code: mã sai → lỗi tại ô (một câu), chữ đã gõ giữ nguyên; ký tự ngoài bảng mã không gọi mạng", async ({ page, context }) => {
  const log = await checkinApi(page, context, { post: (r) => r.fulfill(json({ code: "CHECKIN_TOKEN_INVALID", message: "m", trace_id: "b".repeat(32) }, 422)) });
  await page.goto("/checkin");
  const field = page.getByLabel("Mã điểm danh");
  await field.fill("ABC123");
  await page.getByRole("button", { name: "Điểm danh", exact: true }).first().click();
  await expect(page.getByText("Mã không đúng hoặc đã hết hạn. Nhập mã đang hiện trên màn chiếu.")).toBeVisible();
  await expect(field).toHaveValue("ABC123");
  expect(log.posts).toHaveLength(1);
  await field.fill("U#U");
  await field.press("Enter");
  await expect(page.getByText("Mã không đúng hoặc đã hết hạn. Nhập mã đang hiện trên màn chiếu.")).toBeVisible();
  expect(log.posts).toHaveLength(1); // không gửi
});

// P-11-137 (US-AS-15 AC16d): biết lớp không bật TRƯỚC khi sinh viên gõ: màn «Lớp này không điểm danh bằng mã.», không ô nhập, không nút Điểm danh, không gọi POST.
test("checkin code: lớp không bật điểm danh bằng mã → màn «Lớp này không điểm danh bằng mã.» (không ô nhập, không gọi POST)", async ({ page, context }) => {
  const log = await checkinApi(page, context, { selfCheckin: false });
  await page.goto("/checkin");
  await expect(page.getByText("Lớp này không điểm danh bằng mã.")).toBeVisible();
  await expect(page.getByLabel("Mã điểm danh")).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Điểm danh", exact: true })).toHaveCount(0);
  expect(log.posts).toHaveLength(0);
});

// DEF-R10.8-07 (TC-MOB-CL-4): cỡ chữ hệ thống 2,0× (≈ 30 px) — thanh dock «Điểm danh» phải nằm TRÊN thanh điều hướng theo chiều cao THẬT,
// nhãn thanh điều hướng một dòng (không xuống 2 dòng làm thanh cao thêm).
test("checkin code: chữ 2,0× (30 px) ở 375 → dock không bị thanh điều hướng đè, nhãn nav một dòng", async ({ page, context }) => {
  await checkinApi(page, context);
  await page.setViewportSize({ width: 375, height: 812 });
  await page.addInitScript(() => document.addEventListener("DOMContentLoaded", () => {
    const st = document.createElement("style");
    st.textContent = "html{font-size:30px}";
    document.head.appendChild(st);
  }));
  await page.goto("/checkin");
  await expect(page.locator("[data-part=checkin-dock]")).toBeVisible();
  const nav = await page.locator("[data-part=bottom-nav]").boundingBox();
  const dock = await page.locator("[data-part=checkin-dock]").boundingBox();
  expect(nav).not.toBeNull();
  expect(dock!.y + dock!.height).toBeLessThanOrEqual(nav!.y + 1); // đáy dock không lọt vào thanh điều hướng
  const lines = await page.locator("[data-part=bottom-nav] a, [data-part=bottom-nav] button").evaluateAll((els) =>
    els.map((el) => {
      const label = el.lastElementChild as HTMLElement;
      const lh = parseFloat(getComputedStyle(label).lineHeight) || parseFloat(getComputedStyle(label).fontSize) * 1.2;
      return Math.round(label.getBoundingClientRect().height / lh);
    }),
  );
  expect(lines).toEqual(lines.map(() => 1)); // mỗi nhãn đúng 1 dòng
});
