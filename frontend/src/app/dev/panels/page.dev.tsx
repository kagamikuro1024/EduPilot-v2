import { PanelSamples } from "./PanelSamples";
import { SurfaceMarker } from "./SurfaceMarker";

// Tệp `page.dev.tsx` chỉ là trang khi `next dev` hoặc build đặt NEXT_PUBLIC_DEV_TOOLS=1; bản "như production" → /dev/* 404 (như /dev/ui).
export const metadata = { title: "Ba phương án độ nổi của panel" };

const SURFACES = ["a", "b", "c"] as const;
const ROLES = ["teacher", "student"] as const;

/** `?surface=a|b|c&role=teacher|student`; thiếu hoặc sai → `a` / `teacher`. */
export default async function Page({ searchParams }: { searchParams: Promise<Record<string, string | string[] | undefined>> }) {
  const q = await searchParams;
  const pick = <T extends string>(v: string | string[] | undefined, allowed: readonly T[], fallback: T): T => {
    const s = Array.isArray(v) ? v[0] : v;
    return allowed.find((a) => a === s) ?? fallback;
  };
  const surface = pick(q.surface, SURFACES, "a");
  const role = pick(q.role, ROLES, "teacher");
  return (
    <>
      <SurfaceMarker surface={surface} />
      <PanelSamples role={role} />
    </>
  );
}
