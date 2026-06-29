export function extractErrorMessage(
  err: unknown,
  defaultMessage: string = "An error occurred. Please try again."
): string {
  if (err && typeof err === "object" && "response" in err) {
    const errorObj = err as {
      response?: {
        data?: { message?: string; error?: string; detail?: string } | string;
      };
    };
    const data = errorObj.response?.data;
    if (typeof data === "string" && data.trim()) {
      return data;
    }
    if (data && typeof data === "object") {
      if (typeof data.message === "string" && data.message.trim()) return data.message;
      if (typeof data.error === "string" && data.error.trim()) return data.error;
      if (typeof data.detail === "string" && data.detail.trim()) return data.detail;
    }
  }
  if (err instanceof Error) {
    return err.message;
  }
  return defaultMessage;
}
