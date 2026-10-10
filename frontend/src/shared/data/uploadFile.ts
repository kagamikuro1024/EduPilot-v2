import { apiClient } from "./apiClient";
import { ApiError } from "./ApiError";

export const DOC_MAX_BYTES = 50 * 1024 * 1024;
const EXT = /\.(pdf|docx|pptx)$/i;
const MIME: Record<string, string> = { pdf: "application/pdf", docx: "application/vnd.openxmlformats-officedocument.wordprocessingml.document", pptx: "application/vnd.openxmlformats-officedocument.presentationml.presentation" };

/** Kiểm tệp TRƯỚC khi tải: trả câu báo kèm cách sửa, hoặc null nếu hợp lệ (SRS FEAT-docs-calendar 4.1). */
export function checkDocumentFile(f: File): string | null {
  if (!EXT.test(f.name)) return `${f.name}: chỉ nhận PDF, DOCX hoặc PPTX. Hãy chọn tệp khác.`;
  if (f.size > DOC_MAX_BYTES) return `${f.name}: quá 50 MB. Hãy nén hoặc tách tệp.`;
  if (f.size === 0) return `${f.name}: tệp rỗng.`;
  return null;
}

async function sha256Hex(f: File): Promise<string> {
  const d = await crypto.subtle.digest("SHA-256", await f.arrayBuffer());
  return [...new Uint8Array(d)].map((b) => b.toString(16).padStart(2, "0")).join("");
}

/**
 * Tải một tệp tài liệu lên bằng URL ký sẵn (không qua gateway): `presign` → PUT thẳng tới kho tệp (token KHÔNG gửi tới origin khác) → trả `upload_id` cho `uploads/complete`.
 * Lỗi mạng ném `ApiError{code:"NETWORK"}` để giao diện cho `Tải lại` đúng tệp đó.
 */
export async function uploadDocumentFile(courseId: string, file: File, signal?: AbortSignal): Promise<string> {
  const mime = MIME[file.name.split(".").pop()!.toLowerCase()] ?? file.type;
  const sha256 = await sha256Hex(file);
  const p = await apiClient.post<{ upload_id: string; method: string; url: string; headers: Record<string, string> }>(`/courses/${courseId}/uploads/presign`, { purpose: "document", filename: file.name, mime_type: mime, size_bytes: file.size, sha256 }, { signal });
  let res: Response;
  try {
    res = await fetch(p.data.url, { method: p.data.method, headers: p.data.headers, body: file, signal });
  } catch {
    throw new ApiError({ status: 0, code: "NETWORK" });
  }
  if (!res.ok) throw new ApiError({ status: res.status >= 500 ? 502 : 400, code: res.status >= 500 ? "BAD_GATEWAY" : "UPLOAD_INCOMPLETE" });
  return p.data.upload_id;
}
