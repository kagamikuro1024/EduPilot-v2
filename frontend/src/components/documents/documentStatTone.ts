import type { StatCardItem } from "./types";

export function statToneIconClass(tone: StatCardItem["tone"]): string {
  switch (tone) {
    case "blue":
      return "bg-blue-100 text-blue-600";
    case "green":
      return "bg-emerald-100 text-emerald-600";
    case "purple":
      return "bg-violet-100 text-violet-600";
  }
}
