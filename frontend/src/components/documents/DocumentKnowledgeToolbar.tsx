import { Filter, Search } from "lucide-react";

export interface DocumentKnowledgeToolbarProps {
  /** Wire when search API exists */
  searchPlaceholder?: string;
}

export function DocumentKnowledgeToolbar({ searchPlaceholder = "Search documents..." }: DocumentKnowledgeToolbarProps) {
  return (
    <div className="flex flex-col gap-3 border-b border-slate-100 px-4 py-3 sm:flex-row sm:items-center sm:justify-between">
      <h2 className="text-lg font-semibold text-slate-900">Course Knowledge Base</h2>
      <div className="flex items-center gap-2">
        <label className="relative block w-full sm:w-64">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
          <input
            type="search"
            placeholder={searchPlaceholder}
            readOnly
            className="w-full rounded-lg border border-slate-200 bg-white py-2 pl-9 pr-3 text-sm text-slate-700 placeholder:text-slate-400 outline-none ring-blue-100 focus:border-blue-300 focus:ring-4"
          />
          {/* TODO: API INTEGRATION - GET /api/v1/documents?search=... */}
        </label>
        <button
          type="button"
          className="rounded-lg border border-slate-200 bg-white p-2 text-slate-600 transition hover:bg-slate-50"
          aria-label="Filter documents"
        >
          <Filter className="h-4 w-4" />
          {/* TODO: API INTEGRATION - GET /api/v1/documents?status=... */}
        </button>
      </div>
    </div>
  );
}
