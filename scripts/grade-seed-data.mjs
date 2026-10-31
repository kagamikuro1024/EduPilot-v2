// Dữ liệu seed sổ điểm (US-S8-SEED): đọc inputs-lop1.csv của BA và dựng tệp XLSX để nhập bằng API thật.
// XLSX tối thiểu tự dựng (zip kiểu `deflate` + XML của SpreadsheetML, chuỗi nội tuyến): không thêm thư viện ngoài bảng (Q15).
import { readFileSync } from "node:fs";
import path from "node:path";
import { deflateRawSync } from "node:zlib";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
export const XLSX_MIME = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet";

/** Các dòng của `inputs-lop1.csv` (BA): account, student_code, midterm, weekly_exam_t9, final_exam, …; ô trống → "". */
export function lop1Inputs() {
  const [head, ...rows] = readFileSync(path.join(root, "docs/specs/FEAT-gradebook/fixtures/inputs-lop1.csv"), "utf8").trim().split("\n");
  const cols = head.split(",");
  return rows.map((l) => Object.fromEntries(l.split(",").map((v, i) => [cols[i], v])));
}

const CRC = (() => {
  const t = new Uint32Array(256);
  for (let n = 0; n < 256; n++) {
    let c = n;
    for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    t[n] = c >>> 0;
  }
  return t;
})();
const crc32 = (b) => {
  let c = 0xffffffff;
  for (const x of b) c = CRC[(c ^ x) & 0xff] ^ (c >>> 8);
  return (c ^ 0xffffffff) >>> 0;
};

function zip(files) {
  const parts = [], central = [];
  let off = 0;
  for (const [name, text] of files) {
    const raw = Buffer.from(text, "utf8"), data = deflateRawSync(raw), nm = Buffer.from(name), crc = crc32(raw);
    const lh = Buffer.alloc(30);
    lh.writeUInt32LE(0x04034b50, 0); lh.writeUInt16LE(20, 4); lh.writeUInt16LE(0x0800, 6); lh.writeUInt16LE(8, 8);
    lh.writeUInt32LE(crc, 14); lh.writeUInt32LE(data.length, 18); lh.writeUInt32LE(raw.length, 22); lh.writeUInt16LE(nm.length, 26);
    parts.push(lh, nm, data);
    const ch = Buffer.alloc(46);
    ch.writeUInt32LE(0x02014b50, 0); ch.writeUInt16LE(20, 4); ch.writeUInt16LE(20, 6); ch.writeUInt16LE(0x0800, 8); ch.writeUInt16LE(8, 10);
    ch.writeUInt32LE(crc, 16); ch.writeUInt32LE(data.length, 20); ch.writeUInt32LE(raw.length, 24); ch.writeUInt16LE(nm.length, 28); ch.writeUInt32LE(off, 42);
    central.push(ch, nm);
    off += lh.length + nm.length + data.length;
  }
  const cd = Buffer.concat(central), end = Buffer.alloc(22);
  end.writeUInt32LE(0x06054b50, 0); end.writeUInt16LE(files.length, 8); end.writeUInt16LE(files.length, 10); end.writeUInt32LE(cd.length, 12); end.writeUInt32LE(off, 16);
  return Buffer.concat([...parts, cd, end]);
}

const esc = (s) => String(s).replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
const colName = (i) => String.fromCharCode(65 + i);

/** Một trang tính từ mảng dòng; chuỗi → chuỗi nội tuyến, số → ô số. */
export function xlsx(rows) {
  const sheet = rows
    .map((r, i) => `<row r="${i + 1}">${r.map((v, j) => (typeof v === "number" ? `<c r="${colName(j)}${i + 1}"><v>${v}</v></c>` : `<c r="${colName(j)}${i + 1}" t="inlineStr"><is><t>${esc(v)}</t></is></c>`)).join("")}</row>`)
    .join("");
  const x = '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>';
  return zip([
    ["[Content_Types].xml", `${x}<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>`],
    ["_rels/.rels", `${x}<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`],
    ["xl/workbook.xml", `${x}<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Điểm" sheetId="1" r:id="rId1"/></sheets></workbook>`],
    ["xl/_rels/workbook.xml.rels", `${x}<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`],
    ["xl/worksheets/sheet1.xml", `${x}<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>${sheet}</sheetData></worksheet>`],
  ]);
}

/** Tệp nhập một cột điểm của lớp 1 (`column` = tên cột trong CSV: midterm | final_exam). Dòng 1 là tiêu đề; điểm là ô số. */
export function lop1Column(column, title) {
  return xlsx([["MSSV", "Họ tên", title], ...lop1Inputs().map((r) => [r.student_code, r.account, Number(r[column])])]);
}

/** Tệp lỗi cố ý (chỉ xem trước, không ghi): MSSV lạ, điểm ngoài thang, trùng MSSV, điểm không phải số, MSSV trống kèm điểm. */
export function lop1Faulty() {
  const [a, b] = lop1Inputs();
  return xlsx([["MSSV", "Họ tên", "Điểm giữa kỳ"], [a.student_code, a.account, 9], ["99999999", "Không có trong lớp", 5], [b.student_code, b.account, 11], [a.student_code, a.account, 7], ["20229010", "x", "tám"], ["", "", 4]]);
}
