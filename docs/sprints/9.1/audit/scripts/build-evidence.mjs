// Dựng audit/evidence.md từ data/*.json + data/observations.json (quan sát tay của QC, khoá = tên tệp ảnh).
//   node build-evidence.mjs
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const HERE = path.dirname(fileURLToPath(import.meta.url));
const DATA = path.join(HERE, "../data");
const SHOTS = path.join(HERE, "../shots");
const read = (f, d) => { try { return JSON.parse(fs.readFileSync(path.join(DATA, f), "utf8")); } catch { return d; } };
const obs = read("observations.json", {});
const meta = read("meta.json", {});
const ROLES = [["anon", "Khách (chưa đăng nhập)"], ["teacher", "Giảng viên `teacher@`"], ["ta", "Trợ giảng `ta@`"], ["admin", "Quản trị `admin@`"], ["student", "Sinh viên `sv.gioi@` (lớp 761987 + 761988)"], ["student2", "Sinh viên `sv.kha@` (lớp 761987)"], ["newbie", "Sinh viên mới `sv.moi@` (chưa vào lớp)"]];
const id = (s) => s.replace(/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/g, "‹id›");
const axeCell = (a) => (!a ? "—" : a.error ? "lỗi axe" : `${a.critical}/${a.serious}/${a.moderate}/${a.minor}`);
const auto = (r) => {
  const o = [];
  if (r.fail) o.push(`KHÔNG CHỤP ĐƯỢC: ${r.fail}`);
  if (r.bad?.length) o.push(`chữ lạ trong nội dung: ${r.bad.join(", ")}`);
  if (r.tech?.length) o.push(`từ kỹ thuật hiện cho sinh viên: ${r.tech.join(", ")}`);
  if (r.width === 375 && r.off?.length) o.push(`tràn ngang: ${r.off.join("; ")}`);
  if (r.noName?.length) o.push(`điều khiển không tên: ${r.noName.join(", ")}`);
  if (r.h1 === 0) o.push("không có h1");
  if (r.h1 > 1) o.push(`${r.h1} thẻ h1`);
  if (r.ellipsis?.length) o.push(`chữ bị cắt «…»: ${r.ellipsis.map((t) => `"${t.trim()}"`).join(", ")}`);
  if (r.width === 375 && r.small24 > 0) o.push(`${r.small24} điều khiển < 24 px`);
  if (r.slowApi?.length) o.push(`API chậm: ${r.slowApi.slice(0, 2).join("; ")}`);
  if (r.contentMs === null) o.push("nội dung không hiện sau 25 s");
  if (r.url && r.state === "full" && r.role !== "anon" && r.url.split("?")[0] !== r.route.split("?")[0]) o.push(`chuyển hướng → ${id(r.url).split("?")[0]}`);
  return o;
};
const cell = (s) => String(s).replace(/\|/g, "\\|").replace(/\n/g, " ");
const link = (f) => (fs.existsSync(path.join(SHOTS, f)) ? `[ảnh](shots/${f})` : "—");

let md = "";
const P = (s = "") => { md += s + "\n"; };
const all = ROLES.flatMap(([k]) => read(`${k}.json`, []));
const nShots = fs.readdirSync(SHOTS).filter((f) => f.endsWith(".png")).length;
const sum = (rs, k) => rs.reduce((n, r) => n + (r.axe?.[k] ?? 0), 0);
const full = all.filter((r) => r.state === "full");

