import axios, { type AxiosError } from "axios";
import { useAuthStore } from "@/store/authStore";

const baseURL = (process.env.NEXT_PUBLIC_JAVA_API_URL?.trim() || "http://localhost:8080").replace(/\/$/, "");
console.log("[javaClient] Resolved baseURL:", baseURL);

/** Avoid `{}` in Next/Turbopack dev overlay: JSON.stringify drops keys whose values are undefined. */
function safeResponseDataPreview(data: unknown): string | null {
  if (data === null || data === undefined) {
    return null;
  }
  if (typeof data === "string") {
    return data.length > 800 ? `${data.slice(0, 800)}…` : data;
  }
  try {
    return JSON.stringify(data);
  } catch {
    return String(data);
  }
}

function logApiFailure(axiosError: AxiosError | null, error: unknown): void {
  if (axiosError) {
    const status = axiosError.response?.status ?? null;
    const preview = safeResponseDataPreview(axiosError.response?.data);
    const line = [
      `[javaClient] ${String(axiosError.config?.method ?? "?").toUpperCase()} ${axiosError.config?.url ?? "(no url)"}`,
      status != null ? `HTTP ${status}` : "no response",
      axiosError.message || "Axios error",
      preview ? `body: ${preview}` : "",
    ]
      .filter(Boolean)
      .join(" | ");
    console.error(line);
    return;
  }
  const msg = error instanceof Error ? error.message : String(error);
  console.error(`[javaClient] non-Axios error: ${msg}`);
}

const javaClient = axios.create({
  baseURL,
  headers: {
    "Content-Type": "application/json",
  },
});

// Request Interceptor: Attach JWT to every request EXCEPT auth endpoints
javaClient.interceptors.request.use(
  (config) => {
    // Skip adding Authorization header for auth endpoints (login, register)
    // and other public endpoints like health check
    const isAuthPath = config.url?.includes("/api/v1/auth/");
    const isHealthPath = config.url?.includes("/api/v1/health");

    if (isAuthPath || isHealthPath) {
      return config;
    }

    // Get token directly from Zustand state
    const token = useAuthStore.getState().token?.trim();
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// Response Interceptor: Handle 401 Unauthorized automatically
javaClient.interceptors.response.use(
  (response) => response,
  (error) => {
    const axiosError = axios.isAxiosError(error) ? (error as AxiosError) : null;
    logApiFailure(axiosError, error);

    if (axiosError?.response?.status === 401) {
      console.warn("Unauthorized! Logging out...");
      useAuthStore.getState().logout();
    }
    return Promise.reject(error);
  }
);

export default javaClient;
