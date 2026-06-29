import { copyFileSync, existsSync } from "node:fs";
import { spawn, spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import path from "node:path";

const projectRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const envFile = path.join(projectRoot, ".env.local");
const envExample = path.join(projectRoot, ".env.example");

if (!existsSync(envFile)) {
  if (!existsSync(envExample)) {
    console.error("[dev] Không tìm thấy .env.example để tạo cấu hình local.");
    process.exit(1);
  }

  copyFileSync(envExample, envFile);
  console.log("[dev] Đã tạo .env.local từ .env.example.");
  console.log("[dev] Hãy cập nhật OPENAI_API_KEY trong .env.local để dùng đầy đủ tính năng AI.\n");
}

const dockerCheck = spawnSync("docker", ["info", "--format", "{{.ServerVersion}}"], {
  cwd: projectRoot,
  encoding: "utf8",
  shell: false,
});

if (dockerCheck.error?.code === "ENOENT") {
  console.error("[dev] Chưa tìm thấy Docker. Hãy cài và khởi động Docker Desktop trước.");
  process.exit(1);
}

if (dockerCheck.status !== 0) {
  console.error("[dev] Không kết nối được Docker daemon. Hãy mở Docker Desktop rồi chạy lại pnpm dev.");
  if (dockerCheck.stderr) console.error(dockerCheck.stderr.trim());
  process.exit(dockerCheck.status || 1);
}

console.log("[dev] Khởi động Frontend, Java Backend, Python AI, PostgreSQL, Redis và Jaeger...");
console.log("[dev] Frontend: http://localhost:3000 | Backend: http://localhost:8080 | AI: http://localhost:8000\n");

const compose = spawn(
  "docker",
  [
    "compose",
    "--env-file",
    ".env.local",
    "-f",
    "docker-compose.local.yml",
    "up",
    "--build",
    "--remove-orphans",
  ],
  {
    cwd: projectRoot,
    env: { ...process.env, COMPOSE_PROJECT_NAME: "a20-app-128" },
    stdio: "inherit",
    shell: false,
  },
);

compose.on("error", (error) => {
  console.error(`[dev] Không thể chạy Docker Compose: ${error.message}`);
  process.exit(1);
});

compose.on("exit", (code, signal) => {
  if (signal) console.error(`[dev] Docker Compose đã dừng bởi signal ${signal}.`);
  process.exit(code ?? 1);
});
