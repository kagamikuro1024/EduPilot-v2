// Đoạn tài liệu do docling sinh có chú thích HTML (`<!-- formula-not-decoded -->`, `<!-- image -->`…): không đưa vào bài mẫu của seed.
// Bỏ chú thích (kể cả nhiều dòng); dòng chỉ có chú thích bỏ cả dòng; gộp dòng trống liền nhau.
export function stripDoclingComments(text) {
  return text
    .replace(/^[ \t]*<!--[\s\S]*?-->[ \t]*(\n|$)/gm, "") // dòng chỉ có chú thích: bỏ cả dòng
    .replace(/<!--[\s\S]*?-->/g, "")
    .split("\n")
    .map((l) => l.replace(/[ \t]+$/, ""))
    .join("\n")
    .replace(/\n{3,}/g, "\n\n")
    .trim();
}
