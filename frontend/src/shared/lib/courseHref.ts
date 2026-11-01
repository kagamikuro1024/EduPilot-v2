/**
 * Liên kết tới màn bài tập mang theo `?course=<id>`: màn mở ra gọi `…/mine` ngay khi có phiên, song song với `/me/courses`
 * (bớt một vòng chờ — TL-1). Liên kết đã có tham số hoặc không phải màn bài tập thì giữ nguyên.
 */
export const withCourse = (href: string, courseId: string | null | undefined): string =>
  courseId && href.startsWith("/assignments/") && !href.includes("?") ? `${href}?course=${courseId}` : href;
