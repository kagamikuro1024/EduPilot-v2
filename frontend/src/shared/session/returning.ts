/** Dấu "máy này đã từng đăng nhập" (cờ, không phải bí mật): chỉ để trang chủ chọn khung vẽ đầu tiên trước khi biết phiên. */
export const RETURNING_KEY = "ep_returning";

export function setReturning(on: boolean) {
  try {
    if (on) localStorage.setItem(RETURNING_KEY, "1");
    else localStorage.removeItem(RETURNING_KEY);
  } catch {
    /* storage bị chặn: bỏ qua, trang chủ vẫn đúng chỉ khác khung vẽ đầu */
  }
  if (typeof document !== "undefined") {
    if (on) document.documentElement.dataset.returning = "1";
    else delete document.documentElement.dataset.returning;
  }
}
