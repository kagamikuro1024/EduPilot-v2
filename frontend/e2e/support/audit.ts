import { pathToFileURL } from "node:url";
import { resolve } from "node:path";

// Đoạn Console AUDIT / TOUCH / LEFT nguyên văn của QC 1.5 (docs/sprints/1.5/qc/scripts/audit.mjs).
const file = resolve(__dirname, "../../../docs/sprints/1.5/qc/scripts/audit.mjs");

export async function loadAudit(): Promise<{ AUDIT_SRC: string; TOUCH_SRC: string; LEFT_SRC: string }> {
  const m = await import(pathToFileURL(file).href);
  return { AUDIT_SRC: m.AUDIT_SRC, TOUCH_SRC: m.TOUCH_SRC, LEFT_SRC: m.LEFT_SRC };
}

type AuditResult = { ox: number; cut: string[]; ell: string[] };
/**
 * AUDIT_SRC của QC 1.5 + loại chữ của <option> khỏi `cut`: Chromium của Playwright báo hình chữ nhật 0 × 0 cho <option> trong <select> đóng nên
 * phép đo tưởng chúng "bị cắt" (Chrome của QC không); chúng không hiển thị riêng nên không phải lỗi bố cục. Mọi phép đo khác giữ nguyên.
 */
export async function runAudit(page: import("@playwright/test").Page, auditSrc: string): Promise<AuditResult> {
  const a = (await page.evaluate(auditSrc)) as AuditResult;
  const opts = new Set(await page.evaluate(() => Array.from(document.querySelectorAll("main option, header option")).map((o) => (o.textContent ?? "").trim().slice(0, 30))));
  return { ...a, cut: a.cut.filter((t) => !opts.has(t)) };
}
