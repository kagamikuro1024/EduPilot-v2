import type { MetadataRoute } from "next";
import { BRAND } from "@/shared/styles/brand";

// PWA tối thiểu (US-AS-09 AC6): cài vào Màn hình chính giữ khoá WebCrypto quá 7 ngày trên iOS. KHÔNG có service worker đệm API, KHÔNG có lời mời cài đặt.
export default function manifest(): MetadataRoute.Manifest {
  return {
    name: "EduPilot",
    short_name: "EduPilot",
    start_url: "/",
    scope: "/",
    display: "standalone",
    theme_color: BRAND.red,
    background_color: BRAND.canvas,
    lang: "vi",
    icons: [
      { src: "/brand/icon-192.png", sizes: "192x192", type: "image/png", purpose: "any" },
      { src: "/brand/icon-512.png", sizes: "512x512", type: "image/png", purpose: "any" },
      { src: "/brand/logo-edupilot-mark.svg", sizes: "any", type: "image/svg+xml", purpose: "any" },
    ],
  };
}
