// Dữ liệu của `cmd/mock-graph` (US-P7-11 AC4): sinh từ docs/specs/FEAT-assignments/fixtures/teams-plan.csv, KHÔNG cần stack / DB.
//   node scripts/seed.mjs --mock-graph        (ghi backend-go/testdata/mock-graph/*.json + files/)
// Xác định: cùng đầu vào → cùng đầu ra (id bài nộp = UUID v5 từ tài khoản; giờ nộp = hạn BT03 + hours_vs_due).
import { createHash } from "node:crypto";
import { mkdirSync, readFileSync, writeFileSync, rmSync } from "node:fs";
import path from "node:path";

const FIX = "docs/specs/FEAT-assignments/fixtures";
export const META = {
  tenant_id: "8b6f1d2c-4d57-4c0a-9a43-2f0f5b9a7c11",
  client_id: "5e1d3a9c-7a31-4b2e-8f10-3c6d2a7f9b44",
  client_secret: "mock-graph-client-secret-not-a-real-secret",
  class_id: "37d99af7-cfc5-4e3b-8566-f7d40e4a2070",
  assignment_id: "4cc928e3-666c-4360-8688-a15776ce53b4",
};
const DUE = Date.parse("2026-12-01T16:59:00Z");
const SAMPLE_FILE = { S1: "s1-tot.md", S2: "s2-kha.md", S3: "s3-kem.md", S4: "s4-trong.md", S5: "s5-lac-de.md", S6: "s6-injection.md", S7: "s7-lech-hai-luot.md" };

// UUID v5 (namespace cố định) — không phụ thuộc thư viện.
function uuid5(name) {
  const ns = Buffer.from("6ba7b8109dad11d180b400c04fd430c8", "hex");
  const h = createHash("sha1").update(ns).update(name).digest();
  h[6] = (h[6] & 0x0f) | 0x50;
  h[8] = (h[8] & 0x3f) | 0x80;
  const x = h.subarray(0, 16).toString("hex");
  return `${x.slice(0, 8)}-${x.slice(8, 12)}-${x.slice(12, 16)}-${x.slice(16, 20)}-${x.slice(20)}`;
}

const iso = (ms) => new Date(ms).toISOString().replace(/\.\d{3}Z$/, ".0000000Z");
const nouser = { application: null, device: null, user: { id: null, displayName: null } };
const by = (id) => ({ application: null, device: null, user: { id, displayName: null } });

export function writeMockGraph(root) {
  const rows = readFileSync(path.join(root, FIX, "teams-plan.csv"), "utf8").trim().split("\n").slice(1).map((l) => {
    const [account, id, mail, status, sample, hours] = l.split(",");
    return { account, id, mail, status, sample, hours: hours === "" ? null : Number(hours) };
  });
  const out = path.join(root, "backend-go/testdata/mock-graph");
  rmSync(out, { recursive: true, force: true });
  mkdirSync(path.join(out, "files"), { recursive: true });
  const w = (name, obj) => writeFileSync(path.join(out, name), JSON.stringify(obj, null, 2) + "\n");

  const members = rows.map((r) => {
    const name = r.account.startsWith("(") ? "Người Ngoài Lớp" : r.account;
    return { id: r.id, accountEnabled: true, displayName: name, givenName: name.split(" ")[0], surname: name.split(" ").slice(1).join(" ") || name, mail: r.mail, userPrincipalName: r.mail,
      userType: "Member", primaryRole: "student", onPremisesInfo: { immutableId: null }, student: {} };
  });
  const subs = rows.map((r) => {
    const sid = uuid5("submission:" + r.id);
    const at = r.hours === null ? null : iso(DUE + r.hours * 3600_000);
    const submitted = r.status === "submitted" || r.status === "returned";
    const rec = {
      status: r.status, submittedDateTime: submitted ? at : null, unsubmittedDateTime: null, returnedDateTime: r.status === "returned" ? iso(DUE + (r.hours + 24) * 3600_000) : null,
      reassignedDateTime: r.status === "returned" ? iso(DUE + (r.hours + 24) * 3600_000) : null, resourcesFolderUrl: submitted ? `/files/folders/${sid}` : null,
      webUrl: `https://teams.microsoft.com/l/entity/66aeee93-507d-479a-a3ef-8f494af43945/classroom?submissionId=${sid}`, id: sid,
      recipient: { "@odata.type": "#microsoft.graph.educationSubmissionIndividualRecipient", userId: r.id },
      submittedBy: submitted ? by(r.id) : by(r.id), unsubmittedBy: nouser, returnedBy: r.status === "returned" ? by(uuid5("teacher")) : nouser, reassignedBy: r.status === "returned" ? by(uuid5("teacher")) : nouser,
    };
    const file = r.sample ? SAMPLE_FILE[r.sample] : null;
    const resource = (kind) => ({ assignmentResourceUrl: null, id: uuid5(`${kind}:${r.id}`), resource: { "@odata.type": "#microsoft.graph.educationFileResource", displayName: `bai-${r.account.replace(/[()\s]/g, "")}.md`,
      createdDateTime: iso(DUE + (r.hours ?? -48) * 3600_000 - 600_000), lastModifiedDateTime: at ?? iso(DUE - 3600_000), createdBy: by(r.id), lastModifiedBy: by(r.id), fileUrl: `/files/${sid}` } });
    rec.submittedResources = r.status === "working" ? [] : [resource("submitted")];
    rec.resources = r.status === "working" ? [resource("working")] : [];
    return { account: r.account, file, rec };
  });
  for (const s of subs) {
    if (s.file) writeFileSync(path.join(out, "files", `${s.rec.id}.md`), readFileSync(path.join(root, FIX, "submissions", s.file)));
  }
  w("meta.json", META);
  w("classes.json", [{ id: META.class_id, description: "An ninh mạng — lớp mẫu", classCode: "INT1006-01", createdBy: { user: { displayName: "TS. Lê Thu Hà", id: uuid5("teacher") } }, displayName: "An ninh mạng", externalId: META.class_id, externalName: "An ninh mạng", externalSource: "sis", mailNickname: "edupilot.local" }]);
  w("members.json", members);
  w("assignments.json", [{ id: META.assignment_id, displayName: "Bài tập 03 — Phân tích một vụ tấn công thực tế", status: "assigned", dueDateTime: iso(DUE), classId: META.class_id }]);
  w("submissions.json", subs.map((s) => ({ account: s.account, ...s.rec })));
  console.log(`mock-graph: ${members.length} thành viên, ${subs.length} bài nộp → ${path.relative(root, out)}`);
}
