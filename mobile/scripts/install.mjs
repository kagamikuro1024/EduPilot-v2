// Cài APK release đã dựng lên thiết bị / emulator đang nối (adb).
import { execFileSync } from "node:child_process";
const apk = "android/app/build/outputs/apk/release/app-release.apk";
execFileSync("adb", ["install", "-r", apk], { stdio: "inherit" });
