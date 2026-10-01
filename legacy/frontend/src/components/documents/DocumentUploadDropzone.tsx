import { Upload } from "lucide-react";
import type { DocumentKind } from "@/types/documents";

export interface DocumentUploadDropzoneProps {
  onSelectFiles: () => void;
  isUploading: boolean;
  onImportFromDrive?: () => void;
  documentType: DocumentKind;
  onDocumentTypeChange: (value: DocumentKind) => void;
}

export function DocumentUploadDropzone({
  onSelectFiles,
  isUploading,
  onImportFromDrive,
  documentType,
  onDocumentTypeChange,
}: DocumentUploadDropzoneProps) {
  return (
    <section className="rounded-[2.5rem] border-2 border-dashed border-indigo-200 dark:border-indigo-800/50 bg-white/40 dark:bg-zinc-950/20 px-6 py-14 transition-all duration-300 hover:border-indigo-400 dark:hover:border-indigo-700 hover:bg-indigo-50/30 dark:hover:bg-indigo-900/10 group glass-panel premium-shadow">
      <div className="mx-auto max-w-xl text-center">
        <div className="mx-auto flex h-20 w-20 items-center justify-center rounded-3xl bg-indigo-50 dark:bg-indigo-900/30 text-indigo-600 dark:text-indigo-400 group-hover:scale-110 group-hover:rotate-3 transition-transform duration-500 shadow-inner">
          <Upload className="h-8 w-8" />
        </div>
        <h2 className="mt-6 text-2xl font-extrabold text-slate-900 dark:text-white tracking-tight">Upload knowledge documents</h2>
        <p className="mt-2 text-sm font-medium text-slate-500 dark:text-slate-400 max-w-sm mx-auto leading-relaxed">
          Drag and drop PDF files here, or click to browse. Choose document type so the AI uses the correct, privacy-safe data flow.
        </p>
        <div className="mt-6 flex flex-col items-center gap-2">
          <label htmlFor="doc-upload-type" className="text-[10px] font-bold uppercase tracking-widest text-slate-400">
            Document type
          </label>
          <select
            id="doc-upload-type"
            value={documentType}
            onChange={(e) => onDocumentTypeChange(e.target.value as DocumentKind)}
            disabled={isUploading}
            className="rounded-xl border border-slate-200 dark:border-zinc-700 bg-white dark:bg-zinc-900 px-4 py-2 text-sm font-semibold text-slate-800 dark:text-slate-200 shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:opacity-60"
          >
            <option value="COURSE_MATERIAL">Course material (slides, syllabus)</option>
            <option value="REGULATION">Regulation / quy chế (school policy)</option>
            <option value="GRADE_REPORT">Grade report / bảng điểm (private lookup)</option>
          </select>
        </div>
        <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
          <button
            type="button"
            onClick={onSelectFiles}
            disabled={isUploading}
            className="rounded-xl bg-indigo-600 px-6 py-3 text-sm font-bold text-white shadow-xl shadow-indigo-500/20 transition-all hover:bg-indigo-700 hover:-translate-y-0.5 active:scale-95 disabled:opacity-70"
          >
            {isUploading ? "Uploading..." : "Select PDF Files"}
          </button>
          <button
            type="button"
            onClick={onImportFromDrive}
            disabled={!onImportFromDrive}
            className="rounded-xl border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 px-6 py-3 text-sm font-bold text-slate-700 dark:text-slate-300 shadow-sm transition-all hover:bg-slate-50 dark:hover:bg-zinc-800 active:scale-95 disabled:cursor-not-allowed disabled:opacity-50"
          >
            Import from Drive
          </button>
        </div>
        <div className="mt-8 flex items-center justify-center gap-6">
          <div className="flex flex-col items-center">
            <span className="text-[10px] font-bold uppercase tracking-widest text-slate-400">Max Size</span>
            <span className="text-xs font-bold text-slate-600 dark:text-slate-300 mt-1">50 MB</span>
          </div>
          <div className="h-8 w-px bg-slate-200 dark:bg-zinc-800" />
          <div className="flex flex-col items-center">
            <span className="text-[10px] font-bold uppercase tracking-widest text-slate-400">Format</span>
            <span className="text-xs font-bold text-slate-600 dark:text-slate-300 mt-1">PDF Only</span>
          </div>
          <div className="h-8 w-px bg-slate-200 dark:bg-zinc-800" />
          <div className="flex flex-col items-center">
            <span className="text-[10px] font-bold uppercase tracking-widest text-slate-400">Security</span>
            <span className="text-xs font-bold text-slate-600 dark:text-slate-300 mt-1">Encrypted</span>
          </div>
        </div>
      </div>
    </section>
  );
}

