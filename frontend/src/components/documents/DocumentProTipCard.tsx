import { Info } from "lucide-react";
import { Callout } from "@/components/ui/Callout";

export function DocumentProTipCard() {
  return (
    <div className="px-4 pb-4 pt-3">
      <Callout
        variant="info"
        title="Pro Tip for TAs"
        icon={<Info className="h-4 w-4 text-blue-600" />}
      >
        <p>
          Combine similar related PDFs into one larger document for more coherent AI context. Use clear headings
          within your PDFs to improve retrieval.
        </p>
      </Callout>
    </div>
  );
}
