import { Loader2 } from "lucide-react";
import type { DocumentIndexingToastState } from "@/hooks/useDocuments";

export interface DocumentIndexingToastProps {
  toast: DocumentIndexingToastState;
}

export function DocumentIndexingToast({ toast }: DocumentIndexingToastProps) {
  return (
    <div className="pointer-events-none fixed bottom-12 right-6 z-20 w-[18.5rem] rounded-xl border border-slate-200 bg-white p-3 shadow-lg">
      <div className="flex items-center justify-between gap-2">
        <p className="text-sm font-semibold text-slate-900">{toast.title}</p>
        <span className="rounded bg-slate-100 px-1.5 py-0.5 text-[10px] font-semibold text-slate-600">
          {toast.remainingLabel}
        </span>
      </div>
      <div className="mt-2 flex items-center gap-2 text-xs text-slate-600">
        <Loader2 className="h-3.5 w-3.5 shrink-0 animate-spin text-blue-600" />
        <span>{toast.activeFileName}</span>
      </div>
    </div>
  );
}
