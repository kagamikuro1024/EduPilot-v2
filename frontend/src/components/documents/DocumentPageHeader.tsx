import { FolderSync, Upload } from "lucide-react";

export interface DocumentPageHeaderProps {
  onSyncCourse?: () => void;
  onBatchUpload: () => void;
  isUploading: boolean;
}

export function DocumentPageHeader({ onSyncCourse, onBatchUpload, isUploading }: DocumentPageHeaderProps) {
  return (
    <div className="flex flex-col gap-5 sm:flex-row sm:items-center sm:justify-between mb-8">
      <div className="animate-in fade-in slide-in-from-left duration-500">
        <h1 className="text-4xl font-extrabold tracking-tight text-slate-900 dark:text-white">
          Document <span className="text-gradient">Management</span>
        </h1>
        <p className="mt-2 text-sm font-medium text-slate-500 dark:text-slate-400 max-w-xl leading-relaxed">
          Train your AI Teaching Assistant by uploading course materials, syllabi, and lecture notes to create a rich knowledge base.
        </p>
      </div>
      <div className="flex items-center gap-3 animate-in fade-in slide-in-from-right duration-500">
        <button
          type="button"
          onClick={onSyncCourse}
          disabled={!onSyncCourse}
          className="inline-flex items-center gap-2 rounded-xl border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-950 px-4 py-2.5 text-sm font-bold text-slate-700 dark:text-slate-300 shadow-sm transition-all hover:bg-slate-50 dark:hover:bg-zinc-900 active:scale-95 disabled:cursor-not-allowed disabled:opacity-50"
        >
          <FolderSync className="h-4 w-4" />
          Sync Course
        </button>
        <button
          type="button"
          onClick={onBatchUpload}
          disabled={isUploading}
          className="inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-5 py-2.5 text-sm font-bold text-white shadow-lg shadow-indigo-500/30 transition-all hover:bg-indigo-700 hover:-translate-y-0.5 active:scale-95 disabled:opacity-70 group"
        >
          <Upload className={`h-4 w-4 transition-transform ${isUploading ? 'animate-bounce' : 'group-hover:-translate-y-1'}`} />
          {isUploading ? "Uploading..." : "Batch Upload"}
        </button>
      </div>
    </div>
  );
}

