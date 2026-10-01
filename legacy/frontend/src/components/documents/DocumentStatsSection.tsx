import { CheckCircle2, FileText, Info } from "lucide-react";
import { StatMetricCard } from "@/components/ui/StatMetricCard";
import type { StatCardItem } from "./types";
import { statToneIconClass } from "./documentStatTone";

export interface DocumentStatsSectionProps {
  cards: StatCardItem[];
}

function statIconForId(id: string) {
  if (id === "total-documents") return <FileText className="h-4 w-4" />;
  if (id === "index-health") return <CheckCircle2 className="h-4 w-4" />;
  return <Info className="h-4 w-4" />;
}

export function DocumentStatsSection({ cards }: DocumentStatsSectionProps) {
  return (
    <section className="grid grid-cols-1 gap-3 md:grid-cols-3">
      {cards.map((card) => (
        <StatMetricCard
          key={card.id}
          label={card.label}
          value={card.value}
          helperText={card.helperText}
          icon={statIconForId(card.id)}
          iconAccentClassName={statToneIconClass(card.tone)}
        />
      ))}
    </section>
  );
}
