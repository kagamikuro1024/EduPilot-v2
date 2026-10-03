import { expect, test } from "@playwright/test";
import { safeNext } from "../src/shared/session/safeNext";

// US-P2-02 AC13: `next` chỉ nhận đường dẫn nội bộ. Hàm thuần — không cần trình duyệt (vitest không có trong bảng thư viện của ARCHITECTURE.md).
const OK = ["/", "/chat", "/join/BX4P9TW", "/class/members?tab=pending", "/settings/llm#top", "/a/b/c?x=1&y=%20z"];
const EVIL = [
  "//evil.example", "https://evil.example", "/\\evil.example", "javascript:alert(1)", "/%2F%2Fevil.example", "\\\\evil",
  "http:evil", "/%5Cevil.example", "/a:b", "/x\u0000y", "/x\ny", "", "chat", "/" + "a".repeat(600),
];

test.describe("safeNext", () => {
  for (const v of OK) test(`giữ ${JSON.stringify(v)}`, () => expect(safeNext(v)).toBe(v));
  for (const v of EVIL) test(`bỏ ${JSON.stringify(v.slice(0, 30))}`, () => expect(safeNext(v)).toBe("/"));
  test("thiếu → /", () => {
    expect(safeNext(null)).toBe("/");
    expect(safeNext(undefined)).toBe("/");
  });
});
