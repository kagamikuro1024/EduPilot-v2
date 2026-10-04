// `pnpm dev`: tạo .env.local nếu chưa có, kiểm Docker, dựng stack edupilot và chờ mọi service healthy.
// `pnpm dev:down|dev:status|dev:logs` = `node scripts/dev.mjs down|status|logs [tham số compose]` (cùng tên project và cổng với `pnpm dev`).
// Chạy HAI stack trên một máy (ví dụ QC cạnh dev): đặt EP_PORT_OFFSET=N (N ≠ 0) — project thành `edupilot<N>`, mọi cổng host cộng N
// (HTTPS 443+N, Mailpit 8025+N, Postgres 5433+N…), URL công khai thành https://localhost:<443+N>. Tên project đặt tay bằng COMPOSE_PROJECT_NAME.
import { copyFileSync, existsSync, readFileSync } from "node:fs";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import path from "node:path";

const sub = ["up", "down", "status", "logs"].includes(process.argv[2]) ? process.argv[2] : "up";
const extra = sub === "up" ? [] : process.argv.slice(3);

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const envFile = path.join(root, ".env.local");
const envExample = path.join(root, ".env.example");

if (!existsSync(envFile)) {
  if (!existsSync(envExample)) {
    console.error("[dev] Không tìm thấy .env.example để tạo cấu hình local.");
    process.exit(1);
  }
  copyFileSync(envExample, envFile);
  console.log("[dev] Đã tạo .env.local từ .env.example.");
}

const docker = spawnSync("docker", ["info", "--format", "{{.ServerVersion}}"], { cwd: root, encoding: "utf8" });
if (docker.error?.code === "ENOENT") {
  console.error("[dev] Chưa tìm thấy Docker. Hãy cài và khởi động Docker (hoặc colima) trước.");
  process.exit(1);
}
if (docker.status !== 0) {
  console.error("[dev] Không kết nối được Docker daemon. Hãy khởi động Docker rồi chạy lại pnpm dev.");
  if (docker.stderr) console.error(docker.stderr.trim());
  process.exit(docker.status || 1);
}

// Dev nới giới hạn theo IP để seed (MỘT IP gửi hàng nghìn yêu cầu) chạy được; không đặt ở docker-compose.test.yml nên stack thử giữ giới hạn thật.
const devLimits = { RATE_LIMIT_IP_PER_MIN: "5000", AUTH_LOGIN_IP_PER_MIN: "1000", AUTH_REGISTER_IP_PER_HOUR: "1000", AUTH_TOKEN_IP_PER_MIN: "1000" };
const env = { ...process.env };
for (const [k, v] of Object.entries(devLimits)) env[k] ||= v;

// Cổng host mặc định + EP_PORT_OFFSET (xem đầu tệp). Biến EP_PORT_* đặt tay vẫn thắng.
const offset = Number(process.env.EP_PORT_OFFSET || 0);
if (!Number.isInteger(offset) || offset < 0 || offset > 20000) {
  console.error("[dev] EP_PORT_OFFSET phải là số nguyên từ 0 đến 20000.");
  process.exit(1);
}
const basePorts = { POSTGRES: 5433, REDIS: 6380, MINIO: 9000, MINIO_CONSOLE: 9001, SMTP: 1025, MAILPIT: 8025, HTTP: 80, HTTPS: 443, FRONTEND: 3000 };
for (const [k, v] of Object.entries(basePorts)) env[`EP_PORT_${k}`] ||= String(v + offset);
const project = process.env.COMPOSE_PROJECT_NAME || (offset ? `edupilot${offset}` : "edupilot");
const httpsPort = Number(env.EP_PORT_HTTPS);
const origin = httpsPort === 443 ? "https://localhost" : `https://localhost:${httpsPort}`;
if (httpsPort !== 443) {
  // URL công khai phải mang cổng: frontend nướng NEXT_PUBLIC_API_URL lúc dựng, gateway dùng APP_PUBLIC_URL cho liên kết trong thư và CORS_ORIGINS cho trình duyệt.
  env.NEXT_PUBLIC_API_URL ||= origin;
  env.APP_PUBLIC_URL ||= origin;
  env.CORS_ORIGINS ||= `${origin},http://localhost:${env.EP_PORT_FRONTEND}`;
}
const compose = (...args) => ["compose", "--env-file", ".env.local", "-f", "docker-compose.local.yml", "-p", project, ...args];

if (sub !== "up") {
  const r = spawnSync("docker", compose(...{ down: ["down"], status: ["ps"], logs: ["logs", "-f"] }[sub], ...extra), { cwd: root, stdio: "inherit", env });
  process.exit(r.status ?? 1);
}

console.log(`[dev] Project ${project}: dựng postgres, pgbouncer, redis, minio, mailpit, migrate, gateway, worker, caddy, frontend và chờ healthy...`);
const up = spawnSync("docker", compose("up", "-d", "--build", "--remove-orphans", "--wait"), { cwd: root, stdio: "inherit", env });
if (up.error) {
  console.error(`[dev] Không thể chạy Docker Compose: ${up.error.message}`);
  process.exit(1);
}
if (up.status !== 0) process.exit(up.status ?? 1);
console.log(`[dev] Sẵn sàng — ${origin} (chứng chỉ nội bộ của Caddy: dùng \`curl -k\`) · mail http://localhost:${env.EP_PORT_MAILPIT} · minio http://localhost:${env.EP_PORT_MINIO_CONSOLE}`);

// Seed tự chạy khi DB trống và SEED_ON_EMPTY_DB=true (đặt trong .env.local hoặc môi trường). Lỗi seed KHÔNG làm hỏng `pnpm dev`.
const seedOn = (process.env.SEED_ON_EMPTY_DB ?? readEnvLocal("SEED_ON_EMPTY_DB")) === "true";
if (seedOn) {
  console.log("[dev] SEED_ON_EMPTY_DB=true — chạy scripts/seed.mjs --if-empty sau khi gateway sẵn sàng...");
  const t0 = Date.now();
  const seed = spawnSync("node", ["scripts/seed.mjs", "--if-empty"], { cwd: root, stdio: "inherit", env: { ...process.env, COMPOSE_PROJECT_NAME: project, API_URL: process.env.API_URL ?? `${origin}/api/v1`, MAILPIT_URL: process.env.MAILPIT_URL ?? `http://localhost:${env.EP_PORT_MAILPIT}` } });
  console.log(`[dev] seed mất ${Math.round((Date.now() - t0) / 1000)} s`);
  if (seed.status !== 0) console.warn("[dev] CẢNH BÁO: seed chưa xong (mã thoát " + (seed.status ?? "?") + "). Stack vẫn chạy; chạy lại bằng `pnpm seed`.");
}

function readEnvLocal(key) {
  try {
    const line = readFileSync(envFile, "utf8").split("\n").find((l) => l.startsWith(key + "="));
    return line ? line.slice(key.length + 1).trim() : undefined;
  } catch {
    return undefined;
  }
}
