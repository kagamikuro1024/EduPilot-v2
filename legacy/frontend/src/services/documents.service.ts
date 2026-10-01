import javaClient from "./javaClient";
import type {
  AdminDocumentListItemWire,
  AdminDocumentStatusResponse,
  AdminDocumentStatsWire,
  AdminDocumentUploadResponse,
  DocumentItem,
  DocumentKind,
  DocumentStatus,
} from "@/types/documents";
import { mapWireDocumentType, toDisplaySize, toDisplayUploadedAt } from "@/types/documents";

function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

export function mapBackendDocumentStatus(status: string): DocumentStatus {
  if (status === "READY") return "READY";
  if (status === "PROCESSING") return "VECTORIZING";
  if (status === "DUPLICATE") return "DUPLICATE";
  return "ERROR";
}

export const documentsService = {
  getDocuments: async (): Promise<DocumentItem[]> => {
    const response = await javaClient.get<AdminDocumentListItemWire[]>("/api/v1/admin/documents");
    const map = new Map<string, DocumentItem>();
    for (const item of response.data) {
      map.set(item.document_id, {
        id: item.document_id,
        name: item.name,
        status: mapBackendDocumentStatus(item.status),
        documentType: mapWireDocumentType(item.document_type),
        size: toDisplaySize(item.file_size_bytes),
        uploadedAt: toDisplayUploadedAt(item.created_at),
        progress: mapBackendDocumentStatus(item.status) === "READY" ? 100 : 0,
      });
    }
    return Array.from(map.values());
  },

  uploadDocument: async (file: File, documentType: DocumentKind = "COURSE_MATERIAL"): Promise<AdminDocumentUploadResponse> => {
    const formData = new FormData();
    formData.append("file", file);
    formData.append("document_type", documentType);
    const response = await javaClient.post<AdminDocumentUploadResponse>("/api/v1/admin/documents", formData, {
      headers: { "Content-Type": "multipart/form-data" },
    });
    return response.data;
  },

  getDocumentStatus: async (documentId: string): Promise<AdminDocumentStatusResponse> => {
    const response = await javaClient.get<AdminDocumentStatusResponse>(`/api/v1/admin/documents/${documentId}`);
    return response.data;
  },

  deleteDocument: async (documentId: string): Promise<void> => {
    await javaClient.delete(`/api/v1/admin/documents/${documentId}`);
  },

  getStats: async (): Promise<AdminDocumentStatsWire> => {
    const response = await javaClient.get<AdminDocumentStatsWire>("/api/v1/admin/documents/stats");
    return response.data;
  },

  pollUntilReady: async (
    documentId: string,
    onTick: (status: DocumentStatus, attempt: number) => void,
    initialIntervalMs = 1500,
    maxIntervalMs = 8000,
    maxAttempts = 16,
  ): Promise<DocumentStatus> => {
    for (let attempt = 1; attempt <= maxAttempts; attempt++) {
      try {
        const result = await documentsService.getDocumentStatus(documentId);
        const uiStatus = mapBackendDocumentStatus(result.status);
        onTick(uiStatus, attempt);
        if (uiStatus === "READY" || uiStatus === "ERROR" || uiStatus === "DUPLICATE") {
          return uiStatus;
        }
      } catch {
        // Keep UX stable for temporary network failures during long vectorization.
        onTick("VECTORIZING", attempt);
      }
      const delay = Math.min(maxIntervalMs, initialIntervalMs * Math.pow(2, attempt - 1));
      await sleep(delay);
    }
    return "VECTORIZING";
  },
};
