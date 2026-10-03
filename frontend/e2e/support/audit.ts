import { pathToFileURL } from "node:url";
import { resolve } from "node:path";

// Đoạn Console AUDIT / TOUCH / LEFT nguyên văn của QC 1.5 (docs/sprints/1.5/qc/scripts/audit.mjs).
const file = resolve(__dirname, "../../../docs/sprints/1.5/qc/scripts/audit.mjs");

export async function loadAudit(): Promise<{ AUDIT_SRC: string; TOUCH_SRC: string; LEFT_SRC: string }> {
  const m = await import(pathToFileURL(file).href);
  return { AUDIT_SRC: m.AUDIT_SRC, TOUCH_SRC: m.TOUCH_SRC, LEFT_SRC: m.LEFT_SRC };
}
