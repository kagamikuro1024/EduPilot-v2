export interface DocumentsPageFooterProps {
  versionLabel: string;
}

export function DocumentsPageFooter({ versionLabel }: DocumentsPageFooterProps) {
  return (
    <footer className="flex flex-col gap-2 border-t border-slate-200 px-1 pb-1 pt-3 text-xs text-slate-500 sm:flex-row sm:items-center sm:justify-between">
      <span>© 2024 EduPilot AI. All rights reserved.</span>
      <div className="flex items-center gap-2">
        <span className="inline-flex items-center gap-1">
          <span className="h-2 w-2 rounded-full bg-emerald-500" />
          System Online
        </span>
        <span>•</span>
        <span>{versionLabel}</span>
      </div>
    </footer>
  );
}
