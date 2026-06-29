"use client";

import { WorkspaceLayout } from "@/components/layout/WorkspaceLayout";
import { DocumentIndexingToast } from "@/components/documents/DocumentIndexingToast";
import { DocumentKnowledgeBaseSection } from "@/components/documents/DocumentKnowledgeBaseSection";
import { DocumentPageHeader } from "@/components/documents/DocumentPageHeader";
import { DocumentStatsSection } from "@/components/documents/DocumentStatsSection";
import { DocumentUploadDropzone } from "@/components/documents/DocumentUploadDropzone";
import { DocumentsPageFooter } from "@/components/documents/DocumentsPageFooter";
import { HiddenPdfFileInput } from "@/components/documents/HiddenPdfFileInput";
import type { StatCardItem } from "@/components/documents/types";
import { useDocuments } from "@/hooks/useDocuments";

const SYSTEM_VERSION = "Version 1.2.4-stable";

export default function DocumentsPage() {
  const {
    documents,
    stats,
    indexingToast,
    isLoading,
    isUploading,
    uploadDocumentType,
    setUploadDocumentType,
    fileInputRef,
    openPicker,
    handleFilePicked,
    handleDelete,
  } = useDocuments();

  const statCards: StatCardItem[] = [
    {
      id: "total-documents",
      label: "Total Documents",
      value: stats?.total_documents.toString() ?? "0",
      helperText: "Live registry count",
      tone: "blue",
    },
    {
      id: "index-health",
      label: "AI Index Health",
      value: stats?.index_health ?? "0%",
      helperText: "Percentage of READY docs",
      tone: "green",
    },
    {
      id: "knowledge-volume",
      label: "Knowledge Volume",
      value: stats?.knowledge_volume ?? "0 B",
      helperText: "Total file size",
      tone: "purple",
    },
  ];

  return (
    <WorkspaceLayout
      footerLine2="rights"
      avatarInitialsOverride="AR"
      footerFloating={<DocumentIndexingToast toast={indexingToast} />}
    >
      <HiddenPdfFileInput inputRef={fileInputRef} onChange={handleFilePicked} />
      <div className="mx-auto max-w-6xl space-y-5 px-5 py-5">
        <DocumentPageHeader onBatchUpload={openPicker} isUploading={isUploading} />
        <DocumentStatsSection cards={statCards} />
        <DocumentUploadDropzone
          onSelectFiles={openPicker}
          isUploading={isUploading}
          documentType={uploadDocumentType}
          onDocumentTypeChange={setUploadDocumentType}
        />
        {isLoading ? (
          <section className="rounded-xl border border-slate-200 bg-white p-6 text-sm text-slate-500 shadow-sm">
            Loading documents...
          </section>
        ) : (
          <DocumentKnowledgeBaseSection documents={documents} onDelete={handleDelete} />
        )}
        <DocumentsPageFooter versionLabel={SYSTEM_VERSION} />
      </div>
    </WorkspaceLayout>
  );
}
