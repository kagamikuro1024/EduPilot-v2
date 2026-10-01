import { useCallback, useEffect, useMemo, useRef, useState, type ChangeEvent } from "react";
import { documentsService, mapBackendDocumentStatus } from "@/services/documents.service";
import type { DocumentItem, AdminDocumentStatsWire, DocumentKind } from "@/types/documents";

export interface DocumentIndexingToastState {
  title: string;
  remainingLabel: string;
  activeFileName: string;
}

function dedupeById(items: DocumentItem[]): DocumentItem[] {
  const map = new Map<string, DocumentItem>();
  for (const item of items) {
    map.set(item.id, item);
  }
  return Array.from(map.values());
}

function buildIndexingToast(documents: DocumentItem[]): DocumentIndexingToastState {
  const vectorizingDocs = documents.filter((doc) => doc.status === "VECTORIZING");
  if (vectorizingDocs.length === 0) {
    return {
      title: "AI index is healthy",
      remainingLabel: "0 Remaining",
      activeFileName: "All documents are indexed.",
    };
  }

  return {
    title: "AI is indexing documents",
    remainingLabel: `${vectorizingDocs.length} Remaining`,
    activeFileName: `Optimizing: ${vectorizingDocs[0]?.name ?? "Processing file..."}`,
  };
}

export function useDocuments() {
  const [documents, setDocuments] = useState<DocumentItem[]>([]);
  const [stats, setStats] = useState<AdminDocumentStatsWire | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isUploading, setIsUploading] = useState(false);
  const [uploadDocumentType, setUploadDocumentType] = useState<DocumentKind>("COURSE_MATERIAL");
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  const fetchDocuments = useCallback(async () => {
    try {
      const [list, statsData] = await Promise.all([
        documentsService.getDocuments(),
        documentsService.getStats(),
      ]);
      const deduped = dedupeById(list);
      setDocuments((prev) => {
        return deduped.map((newDoc) => {
          const existing = prev.find((d) => d.id === newDoc.id);
          if (existing && newDoc.status !== "READY") {
            return {
              ...newDoc,
              progress: Math.max(existing.progress, newDoc.progress),
            };
          }
          return newDoc;
        });
      });
      setStats(statsData);
      return deduped;
    } catch (error) {
      console.error("Failed to fetch documents or stats", error);
      return [];
    }
  }, []);

  useEffect(() => {
    let mounted = true;

    (async () => {
      try {
        const [list, statsData] = await Promise.all([
          documentsService.getDocuments(),
          documentsService.getStats(),
        ]);
        if (mounted) {
          setDocuments((prev) => {
            const deduped = dedupeById(list);
            return deduped.map((newDoc) => {
              const existing = prev.find((d) => d.id === newDoc.id);
              if (existing && newDoc.status !== "READY") {
                return {
                  ...newDoc,
                  progress: Math.max(existing.progress, newDoc.progress),
                };
              }
              return newDoc;
            });
          });
          setStats(statsData);
        }
      } catch (error) {
        console.error("Failed to fetch documents", error);
      } finally {
        if (mounted) {
          setIsLoading(false);
        }
      }
    })();

    return () => {
      mounted = false;
    };
  }, []);

  useEffect(() => {
    if (!documents.some((doc) => doc.status === "VECTORIZING")) {
      return;
    }

    const intervalId = window.setInterval(() => {
      fetchDocuments().catch((error) => {
        console.error("Polling documents failed", error);
      });
    }, 5000);

    return () => {
      window.clearInterval(intervalId);
    };
  }, [documents, fetchDocuments]);

  const openPicker = useCallback(() => {
    fileInputRef.current?.click();
  }, []);

  const handleFilePicked = useCallback(async (event: ChangeEvent<HTMLInputElement>) => {
    const picked = event.target.files?.[0];
    if (!picked) return;

    setIsUploading(true);
    try {
      const uploadResponse = await documentsService.uploadDocument(picked, uploadDocumentType);
      const initialStatus = mapBackendDocumentStatus(uploadResponse.status);

      const optimisticDoc: DocumentItem = {
        id: uploadResponse.document_id,
        name: picked.name,
        status: initialStatus,
        documentType: uploadDocumentType,
        size: `${(picked.size / (1024 * 1024)).toFixed(1)} MB`,
        uploadedAt: new Date().toLocaleString("en-US", {
          month: "short",
          day: "2-digit",
          year: "numeric",
          hour: "2-digit",
          minute: "2-digit",
        }),
        progress: initialStatus === "READY" ? 100 : 10,
      };

      setDocuments((prev) => {
        const withoutSameId = prev.filter((doc) => doc.id !== optimisticDoc.id);
        return [optimisticDoc, ...withoutSameId];
      });

      await documentsService.pollUntilReady(uploadResponse.document_id, (status, attempt) => {
        setDocuments((prev) =>
          prev.map((doc) =>
            doc.id === uploadResponse.document_id
              ? {
                  ...doc,
                  status,
                  progress:
                    status === "READY" ? 100 : status === "ERROR" ? doc.progress : Math.min(95, 10 + attempt * 8),
                }
              : doc,
          ),
        );
      });

      await fetchDocuments();
    } catch (error) {
      console.error("Document upload failed", error);
    } finally {
      setIsUploading(false);
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
    }
  }, [fetchDocuments, uploadDocumentType]);

  const handleDelete = useCallback(async (documentId: string) => {
    try {
      await documentsService.deleteDocument(documentId);
      setDocuments((prev) => prev.filter((doc) => doc.id !== documentId));
    } catch (error) {
      console.error("Document delete failed", error);
    }
  }, []);

  const indexingToast = useMemo(() => buildIndexingToast(documents), [documents]);

  return {
    documents,
    stats,
    isLoading,
    isUploading,
    uploadDocumentType,
    setUploadDocumentType,
    indexingToast,
    fileInputRef,
    openPicker,
    handleFilePicked,
    handleDelete,
    fetchDocuments,
  };
}
