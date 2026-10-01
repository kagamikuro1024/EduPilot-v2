import Image from "next/image";

export default function Home() {
  return (
    <main style={{ padding: "var(--ep-space-12) var(--ep-space-6)" }}>
      <Image src="/brand/logo-edupilot.svg" alt="EduPilot" width={160} height={40} priority />
    </main>
  );
}
