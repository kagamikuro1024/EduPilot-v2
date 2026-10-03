"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState, type ReactNode } from "react";

/** Mặc định của dự án (SRS 6.3): thử lại do apiClient lo nên `retry: false`. */
export function makeQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { staleTime: 30_000, gcTime: 5 * 60_000, retry: false, refetchOnWindowFocus: true, networkMode: "offlineFirst" },
      mutations: { retry: false, networkMode: "offlineFirst" },
    },
  });
}

export function QueryProvider({ children }: { children: ReactNode }) {
  const [client] = useState(makeQueryClient);
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}
