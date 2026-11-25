// ERR-09c/09e + MSG-06 phần API: trong lúc 429, mã ĐÚNG vẫn bị 429; gateway trả Retry-After.
import { test } from "@playwright/test";
import { BASE, C, check, closeWindow, openWindow, token } from "./qc-mob-lib";

test("ERR-09c API: sáu mã sai → 429 (có Retry-After), mã đúng trong lúc 429 vẫn 429", async () => {
  const t = await token("sv.qc-b");
  const post = async (code: string) => {
    const r = await fetch(`${BASE}/api/v1/courses/${C.QCMOB1}/checkins`, {
      method: "POST",
      headers: { "content-type": "application/json", authorization: `Bearer ${t}`, "Idempotency-Key": crypto.randomUUID() },
      body: JSON.stringify({ code }),
    });
    return { s: r.status, ra: r.headers.get("retry-after"), body: (await r.text()).slice(0, 120) };
  };
  const st: number[] = [];
  let last = { s: 0, ra: null as string | null, body: "" };
  for (let i = 1; i <= 6; i++) {
    last = await post(`BBBBB${i}`);
    st.push(last.s);
  }
  check("TC-MOB-ERR-09e/API lần sai thứ 6 → 429 kèm Retry-After", last.s === 429 && Number(last.ra ?? 0) >= 1, `st=${st.join(",")} retry-after=${last.ra} body=${last.body}`);

  const w = await openWindow(C.QCMOB1);
  const d = await fetch(`${BASE}/api/v1/courses/${C.QCMOB1}/sessions/${w.sid}/checkin-windows/${w.wid}/display`, { headers: { authorization: `Bearer ${t}` } });
  void d;
  const good = await token("teacher").then(async (tt) => {
    const r = await fetch(`${BASE}/api/v1/courses/${C.QCMOB1}/sessions/${w.sid}/checkin-windows/${w.wid}/display`, { headers: { authorization: `Bearer ${tt}` } });
    return ((await r.json()) as { short_code: string }).short_code.replace(/\s/g, "");
  });
  const g = await post(good);
  check("TC-MOB-ERR-09c/API mã ĐÚNG trong lúc 429 vẫn 429, không ghi lượt", g.s === 429, `status=${g.s} retry-after=${g.ra} code=${good}`);
  await closeWindow(C.QCMOB1, w.wid).catch(() => undefined);
});
