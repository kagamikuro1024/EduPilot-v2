export type DocumentStatus = "READY" | "VECTORIZING" | "ERROR" | "DUPLICATE";

/** Matches backend `DocumentType` / DB `documents.document_type` */
export type DocumentKind = "COURSE_MATERIAL" | "REGULATION";

export interface DocumentItem {
  id: string;
  name: string;
  status: DocumentStatus;
  /** RAG bucket: course slides vs school regulations */
  documentType: DocumentKind;
  size: string;
  uploadedAt: string;
  progress: number;
}

export interface AdminDocumentUploadResponse {
  document_id: string;
  status: string;
}

export interface AdminDocumentStatusResponse {
  status: string;
}

export interface AdminDocumentStatsWire {
  total_documents: number;
  index_health: string;
  knowledge_volume: string;
}

export interface AdminDocumentListItemWire {
  document_id: string;
  name: string;
  status: string;
  document_type?: string | null;
  file_size_bytes: number | null;
  created_at: string | null;
}

export function mapWireDocumentType(raw: string | null | undefined): DocumentKind {
  if (raw === "REGULATION") return "REGULATION";
  return "COURSE_MATERIAL";
}

export function toDisplaySize(bytes: number | null): string {
  if (bytes == null) return "-";
  if (bytes < 1024) return `${bytes} B`;
  const kb = bytes / 1024;
  if (kb < 1024) return `${kb.toFixed(0)} KB`;
  const mb = kb / 1024;
  return `${mb.toFixed(1)} MB`;
}

export function toDisplayUploadedAt(isoDate: string | null): string {
  if (!isoDate) return "-";
  const date = new Date(isoDate);
  if (Number.isNaN(date.getTime())) return "-";

  return date.toLocaleString("en-US", {
    month: "short",
    day: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}
