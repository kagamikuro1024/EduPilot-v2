import type { ReactNode } from "react";

export type CalloutVariant = "info";

const variantStyles: Record<CalloutVariant, string> = {
  info: "border-blue-100 bg-blue-50 text-blue-900",
};

export interface CalloutProps {
  title: string;
  children: ReactNode;
  variant?: CalloutVariant;
  /** Optional leading icon */
  icon?: ReactNode;
}

/**
 * Lightweight callout for tips, notices, and inline guidance.
 */
export function Callout({ title, children, variant = "info", icon }: CalloutProps) {
  return (
    <div className={`flex items-start gap-2 rounded-xl px-3 py-3 text-sm ${variantStyles[variant]}`}>
      {icon != null ? <span className="mt-0.5 shrink-0">{icon}</span> : null}
      <div>
        <p className="font-semibold">{title}</p>
        <div className={variant === "info" ? "text-blue-800" : ""}>{children}</div>
      </div>
    </div>
  );
}
