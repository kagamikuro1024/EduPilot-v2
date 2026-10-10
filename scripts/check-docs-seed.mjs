// Kiểm dữ liệu mẫu tài liệu + lịch của US-P8-03 AC17 / US-P8-01 AC17 bằng API thật.
//   node scripts/check-docs-seed.mjs            docs=… READY answer_key=1 shared=… reembedded=0 events=2/1 idempotent=ok   (--no-rerun bỏ chạy lại seed)
//   node scripts/check-docs-seed.mjs timing     giây / trang của từng tệp ≥ 5 trang (updated_at − created_at, seed chờ từng tệp xong): ≤ 1 (PDF có chữ), ≤ 4 (bản scan `Quyche`)
import { call, courses, finish, listAll, must, reseed, tok } from "./seed-check-lib.mjs";

const mode = process.argv[2] && !process.argv[2].startsWith("--") ? process.argv[2] : "all";
const noRerun = process.argv.includes("--no-rerun");
const TITLES = ["Network Security Threats", "Forecasting (QMB ch. 6b)", "Quy chế học vụ", "Đề tham khảo tuần 5", "Đáp án đề tham khảo tuần 5"];

const { c1, c2 } = await courses();
const t = await tok("teacher");
const events = async (c) => {
  const t0 = Date.now(); // đúng 62 ngày: GET calendar từ chối khoảng dài hơn
  return (await listAll(`/courses/${c}/calendar?from=${encodeURIComponent(new Date(t0 - 864e5).toISOString())}&to=${encodeURIComponent(new Date(t0 + 61 * 864e5).toISOString())}`, t)).filter((e) => e.source === "calendar_event");
};
const snap = async () => ({
  docs: (await listAll(`/courses/${c1}/documents`, t)).length, shared: (await listAll(`/courses/${c2}/documents`, t)).length,
  ev1: (await events(c1)).length, ev2: (await events(c2)).length, chunks: (await call("GET", `/courses/${c1}/documents/stats`, { token: t })).body.chunks,
});

const docs = await listAll(`/courses/${c1}/documents`, t);
if (mode === "timing") {
  for (const d of docs.filter((x) => TITLES.includes(x.title) && x.page_count >= 5)) { // PDF một trang do script sinh: thời gian cố định mỗi việc (khởi động docling + nhúng) lấn át, không phải số đo giây / trang
    const secs = (new Date(d.updated_at) - new Date(d.created_at)) / 1000;
    const perPage = secs / d.page_count;
    const limit = d.title === "Quy chế học vụ" ? 4 : 1; // bản scan đi lượt OCR
    console.log(`${d.title}: ${d.page_count} trang, ${secs.toFixed(0)} s, ${perPage.toFixed(2)} s/trang (≤ ${limit})`);
    must(perPage <= limit, `${d.title}: ${perPage.toFixed(2)} s/trang > ${limit}`);
  }
  finish("timing xong");
  process.exit(0);
}

const ready = TITLES.filter((x) => docs.find((d) => d.title === x)?.status === "READY");
must(ready.length === TITLES.length, `READY ${ready.length}/${TITLES.length}: thiếu ${TITLES.filter((x) => !ready.includes(x)).join(", ")}`);
const keys = docs.filter((d) => d.type === "ANSWER_KEY");
must(keys.length === 1 && keys[0].visible_to_students === false, `ANSWER_KEY = ${keys.length} (cần 1, ẩn với sinh viên)`);
const policy = docs.filter((d) => d.type === "COURSE_POLICY");
must(policy.length === 1, `COURSE_POLICY = ${policy.length} (cần 1)`);
// canary của đáp án không bao giờ lọt tới sinh viên: thư viện và chat
const sv = await tok("sv.gioi");
const lib = await listAll(`/courses/${c1}/library?q=CANARY-7Q2X`, sv);
must(lib.length === 0, `thư viện của sinh viên trả ${lib.length} mục cho canary`);

const sharedDocs = (await listAll(`/courses/${c2}/documents`, t)).filter((d) => d.shared_from);
must(sharedDocs.length >= 2, `tài liệu dùng chung ở lớp 2 = ${sharedDocs.length} (cần ≥ 2 bài giảng)`);
must(!sharedDocs.some((d) => d.type === "COURSE_POLICY"), "quy chế môn học không được chia sẻ");
// không nhúng lại: mọi đoạn của tài liệu chia sẻ ở lớp 2 chính là đoạn đã có ở lớp 1
let reembedded = 0;
for (const d of sharedDocs) {
  const a = new Set((await listAll(`/courses/${c1}/documents/${d.id}/chunks`, t)).map((c) => c.id));
  for (const c of await listAll(`/courses/${c2}/documents/${d.id}/chunks`, t)) if (!a.has(c.id)) reembedded++;
}
must(reembedded === 0, `đoạn nhúng lại ở lớp 2: ${reembedded}`);
const e1 = await events(c1), e2 = await events(c2);
must(e1.filter((e) => e.type === "EXAM").length === 2, `sự kiện thi lớp 1 = ${e1.length} (cần 2)`);
must(e2.length === 1 && e2[0].type === "OTHER", `sự kiện lớp 2 = ${e2.length} (cần 1 OTHER)`);

let idem = "bỏ qua (--no-rerun)";
if (!noRerun) {
  const a = await snap();
  must(reseed() === 0, "chạy lại seed.mjs lỗi");
  const b = await snap();
  idem = JSON.stringify(a) === JSON.stringify(b) ? "ok" : "KHÁC";
  must(idem === "ok", `chạy lại seed làm đổi số liệu: ${JSON.stringify(a)} → ${JSON.stringify(b)}`);
}
finish(`docs=${docs.length} READY=${ready.length} answer_key=${keys.length} shared=${sharedDocs.length} reembedded=${reembedded} events=${e1.length}/${e2.length} idempotent=${idem}`);
