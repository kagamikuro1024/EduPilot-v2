/** View model for dashboard stat tiles (not the same shape as backend DTOs yet). */
export interface StatCardItem {
  id: string;
  label: string;
  value: string;
  helperText: string;
  tone: "blue" | "green" | "purple";
}
