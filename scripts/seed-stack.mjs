// Chốt chặn «cùng stack» (DEF-R105S-05): lệnh `gateway admin` (SEED_ADMIN_CMD hoặc compose theo COMPOSE_PROJECT_NAME) có thể trỏ DB KHÁC với API_URL
// (vd shell còn COMPOSE_PROJECT_NAME=ep-qc) và ghi nhầm. Dấu nhận = id lớp 761987: uuid v7 của mỗi DB, lấy qua API và qua `admin seed-identity`.

/** Trả null khi hai id khớp, ngược lại thông báo lỗi (nêu stack đích của lệnh admin). `adminOut` là stdout của seed-identity hoặc null nếu lệnh lỗi. */
export function stackMismatch({ apiId, adminOut, adminTarget, api }) {
  const adminId = /^seed-identity: (\S+)$/m.exec(adminOut ?? "")?.[1];
  if (adminId && adminId === apiId) return null;
  return `Lệnh admin ghi vào DB KHÁC với API_URL ${api}: lớp 761987 qua API = ${apiId}, qua lệnh admin = ${adminId ?? "(không có)"}. Lệnh admin: ${adminTarget}. ` +
    "Đặt SEED_ADMIN_CMD đúng stack (hoặc COMPOSE_PROJECT_NAME / unset nó) rồi chạy lại; chưa ghi gì qua lệnh admin.";
}
