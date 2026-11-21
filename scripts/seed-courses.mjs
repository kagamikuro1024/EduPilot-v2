// Tra lớp theo mã bằng `q=` (tiền tố mã lớp): DB có > 100 lớp thì lớp seed không nằm ở trang đầu của danh sách mới-nhất-trước (DEF-R4-08, DEF-R9.5-26).
export async function findCourse(call, token, code) {
  const r = await call("GET", `/admin/courses?limit=100&q=${encodeURIComponent(code)}`, { token });
  return r.body.items.find((c) => c.class_code === code) ?? null;
}
