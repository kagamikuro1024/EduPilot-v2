"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/shared/data";

// Hợp đồng thật: backend-go/api/openapi.yaml (`/me/sessions*`, `/me/password`).
export type DeviceSession = { id: string; current: boolean; device_label: string; ip_masked: string; created_at: string; last_used_at: string };

export const SESSIONS_KEY = ["me", "sessions"] as const;

export const useSessions = () =>
  useQuery({ queryKey: SESSIONS_KEY, queryFn: async ({ signal }) => (await apiClient.get<{ items: DeviceSession[] }>("/me/sessions", { signal })).data.items });

/** Không làm mới danh sách ở đây: hàng vừa đăng xuất còn hiện dòng tĩnh tại chỗ; `DevicesSection` làm mới khi dòng đó tự biến. */
export function useRevokeSession() {
  return useMutation({ mutationFn: (id: string) => apiClient.delete(`/me/sessions/${id}`) });
}

export function useRevokeOthers() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => (await apiClient.delete<{ revoked: number }>("/me/sessions")).data,
    onSettled: () => qc.invalidateQueries({ queryKey: SESSIONS_KEY }),
  });
}

export function useChangePassword() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (b: { current_password: string; new_password: string }) => apiClient.post("/me/password", b),
    onSuccess: () => qc.invalidateQueries({ queryKey: SESSIONS_KEY }),
  });
}
