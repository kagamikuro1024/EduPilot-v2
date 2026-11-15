import type { MetadataRoute } from "next";

// PWA tối thiểu (US-AS-09 AC6): cài vào Màn hình chính giữ khoá WebCrypto quá 7 ngày trên iOS. KHÔNG có service worker đệm API, KHÔNG có lời mời cài đặt.
// Màu: giá trị của token `--ep-red` / `--ep-canvas` (manifest không đọc được biến CSS).
export default function manifest(): MetadataRoute.Manifest {
  return {
    name: "EduPilot",
    short_name: "EduPilot",
    start_url: "/",
    scope: "/",
    display: "standalone",
    theme_color: "#CE1628",
    background_color: "#FFFFFF",
    lang: "vi",
    icons: [{ src: "/brand/logo-edupilot-mark.svg", sizes: "any", type: "image/svg+xml", purpose: "any" }],
  };
}
