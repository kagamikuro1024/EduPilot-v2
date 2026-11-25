// Chạy: node --test scripts/assignment-seed-data.test.mjs
// QC L2.11: mô tả bài tập seed không lặp tiêu đề với giao diện (tên bài là tiêu đề trang, «Đề bài» là tên khối chứa mô tả).
import test from "node:test";
import assert from "node:assert/strict";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { createBody } from "./assignment-seed-data.mjs";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");

test("mô tả BT03 không có tiêu đề «# tên bài» và «## Đề bài»; vẫn giữ phần đề và «## Yêu cầu nộp»", () => {
  const b = createBody(root);
  const headings = [...b.description.matchAll(/^#{1,6} .*/gm)].map((m) => m[0]);
  assert.deepEqual(headings, ["## Yêu cầu nộp"]);
  assert.match(b.description, /SolarWinds Orion/);
});