P("# Bằng chứng UI/UX — sprint 9.1, pha 1 (QC)");
P();
P(`Chụp ngày ${meta.date ?? "2026-11-08"} trên bản demo VPS (\`${meta.url ?? "https://v2202609419695521098.happysrv.de"}\`), trình duyệt Chromium trên Mac (\`playwright-core\` đi kèm \`playwright-cli\`), ngôn ngữ vi-VN. Chỉ quan sát, không đề xuất (đề xuất là việc của BA). **${nShots} ảnh** ở \`shots/\`, số liệu thô ở \`data/*.json\`, mã chụp ở \`scripts/\` (chạy lại được).`);
P();
P("## 1. Cách chụp và giới hạn");
P();
P("- **Tên ảnh:** `<route>__<vai>__<cỡ>.png` (cỡ 1440 = 1440×900, 375 = 375×812), ảnh dài toàn trang. Trạng thái khác: `<route>@loading|error|empty__…`; hộp thoại `<route>@dlg-<tên>__…`; focus bàn phím `<route>@focus__…`. `ID` trong tên = id thật của một bản ghi seed.");
P("- **Lớp:** mọi màn của giảng viên / trợ giảng / sinh viên mở bằng `?course=` lớp **761987** (đầy dữ liệu) trừ khi ghi khác. Ảnh `@empty` dùng lớp **761988** (ít dữ liệu). Không truyền `?course=` thì giảng viên và sinh viên `sv.gioi` rơi vào lớp 761988 (lớp ít dữ liệu).");
P("- **Tải** (`@loading`): mọi lời gọi `/api/v1/**` (trừ `/auth/`) bị giữ 8 s; ảnh chụp khung nhìn lúc ~1,8 s sau khi tải trang. **Lỗi** (`@error`): mọi lời gọi `/api/v1/**` (trừ `/auth/`) bị huỷ như mất mạng; trang vẫn nạp được. **Rỗng**: lớp 761988 và tài khoản `sv.moi`.");
P("- **Đo:** *nội dung* = ms từ lúc điều hướng tới khi `<main>` có chữ và hết khung xương (ước lượng); *LCP* theo `PerformanceObserver`; axe 4.11 với thẻ wcag2a/2aa/21aa/22aa, cột `axe` = nghiêm trọng/cao/trung bình/nhẹ (số nút). Nhiều trình duyệt chạy song song nên LCP và nội dung dao động ±vài trăm ms.");
P("- **Ảnh dài:** không dùng `fullPage` của Playwright vì nó đôi khi vẽ lệch nội dung ~24 px sang trái (đã kiểm: vị trí tiêu đề trong DOM luôn 240 px, `data/extra-layout.json`, 34 route × 2 lần; ảnh `fullPage` lệch ở ~1/5 lượt). Thay vào đó khung nhìn được kéo dài bằng chiều cao trang (tối đa 9000 px), nên sidebar và thanh điều hướng dưới nằm sát đáy ảnh. Bảng có khung cuộn riêng (Sổ điểm, Sinh viên) chỉ hiện phần trong khung.");
P("- **Không chạm dữ liệu demo:** không bấm Lưu / Duyệt / Công bố / Chốt / Xoá / Nhận / Mời / Tạo. Hộp thoại chỉ mở rồi đóng bằng Esc. Mọi request không phải GET trong lúc mở hộp thoại: xem đầu mục 4 (0 request).");
P();
P("## 2. Tổng hợp");
P();
P(`| Chỉ số | Giá trị |\n| --- | --- |\n| Lượt chụp | ${all.length} (đầy ${full.length}, tải ${all.filter((r) => r.state === "loading").length}, lỗi ${all.filter((r) => r.state === "error").length}, rỗng ${all.filter((r) => r.state === "empty").length}) |\n| Lượt không chụp được | ${all.filter((r) => r.fail).length} |\n| axe ở trạng thái đầy (nút) | nghiêm trọng ${sum(full, "critical")} · cao ${sum(full, "serious")} · trung bình ${sum(full, "moderate")} · nhẹ ${sum(full, "minor")} |\n| Lượt có lỗi / cảnh báo console (đầy) | ${full.filter((r) => r.console?.length).length} |\n| Lượt có lời gọi API ≥ 400 (đầy) | ${full.filter((r) => r.http?.length).length} |\n| Lượt tràn ngang ở 375 | ${full.filter((r) => r.width === 375 && r.off?.length).length} |`);
P();

// Gộp: luật axe và thông điệp console lặp nhiều nhất (trạng thái đầy)
const group = (xs) => { const m = new Map(); for (const x of xs) m.set(x, (m.get(x) ?? 0) + 1); return [...m].sort((a, b) => b[1] - a[1]); };
const axeRules = group(full.flatMap((r) => (r.axe?.items ?? []).map((i) => i.split(":")[0].replace(/ ×\d+$/, ""))));
const cons = group(full.flatMap((r) => (r.console ?? []).map((c) => c.replace(/https?:\/\/\S+/g, "‹url›").slice(0, 110))));
const https = group(full.flatMap((r) => (r.http ?? []).map((h) => h.replace(/[0-9a-f]{8}-[0-9a-f-]{27}/g, "‹id›").replace(/\?.*$/, ""))));
P("**Luật axe** (số lượt chụp trạng thái đầy có vi phạm): " + (axeRules.length ? axeRules.map(([k, n]) => `${k} ×${n}`).join("; ") : "không có vi phạm nào"));
P();
P("**Thông điệp console lặp nhiều nhất** (trạng thái đầy): " + (cons.length ? cons.slice(0, 8).map(([k, n]) => `«${k}» ×${n}`).join("; ") : "không có"));
P();
P("**Phản hồi HTTP ≥ 400** (trạng thái đầy): " + (https.length ? https.slice(0, 8).map(([k, n]) => `${k} ×${n}`).join("; ") : "không có"));
P();
let sec = 3;
P(`## ${sec++}. Từng màn, theo vai`);
for (const [k, title] of ROLES) {
  const rs = read(`${k}.json`, []);
  if (!rs.length) continue;
  P();
  P(`### ${title}`);
  P();
  P("**Đầy dữ liệu** (lớp 761987)");
  P();
  P("| Route | Cỡ | Console | HTTP ≥400 | axe C/S/M/m | Nội dung ms | LCP ms | Ảnh | Ghi nhận |");
  P("| --- | --- | --- | --- | --- | --- | --- | --- | --- |");
  for (const r of rs.filter((x) => x.state === "full")) {
    const notes = [...auto(r), ...(obs[r.file] ?? [])];
    P(`| \`${cell(id(r.route))}\` | ${r.width} | ${r.console?.length ? cell(r.console.slice(0, 2).join(" ¦ ")) : "0"} | ${r.http?.length ? cell(r.http.slice(0, 2).join(" ¦ ")) : "0"} | ${axeCell(r.axe)} | ${r.contentMs ?? "—"} | ${r.lcp || "—"} | ${link(r.file)} | ${cell(notes.join(" · "))} |`);
  }
  for (const [st, label] of [["loading", "**Đang tải** (API chậm 8 s)"], ["error", "**Lỗi** (mất mạng)"], ["empty", "**Rỗng / ít dữ liệu** (lớp 761988)"]]) {
    const xs = rs.filter((x) => x.state === st);
    if (!xs.length) continue;
    P();
    P(label);
    P();
    P(st === "loading" ? "| Route | Ảnh | Ghi nhận |\n| --- | --- | --- |" : "| Route | Console | axe C/S/M/m | Ảnh | Ghi nhận |\n| --- | --- | --- | --- | --- |");
    for (const r of xs) {
      const notes = [...(st === "loading" ? [] : auto(r).filter((n) => !n.startsWith("chuyển hướng"))), ...(obs[r.file] ?? [])];
      P(st === "loading" ? `| \`${cell(id(r.route))}\` | ${link(r.file)} | ${cell((r.fail ? [`KHÔNG CHỤP ĐƯỢC: ${r.fail}`] : notes).join(" · "))} |` : `| \`${cell(id(r.route))}\` | ${r.console?.length ? cell(r.console.slice(0, 1).join("")) : "0"} | ${axeCell(r.axe)} | ${link(r.file)} | ${cell(notes.join(" · "))} |`);
    }
  }
}

