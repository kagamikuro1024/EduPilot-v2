import type { DocumentItem } from "@/types/documents";
import { DocumentKnowledgeBaseTable } from "./DocumentKnowledgeBaseTable";
import { DocumentKnowledgeToolbar } from "./DocumentKnowledgeToolbar";
import { DocumentProTipCard } from "./DocumentProTipCard";

export interface DocumentKnowledgeBaseSectionProps {
  documents: DocumentItem[];
  onDelete: (id: string) => void;
}

export function DocumentKnowledgeBaseSection({ documents, onDelete }: DocumentKnowledgeBaseSectionProps) {
  return (
    <section className="rounded-xl glass-panel overflow-hidden">
      <DocumentKnowledgeToolbar />
      <DocumentKnowledgeBaseTable documents={documents} onDelete={onDelete} />
      <DocumentProTipCard />
    </section>
  );
}
