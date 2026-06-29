import { FileText, Trash2 } from "lucide-react";
import type { DocumentItem } from "@/types/documents";
import { DocumentStatusCell } from "./DocumentStatusCell";

function documentTypeLabel(t: DocumentItem["documentType"]): string {
  return t === "REGULATION" ? "Quy chế" : "Môn học";
}

export interface DocumentKnowledgeBaseTableProps {
  documents: DocumentItem[];
  onDelete: (id: string) => void;
}

export function DocumentKnowledgeBaseTable({ documents, onDelete }: DocumentKnowledgeBaseTableProps) {
  return (
    <div className="overflow-x-auto overflow-y-auto max-h-[600px] rounded-2xl border border-slate-200 dark:border-zinc-800 bg-white/50 dark:bg-zinc-950/50 backdrop-blur-md relative">
      <table className="min-w-full text-left text-sm border-separate border-spacing-0">
        <thead className="sticky top-0 z-10 bg-slate-100/90 dark:bg-zinc-900/90 backdrop-blur-md text-[10px] font-bold uppercase tracking-widest text-slate-500 dark:text-slate-400">
          <tr>
            <th className="px-6 py-4 border-b border-slate-200 dark:border-zinc-800">Document Name</th>
            <th className="px-6 py-4 border-b border-slate-200 dark:border-zinc-800">Type</th>
            <th className="px-6 py-4 border-b border-slate-200 dark:border-zinc-800">Status</th>
            <th className="px-6 py-4 border-b border-slate-200 dark:border-zinc-800">Size</th>
            <th className="px-6 py-4 border-b border-slate-200 dark:border-zinc-800">Uploaded At</th>
            <th className="px-6 py-4 border-b border-slate-200 dark:border-zinc-800 text-right">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100 dark:divide-zinc-800">
          {documents.map((document) => (
            <tr key={document.id} className="group hover:bg-indigo-50/50 dark:hover:bg-indigo-950/30 transition-all duration-300">
              <td className="px-6 py-4">
                <div className="flex items-center gap-3">
                  <div className="rounded-xl bg-slate-100 dark:bg-zinc-800 p-2.5 text-slate-500 dark:text-slate-400 group-hover:bg-indigo-100 group-hover:text-indigo-600 dark:group-hover:bg-indigo-900/50 dark:group-hover:text-indigo-400 transition-colors">
                    <FileText className="h-4.5 w-4.5" />
                  </div>
                  <span className="font-bold text-slate-900 dark:text-white tracking-tight">{document.name}</span>
                </div>
              </td>
              <td className="px-6 py-4">
                <span className="inline-flex rounded-lg bg-slate-100 dark:bg-zinc-800 px-2.5 py-1 text-[10px] font-bold uppercase tracking-wide text-slate-600 dark:text-slate-300">
                  {documentTypeLabel(document.documentType)}
                </span>
              </td>
              <td className="px-6 py-4">
                <DocumentStatusCell document={document} />
              </td>
              <td className="px-6 py-4 text-xs font-bold text-slate-500 dark:text-slate-400 uppercase tracking-tight">{document.size}</td>
              <td className="px-6 py-4 text-xs font-bold text-slate-500 dark:text-slate-400 uppercase tracking-tight">{document.uploadedAt}</td>
              <td className="px-6 py-4 text-right">
                <button
                  type="button"
                  className="rounded-lg p-2 text-slate-400 transition-all hover:bg-rose-50 hover:text-rose-600 dark:hover:bg-rose-900/20 active:scale-90"
                  aria-label={`Delete ${document.name}`}
                  onClick={() => onDelete(document.id)}
                >
                  <Trash2 className="h-4.5 w-4.5" />
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

