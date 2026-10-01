export const SENSITIVE_CONTENT_REGEX =
  /\b(\d{8,12}|student\s*id|mssv|grade|gpa|password|phone|email|cccd|social\s*security)\b/i;

export function hasSensitiveSignals(content: string): boolean {
  return SENSITIVE_CONTENT_REGEX.test(content);
}
