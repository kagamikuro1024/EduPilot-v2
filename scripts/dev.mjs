// `pnpm dev`: tạo .env.local nếu chưa có, kiểm Docker, dựng stack edupilot và chờ mọi service healthy.
import { copyFileSync, existsSync } from "node:fs";
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

console.log("[dev] Dựng postgres, pgbouncer, redis, minio, mailpit, migrate, gateway, worker, caddy, frontend và chờ healthy...");
const up = spawnSync(
  "docker",
  ["compose", "--env-file", ".env.local", "-f", "docker-compose.local.yml", "-p", "edupilot", "up", "-d", "--build", "--remove-orphans", "--wait"],
  { cwd: root, stdio: "inherit" },
);
if (up.error) {
  console.error(`[dev] Không thể chạy Docker Compose: ${up.error.message}`);
  process.exit(1);
}
if (up.status !== 0) process.exit(up.status ?? 1);
console.log("[dev] Sẵn sàng — https://localhost (chứng chỉ nội bộ của Caddy: dùng `curl -k`) · mail http://localhost:8025 · minio http://localhost:9001");
