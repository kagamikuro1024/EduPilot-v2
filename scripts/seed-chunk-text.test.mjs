// node --test scripts/seed-chunk-text.test.mjs — bài mẫu của seed không còn chú thích HTML của docling.
import { test } from "node:test";
import assert from "node:assert/strict";
import { stripDoclingComments } from "./seed-chunk-text.mjs";

test("bỏ <!-- formula-not-decoded --> và chú thích nhiều dòng, giữ chữ và danh sách", () => {
  assert.equal(stripDoclingComments("Tiêu đề\n- a\n<!-- formula-not-decoded -->\n- b <!-- image --> c\n<!-- nhiều\ndòng -->\nhết"), "Tiêu đề\n- a\n- b  c\nhết");
});
test("không chú thích thì giữ nguyên; chỉ có chú thích thì rỗng", () => {
  assert.equal(stripDoclingComments("a\n\nb"), "a\n\nb");
  assert.equal(stripDoclingComments("<!-- formula-not-decoded -->"), "");
});
