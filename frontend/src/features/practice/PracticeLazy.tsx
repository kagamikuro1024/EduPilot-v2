"use client";

import dynamic from "next/dynamic";

// Ba màn Luyện đề nạp lười: Next prefetch chunk của mọi route trong thanh bên, nên nạp tĩnh thì MỌI route (/chat, /me…) vượt ngân sách JS 256 KB của Lighthouse CI
// (`resource-summary:script`). Chunk chỉ tải khi vào đúng route.
export const PracticeScreen = dynamic(() => import("./PracticeHome").then((m) => m.PracticeHome), { loading: () => null });
export const PracticeHistory = dynamic(() => import("./PracticeHistory").then((m) => m.PracticeHistory), { loading: () => null });
export const AttemptScreen = dynamic(() => import("./PracticeRun").then((m) => m.PracticeRun), { loading: () => null });
