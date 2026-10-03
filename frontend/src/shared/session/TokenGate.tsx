"use client";

import { useState } from "react";
import { tokenStore } from "@/shared/data/tokenStore";
import { Button, Field, Input, InlineNotice, Page, PageHeader, Section } from "@/shared/ui";
import { checkToken } from "./jwt";

/**
 * Cổng dán token quản trị — CHỈ có ở bản dựng `NEXT_PUBLIC_DEV_AUTH=1` (US-PU-04 AC9). Token chỉ vào `tokenStore` (bộ nhớ):
 * không lưu, không hiện lại, ô nhập là password. Bỏ khi P2 có đăng nhập thật.
 */
export default function TokenGate({ expired }: { expired: boolean }) {
  const [value, setValue] = useState("");
  const [problem, setProblem] = useState<"invalid" | "expired" | null>(null);

  function submit(e: React.FormEvent) {
    e.preventDefault();
    const r = checkToken(value);
    if (!r.ok) {
      setProblem(r.reason);
      return;
    }
    setProblem(null);
    tokenStore.set(value);
    setValue("");
  }

  return (
    <Page>
      <PageHeader title="Dán token quản trị để tiếp tục" description="Màn này làm việc với máy chủ thật. Đăng nhập chính thức sẽ có ở bản sau; trong lúc chờ, dán token do quản trị viên cấp." />
      <Section>
        <form data-part="token-gate" onSubmit={submit} style={{ display: "grid", gap: "var(--ep-space-4)", maxWidth: 480, justifyItems: "start" }}>
          {(problem === "expired" || (expired && !problem)) && <InlineNotice tone="warning" title="Phiên đã hết hạn">Dán token mới để tiếp tục.</InlineNotice>}
          <Field label="Token" error={problem === "invalid" ? "Token không hợp lệ." : undefined}>
            {(id, describedBy) => (
              <Input id={id} aria-describedby={describedBy} invalid={problem === "invalid"} type="password" autoComplete="off" spellCheck={false} value={value} onChange={(e) => setValue(e.target.value)} />
            )}
          </Field>
          <Button type="submit" variant="primary">
            Dùng token
          </Button>
        </form>
      </Section>
    </Page>
  );
}