const it = read("interact.json", null);
if (it) {
  P();
  P(`## ${sec++}. Hộp thoại, drawer, menu, focus bàn phím`);
  P();
  P(`Request không phải GET (ngoài \`/auth/\`) trong cả đợt mở hộp thoại / focus: **${it.writes.length}**${it.writes.length ? " — " + it.writes.join("; ") : ""}.`);
  P();
  P("| Vai | Route | Cỡ | Mở | Esc đóng | Chuyển động lúc mở | axe C/S/M/m | Ảnh | Ghi nhận |");
  P("| --- | --- | --- | --- | --- | --- | --- | --- | --- |");
  for (const a of it.actions) P(`| ${a.role} | \`${cell(id(a.route))}\` | ${a.width} | ${a.error ? "KHÔNG — " + cell(a.error) : a.opened ? `có lớp phủ / menu (${a.name})` : `không có lớp phủ (${a.name}) — mở nội tuyến hoặc đổi trang, xem ảnh`} | ${a.closedByEscape ?? "—"} | ${cell((a.motion ?? []).join(", ") || "không có")} | ${axeCell(a.axe)} | ${link(a.file)} | ${cell([...((a.console ?? []).filter((c) => !/preloaded using link preload/.test(c)).slice(0, 1).map((c) => `console: ${c}`)), ...(obs[a.file] ?? [])].join(" · "))} |`);
  P();
  P("**Focus bàn phím** (Tab 10 lần từ đầu trang; `vòng` = có outline / shadow trên phần tử đang focus)");
  P();
  for (const f of it.focus) {
    P(`- \`${id(f.route)}\` · ${f.role} · ${f.width} px · ${link(f.file)}${f.error ? " · LỖI " + f.error : ""}`);
    P("  " + f.stops.map((s, i) => `${i + 1}. ${s}`).join("  \n  "));
    for (const o of obs[f.file] ?? []) P(`  - ${o}`);
  }
}

const ld = read("landing.json", null);
if (ld) {
  P();
  P(`## ${sec++}. Trang giới thiệu — chuyển động và ngân sách (khách)`);
  P();
  P("| Cỡ | `prefers-reduced-motion` | JS nén KB | JS giải nén KB | Tổng KB | LCP ms | Luật reduced-motion trong CSS | Scroll-driven CSS | Video | Số animation ngay sau khi cuộn → sau 1 s (mỗi bước ~800 px) |");
  P("| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |");
  for (const r of ld) P(`| ${r.width} | ${r.reducedMotion} | ${r.jsKB} | ${r.jsRawKB ?? "—"} | ${r.totalKB} | ${r.static.lcp} | ${r.static.rmRules} | ${r.static.scrollTimeline ? "có" : "không"} | ${r.static.videos.length} | ${r.steps.map((s) => `${s.animsJustAfterScroll}→${s.animsAfter1s}`).join(", ")} |`);
  P();
  for (const o of obs["__landing"] ?? []) P(`- ${o}`);
}

const ex = read("extra.json", null);
if (ex) { P(); P(`## ${sec++}. Ghi chú thêm`); P(); for (const l of ex) P(`- ${l}`); }
fs.writeFileSync(path.join(HERE, "../evidence.md"), md);
console.log("evidence.md", md.length, "ký tự");
