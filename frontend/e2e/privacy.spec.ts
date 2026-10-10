import { expect, test } from "@playwright/test";
import { asDemo } from "./support/session";
import { MID, SID, chatApi, frame, json, msg, noContent, session, sse } from "./support/chat-fixtures";

// US-P3-05 AC13: không placeholder trên màn hình; dòng "Đã ẩn n …" + "Tìm hiểu" mở giải thích tại chỗ; n = 0 thì không có dòng.
test.beforeEach(async ({}, info) => {
  test.skip(info.project.name !== "desktop", "một bề rộng là đủ");
});

for (const [masked, text] of [[2, "Em là Vũ Hoàng Giang, MSSV 20229001, điểm giữa kỳ của em thế nào?"], [0, "Quy chế thi nói gì?"]] as const) {
  test(`no placeholder visible (masked=${masked})`, async ({ page, context }) => {
    await asDemo(context, "student", { person: "sv-2" });
    let sent = false;
    await chatApi(page, {
      "POST /chat/sessions": json(session(), 201),
      [`POST /chat/sessions/${SID}/messages`]: (r) => {
        sent = true;
        return r.fulfill(sse(frame("status", { message_id: MID, stage: "received" }, "1:1-0"), frame("token", { off: 0, t: "Chào Vũ Hoàng Giang, mình xem được dữ liệu của bạn." }, "1:2-0"),
          ...(masked ? [frame("notice", { masked }, "1:3-0")] : []), frame("done", { message_id: MID, citations: [], low_confidence: false, degraded: false }, "1:4-0")));
      },
      [`GET /chat/sessions/${SID}/messages`]: (r) => r.fulfill(json({ items: sent ? [msg({ content: "Chào Vũ Hoàng Giang, mình xem được dữ liệu của bạn.", masked_count: masked, citations: [] }), msg({ id: "u1", role: "USER", content: text, citations: [] })] : [], next_cursor: null })),
      [`PUT /chat/messages/${MID}/feedback`]: noContent,
    });
    await page.goto("/chat");
    const box = page.getByRole("textbox", { name: "Câu hỏi của bạn" });
    await box.fill(text);
    await box.press("Enter");
    await expect(page.getByText("Chào Vũ Hoàng Giang")).toBeVisible();
    const notice = page.getByText(`Đã ẩn ${masked} thông tin cá nhân trước khi gửi cho AI`);
    if (masked) {
      await expect(notice).toBeVisible();
      await page.getByRole("button", { name: "Tìm hiểu" }).click();
      await expect(page.getByText(/được thay bằng ký hiệu/)).toBeVisible();
    } else {
      await expect(page.getByText(/Đã ẩn \d+ thông tin/)).toHaveCount(0);
    }
    expect(await page.locator("body").innerText()).not.toMatch(/\[\[\s*(SV|MSSV|EMAIL|SDT|CCCD)(_\d*)?/i);
  });
}
