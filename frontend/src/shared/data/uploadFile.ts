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
export const uploadDocumentFile = (courseId: string, file: File, signal?: AbortSignal) => uploadFile(courseId, file, "document", signal);

/** Tệp XLSX nhập điểm (≤ 2 MiB): cùng đường tải, `purpose = grade_import`; chỉ TEACHER. */
export const GRADE_IMPORT_MAX_BYTES = 2 * 1024 * 1024;
const XLSX_MIME = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet";
export function checkGradeImportFile(f: File): string | null {
  if (!/\.xlsx$/i.test(f.name)) return `${f.name}: chỉ nhận XLSX. Hãy chọn tệp khác.`;
  if (f.size > GRADE_IMPORT_MAX_BYTES) return `${f.name}: quá 2 MB. Hãy tách tệp.`;
  if (f.size === 0) return `${f.name}: tệp rỗng.`;
  return null;
}
export const uploadGradeImportFile = (courseId: string, file: File, signal?: AbortSignal) => uploadFile(courseId, file, "grade_import", signal);

// ---- nộp bài tập (US-P7-03 AC2, AC3): .pdf .docx .txt .md, ≤ 20 MiB ------------------------------------------------------
export const ASSIGN_MAX_BYTES = 20 * 1024 * 1024;
const ASSIGN_EXT = /\.(pdf|docx|txt|md)$/i;

/** Kiểm TRƯỚC mọi yêu cầu mạng; trả câu lỗi hoặc null. Máy chủ kiểm lại độc lập. */
export function checkSubmissionFile(f: File): string | null {
  if (!ASSIGN_EXT.test(f.name)) return "Chỉ nhận PDF, DOCX, TXT, MD";
  if (f.size > ASSIGN_MAX_BYTES) return "Tệp lớn hơn 20 MB";
  if (f.size === 0) return "Tệp rỗng";
  return null;
}

/**
 * Xin URL ký sẵn rồi `PUT` thẳng tệp lên kho (byte không qua gateway, token không gửi sang origin khác):
 * gửi ĐÚNG các header máy chủ trả về, kể cả `x-amz-checksum-sha256`; `Content-Length` do trình duyệt tự đặt.
 * Trả `upload_id` để `POST …/submit` hoàn tất.
 */
export async function uploadSubmissionFile(courseId: string, assignmentId: string, file: File, signal?: AbortSignal): Promise<string> {
  const sha256 = await sha256Hex(file);
  const p = await apiClient.post<{ upload_id: string; method: string; url: string; headers: Record<string, string> }>(
    `/courses/${courseId}/assignments/${assignmentId}/submit/presign`,
    { filename: file.name, size_bytes: file.size, sha256 },
    { signal },
  );
  const headers = { ...p.data.headers };
  delete headers["Content-Length"];
  let res: Response;
  try {
    res = await fetch(p.data.url, { method: p.data.method, headers, body: file, signal });
  } catch {
    throw new ApiError({ status: 0, code: "NETWORK" });
  }
  if (!res.ok) throw new ApiError({ status: res.status >= 500 ? 502 : 400, code: res.status >= 500 ? "BAD_GATEWAY" : "UPLOAD_INCOMPLETE" });
  return p.data.upload_id;
}

/** Tệp lẻ hoặc ZIP bài nộp do nhân viên tải lên (US-P7-04 / US-P7-08): `purpose = submission_upload`. */
const BUNDLE_EXT = /\.(zip|pdf|docx|txt|md)$/i;
export function checkSubmissionBundle(f: File): string | null {
  if (!BUNDLE_EXT.test(f.name)) return "Chỉ nhận ZIP, PDF, DOCX, TXT, MD";
  if (f.size === 0) return "Tệp rỗng";
  return null;
}
export const uploadSubmissionBundle = (courseId: string, file: File, signal?: AbortSignal) => uploadFile(courseId, file, "submission_upload", signal);

async function uploadFile(courseId: string, file: File, purpose: "document" | "grade_import" | "submission_upload", signal?: AbortSignal): Promise<string> {
  const mime = purpose === "grade_import" ? XLSX_MIME : (MIME[file.name.split(".").pop()!.toLowerCase()] ?? file.type);
  const sha256 = await sha256Hex(file);
  const p = await apiClient.post<{ upload_id: string; method: string; url: string; headers: Record<string, string> }>(`/courses/${courseId}/uploads/presign`, { purpose, filename: file.name, mime_type: mime, size_bytes: file.size, sha256 }, { signal });
  let res: Response;
  try {
    res = await fetch(p.data.url, { method: p.data.method, headers: p.data.headers, body: file, signal });
  } catch {
    throw new ApiError({ status: 0, code: "NETWORK" });
  }
  if (!res.ok) throw new ApiError({ status: res.status >= 500 ? 502 : 400, code: res.status >= 500 ? "BAD_GATEWAY" : "UPLOAD_INCOMPLETE" });
  return p.data.upload_id;
}
