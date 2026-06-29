import { CheckCircle2, CircleAlert, Loader2 } from "lucide-react";
import type { DocumentItem } from "@/types/documents";

export interface DocumentStatusCellProps {
  document: DocumentItem;
}

/**
 * Renders pipeline status for a single document row (UI contract for documents table).
 */
export function DocumentStatusCell({ document }: DocumentStatusCellProps) {
  if (document.status === "READY") {
    return (
      <span className="inline-flex items-center gap-1 rounded-full bg-emerald-50 px-2 py-0.5 text-xs font-semibold text-emerald-700 ring-1 ring-emerald-100">
        <CheckCircle2 className="h-3.5 w-3.5" />
        Ready
      </span>
    );
  }

  if (document.status === "ERROR") {
    return (
      <span className="inline-flex items-center gap-1 rounded-full bg-red-100 px-2 py-0.5 text-xs font-semibold text-red-700 ring-1 ring-red-200">
        <CircleAlert className="h-3.5 w-3.5" />
        Error
      </span>
    );
  }

  if (document.status === "DUPLICATE") {
    return (
      <span className="inline-flex items-center gap-1 rounded-full bg-amber-100 px-2 py-0.5 text-xs font-semibold text-amber-700 ring-1 ring-amber-200">
        <CircleAlert className="h-3.5 w-3.5" />
        Duplicate file
      </span>
    );
  }

  return (
    <div className="w-full max-w-[13rem] space-y-1">
      <div className="flex items-center justify-between gap-2 text-xs">
        <span className="inline-flex items-center gap-1 font-semibold text-blue-700">
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          Vectorizing...
        </span>
        <span className="font-semibold text-slate-500">{document.progress}%</span>
      </div>
      <div className="h-1.5 w-full rounded-full bg-slate-200">
        <div
          className="h-full rounded-full bg-blue-600 transition-all"
          style={{ width: `${document.progress}%` }}
        />
      </div>
    </div>
  );
}
