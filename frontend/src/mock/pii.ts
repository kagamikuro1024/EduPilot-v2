// Bộ nhận diện thông tin cá nhân cho kênh công khai (Threads) — bảng ở SRS 4.3.3, bảng đó thắng mọi bộ nhận cũ.
// Không in lại giá trị tìm thấy ở bất kỳ đâu: chỉ đếm theo loại và thay bằng "[đã ẩn]".
import { STUDENTS } from "./core";

export type PiiKind = "email" | "phone" | "code" | "name" | "grade";
export type PiiHit = { kind: PiiKind; start: number; end: number };

export const REDACTED = "[đã ẩn]";

const EMAIL = /[^\s@,;]+@[^\s@,;]+\.[A-Za-z]{2,}/g;
const PHONE_CAND = /(?<![\w+])\+?(?:84|0)(?:[ .-]?\d){8,10}(?!\d)/g;
const CODE = /(?<![\d.,])\d{8}(?![\d])/g;
const NUM = "(10|\\d)(?:[.,]\\d{1,2})?";
// "8,5 điểm" · "điểm của X là 4,9" · "điểm: 7"
const GRADE_BEFORE = new RegExp(`(?<![\\d.,+\\-])(${NUM})\\s*điểm(?!\\s*danh)`, "gi");
const GRADE_AFTER = new RegExp(`điểm(?!\\s*danh)[^\\d.,;!?]{0,48}?(?:là|được|:)\\s*(${NUM})(?![\\d])`, "gi");
const SELF = /(?<![\p{L}])(em|mình|tôi)(?![\p{L}])/iu;

function normalizeName(s: string) {
  return s.normalize("NFC").toLowerCase();
}

const NAMES = [...new Set(STUDENTS.map((s) => s.name))].map((n) => ({ key: normalizeName(n) }));

function isPhone(raw: string) {
  const d = raw.replace(/\D/g, "");
  if (raw.trim().startsWith("+") && !d.startsWith("84")) return false;
  if (d.startsWith("84")) return d.length === 11 && /^[35789]/.test(d.slice(2));
  return d.length === 10 && /^0[35789]/.test(d);
}

/** Điểm hợp lệ: 0–10, không phải mã (số nguyên có số 0 đứng đầu như "03"). */
function plausibleScore(raw: string) {
  if (/^0\d/.test(raw)) return false;
  const v = Number(raw.replace(",", "."));
  return v >= 0 && v <= 10;
}

function overlaps(hits: PiiHit[], start: number, end: number) {
  return hits.some((h) => start < h.end && end > h.start);
}

export function findPii(input: string): PiiHit[] {
  const text = input.normalize("NFC");
  const hits: PiiHit[] = [];
  const add = (kind: PiiKind, start: number, end: number) => {
    if (!overlaps(hits, start, end)) hits.push({ kind, start, end });
  };

  for (const m of text.matchAll(EMAIL)) add("email", m.index!, m.index! + m[0].length);
  for (const m of text.matchAll(PHONE_CAND)) if (isPhone(m[0])) add("phone", m.index!, m.index! + m[0].length);
  for (const m of text.matchAll(CODE)) add("code", m.index!, m.index! + m[0].length);

  const lower = normalizeName(text);
  const names: Array<[number, number]> = [];
  for (const { key } of NAMES) {
    for (let at = lower.indexOf(key); at !== -1; at = lower.indexOf(key, at + 1)) {
      const end = at + key.length;
      const before = text[at - 1];
      const after = text[end];
      if ((before && /\p{L}/u.test(before)) || (after && /\p{L}/u.test(after))) continue;
      add("name", at, end);
      names.push([at, end]);
    }
  }

  // Điểm gắn danh tính: số 0–10 đi với "điểm" VÀ (em / mình / tôi hoặc họ tên) trong cùng một vế câu.
  const clauses: Array<[number, number]> = [];
  let from = 0;
  for (const m of text.matchAll(/,(?!\d)|[;!?\n]|\.(?!\d)/g)) {
    clauses.push([from, m.index!]);
    from = m.index! + 1;
  }
  clauses.push([from, text.length]);
  for (const [a, b] of clauses) {
    const part = text.slice(a, b);
    const hasId = SELF.test(part) || names.some(([s, e]) => s >= a && e <= b);
    if (!hasId) continue;
    for (const re of [GRADE_BEFORE, GRADE_AFTER]) {
      re.lastIndex = 0;
      for (const m of part.matchAll(re)) {
        const raw = m[1];
        const at = a + m.index! + m[0].indexOf(raw);
        if (plausibleScore(raw) && text[at - 1] !== "+") add("grade", at, at + raw.length);
      }
    }
  }
  return hits.sort((x, y) => x.start - y.start);
}

/** Thay mỗi mục khớp bằng "[đã ẩn]". */
export function redactPii(input: string): string {
  const text = input.normalize("NFC");
  let out = "";
  let at = 0;
  for (const h of findPii(text)) {
    out += text.slice(at, h.start) + REDACTED;
    at = h.end;
  }
  return out + text.slice(at);
}

const LABEL: Record<PiiKind, string> = {
  email: "địa chỉ email",
  phone: "số điện thoại",
  code: "mã số sinh viên",
  name: "họ tên",
  grade: "điểm gắn với một người",
};
const ORDER: PiiKind[] = ["email", "phone", "code", "name", "grade"];

/** "1 địa chỉ email, 1 số điện thoại" — đếm theo loại, không in lại giá trị. */
export function describePii(hits: PiiHit[]): string {
  return ORDER.map((k) => [k, hits.filter((h) => h.kind === k).length] as const)
    .filter(([, n]) => n > 0)
    .map(([k, n]) => `${n} ${LABEL[k]}`)
    .join(", ");
}
