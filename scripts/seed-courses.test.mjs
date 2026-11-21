import assert from "node:assert/strict";
import test from "node:test";
import { findCourse } from "./seed-courses.mjs";

// DEF-R9.5-26: lớp seed nằm ngoài 100 dòng đầu vẫn tìm được; không có thì null.
test("findCourse tra theo q=, không phụ thuộc trang đầu", async () => {
  const all = [...Array.from({ length: 150 }, (_, i) => ({ class_code: `QC${i}`, id: `i${i}` })), { class_code: "761987", id: "a" }];
  const call = async (_m, route) => {
    const u = new URL(route, "http://x");
    const q = u.searchParams.get("q");
    return { body: { items: (q ? all.filter((c) => c.class_code.startsWith(q)) : all.slice(0, 100)).slice(0, Number(u.searchParams.get("limit")) || 100) } };
  };
  assert.equal((await findCourse(call, "t", "761987")).id, "a");
  assert.equal(await findCourse(call, "t", "761988"), null);
});
