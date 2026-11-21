import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import { crmPlan } from "./crm-plan.mjs";

// DEF-Q2-S8-02: phát biểu (điểm cộng vào điểm học phần) không đổi theo SEED_RNG; mặc định khớp seed/crm_plan.json.
test("speak cố định khi đổi SEED_RNG, điểm danh vẫn đổi", () => {
  const base = crmPlan();
  const other = crmPlan(1);
  assert.deepEqual(other.speak, base.speak);
  assert.equal(other.totals.speak, base.totals.speak);
  assert.deepEqual(other.expected, base.expected);
  assert.notDeepEqual(other.attendance, base.attendance);
});

test("kế hoạch mặc định khớp seed/crm_plan.json", () => {
  const file = JSON.parse(readFileSync(new URL("../seed/crm_plan.json", import.meta.url), "utf8"));
  assert.deepEqual(JSON.parse(JSON.stringify(crmPlan())), file);
});
