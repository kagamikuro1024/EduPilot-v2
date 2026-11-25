// node --test scripts/seed-chunk-text.test.mjs — bài mẫu của seed không còn chú thích HTML của docling.
import { test } from "node:test";
import assert from "node:assert/strict";
import { cleanThreadBody, cleanThreadTitle, stripDoclingComments } from "./seed-chunk-text.mjs";

test("bỏ <!-- formula-not-decoded --> và chú thích nhiều dòng, giữ chữ và danh sách", () => {
  assert.equal(stripDoclingComments("Tiêu đề\n- a\n<!-- formula-not-decoded -->\n- b <!-- image --> c\n<!-- nhiều\ndòng -->\nhết"), "Tiêu đề\n- a\n- b  c\nhết");
});
test("không chú thích thì giữ nguyên; chỉ có chú thích thì rỗng", () => {
  assert.equal(stripDoclingComments("a\n\nb"), "a\n\nb");
  assert.equal(stripDoclingComments("<!-- formula-not-decoded -->"), "");
});

test("tiêu đề: bỏ ký tự đầu dòng của PDF; thân: gạch đầu dòng lạ thành «- »", () => {
  assert.equal(cleanThreadTitle("· Phreaker"), "Phreaker");
  assert.equal(cleanThreadTitle("– - Tiêu đề thường"), "Tiêu đề thường");
  assert.equal(cleanThreadTitle("Không đầu dòng"), "Không đầu dòng");
  assert.equal(cleanThreadTitle("■ Example: phishing"), "Example: phishing");
  assert.equal(cleanThreadTitle("▪ ● ○ Tiêu đề"), "Tiêu đề");
  assert.equal(cleanThreadBody("Mở đầu\n– ý một\n· ý hai\n  • ý ba\n■ ý bốn\n- đã chuẩn\nA – B giữa dòng"), "Mở đầu\n- ý một\n- ý hai\n  - ý ba\n- ý bốn\n- đã chuẩn\nA – B giữa dòng");
});
