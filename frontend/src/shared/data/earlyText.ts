/**
 * Chữ thô cho ô sớm từ Markdown của đề / câu hỏi — KHÔNG phải bộ vẽ (không đậm / bảng / khối mã): bỏ chú thích HTML, dấu tiêu đề `#`, dấu nhấn, hàng gạch bảng, rào mã;
 * mục danh sách thành «• ». Hàm tự chứa (không tham chiếu ngoài) vì được `toString()` thẳng vào mã nội tuyến; khung thật thay ô này bằng `Markdown` đầy đủ.
 */
export function earlyText(src: string): string {
  return src
    .replace(/\r\n?/g, "\n")
    .replace(/<!--[\s\S]*?(-->|$)/g, "")
    .split("\n")
    .filter((l) => !/^\s*```/.test(l) && !/^\s*\|?\s*:?-{2,}:?\s*(\|\s*:?-{2,}:?\s*)*\|?\s*$/.test(l))
    .map((l) => l.replace(/^\s{0,3}#{1,6}\s+/, "").replace(/^\s*[-*]\s+/, "• ").replace(/\*\*?|`/g, ""))
    .join("\n")
    .replace(/\n{3,}/g, "\n\n")
    .trim();
}
