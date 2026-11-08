"use client";

import dynamic from "next/dynamic";

// Ba màn Luyện đề còn dùng dữ liệu mô phỏng (`@/mock/*`, ~17 KB gzip). Next prefetch chunk của mọi route trong thanh bên, nên nạp tĩnh thì MỌI route (/chat, /me…)
// vượt ngân sách JS 256 KB của Lighthouse CI (`resource-summary:script`). Nạp lười như các màn khác: chunk chỉ tải khi vào đúng route.
export const PracticeScreen = dynamic(() => import("./PracticeScreen").then((m) => m.PracticeScreen), { loading: () => null });
export const PracticeHistory = dynamic(() => import("./PracticeHistory").then((m) => m.PracticeHistory), { loading: () => null });
export const AttemptScreen = dynamic(() => import("./AttemptScreen").then((m) => m.AttemptScreen), { loading: () => null });
