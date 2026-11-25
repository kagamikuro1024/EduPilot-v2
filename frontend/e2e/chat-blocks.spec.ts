import { expect, test } from "@playwright/test";
import { asDemo } from "./support/session";
import { MID, SID, chatApi, frame, json, msg, session, sse } from "./support/chat-fixtures";

// DEF-GP10-01: khối dữ liệu tool (chuyên cần, phát biểu, tài liệu) KHÔNG lộ khoá tiếng Anh / JSON thô cho sinh viên (DESIGN §13). Chạy cả desktop và mobile 375.
const ATTENDANCE = { recorded: 9, present: 6, late: 1, excused: 0, absent: 2, recent_absent: [{ session_no: 7, date: "2026-11-12" }, { session_no: 5, date: "2026-11-05" }] };
const PARTICIPATION = { speak_count: 2, points_total: "0.50", recent: [{ date: "2026-11-12", type: "SPEAK", points: "0.25" }] };
const LIBRARY = { results: [{ doc_id: "d1", title: "Quy chế", snippet: "…" }] };
const blocks = [{ kind: "attendance", data: ATTENDANCE }, { kind: "participation", data: PARTICIPATION }, { kind: "library_results", data: LIBRARY }];

test("khối chuyên cần / phát biểu: nhãn tiếng Việt, không khoá kỹ thuật, không JSON; khối tài liệu không dựng bảng thô", async ({ page, context }) => {
  await asDemo(context, "student", { person: "sv-2", expIn: 366 * 86400 });
  let sent = false;
  await chatApi(page, {
    "POST /chat/sessions": json(session(), 201),
    [`POST /chat/sessions/${SID}/messages`]: (r) => {
      sent = true;
      return r.fulfill(sse(frame("status", { message_id: MID, stage: "received" }, "1:1-0"), ...blocks.map((b, i) => frame("block", b, `1:${i + 2}-0`)), frame("token", { off: 0, t: "Đây là dữ liệu của bạn." }, "1:9-0"),
        frame("done", { message_id: MID, citations: [], low_confidence: false, degraded: false }, "1:10-0")));
    },
    [`GET /chat/sessions/${SID}/messages`]: (r) => r.fulfill(json({ items: sent ? [msg({ citations: [], content: "Đây là dữ liệu của bạn.", blocks }), msg({ id: "u1", role: "USER", content: "Em vắng mấy buổi?", citations: [] })] : [], next_cursor: null })),
  });
  await page.goto("/chat");
  await page.locator("[data-part=chat-thread]").waitFor();
  await page.getByRole("textbox", { name: "Câu hỏi của bạn" }).fill("Em vắng mấy buổi?");
  await page.getByRole("textbox", { name: "Câu hỏi của bạn" }).press("Enter");
  const att = page.getByRole("region", { name: "Chuyên cần" });
  await expect(att).toBeVisible();
  await expect(att).toContainText("Vắng2");
  await expect(att).toContainText("Có mặt6");
  await expect(att).toContainText("Buổi vắng gần đây12/11, 05/11");
  await expect(page.getByRole("region", { name: "Phát biểu" })).toContainText("Điểm cộng0,50");
  const body = await page.locator("[data-part=chat-thread]").innerText();
  expect(body).not.toMatch(/[{}\[\]"]|absent|present|late|excused|recorded|recent|speak_count|points_total|Kết quả/);
  await expect(page.getByRole("region", { name: "Kết quả" })).toHaveCount(0);
});
