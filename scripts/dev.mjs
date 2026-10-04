// `pnpm dev`: tạo .env.local nếu chưa có, kiểm Docker, dựng stack edupilot và chờ mọi service healthy.
import { copyFileSync, existsSync, readFileSync } from "node:fs";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import path from "node:path";

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

console.log("[dev] Dựng postgres, pgbouncer, redis, minio, mailpit, migrate, gateway, worker, caddy, frontend và chờ healthy...");
const up = spawnSync(
  "docker",
  ["compose", "--env-file", ".env.local", "-f", "docker-compose.local.yml", "-p", "edupilot", "up", "-d", "--build", "--remove-orphans", "--wait"],
  { cwd: root, stdio: "inherit", env },
);
if (up.error) {
  console.error(`[dev] Không thể chạy Docker Compose: ${up.error.message}`);
  process.exit(1);
}
if (up.status !== 0) process.exit(up.status ?? 1);
console.log("[dev] Sẵn sàng — https://localhost (chứng chỉ nội bộ của Caddy: dùng `curl -k`) · mail http://localhost:8025 · minio http://localhost:9001");

// Seed tự chạy khi DB trống và SEED_ON_EMPTY_DB=true (đặt trong .env.local hoặc môi trường). Lỗi seed KHÔNG làm hỏng `pnpm dev`.
const seedOn = (process.env.SEED_ON_EMPTY_DB ?? readEnvLocal("SEED_ON_EMPTY_DB")) === "true";
if (seedOn) {
  console.log("[dev] SEED_ON_EMPTY_DB=true — chạy scripts/seed.mjs --if-empty sau khi gateway sẵn sàng...");
  const t0 = Date.now();
  const seed = spawnSync("node", ["scripts/seed.mjs", "--if-empty"], { cwd: root, stdio: "inherit", env: process.env });
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
