"use client";

import dynamic from "next/dynamic";
import { HistoryShell, HomeShell, RunShell } from "./PracticeShell";

// Ba màn Luyện đề nạp lười: Next prefetch chunk của mọi route trong thanh bên, nên nạp tĩnh thì MỌI route (/chat, /me…) vượt ngân sách JS 256 KB của Lighthouse CI
// (`resource-summary:script`). Chunk chỉ tải khi vào đúng route; `loading` là khung có tiêu đề (PracticeShell) để chữ đầu tiên không đợi chunk.
export const PracticeScreen = dynamic(() => import("./PracticeHome").then((m) => m.PracticeHome), { loading: HomeShell });
export const PracticeHistory = dynamic(() => import("./PracticeHistory").then((m) => m.PracticeHistory), { loading: HistoryShell });
export const AttemptScreen = dynamic(() => import("./PracticeRun").then((m) => m.PracticeRun), { loading: RunShell });
