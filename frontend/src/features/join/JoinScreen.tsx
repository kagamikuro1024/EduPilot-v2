"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { COURSES } from "@/mock/core";
import { KEYS, MEMBERS_SEED, type MembersState } from "@/mock/state";
import { useSession } from "@/shared/session/session";
import { useDemoSlice } from "@/shared/state/demo";
import { Button, EmptyState, Field, InlineNotice, Input, Page, PageHeader, PageState, Section, Skeleton } from "@/shared/ui";
import s from "./Join.module.css";

export const JOIN_FAILS_KEY = "join.fails";
export const MAX_TRIES = 5;
/** Giờ giả lập (ms) lúc gửi yêu cầu vào lớp, theo lớp — Hôm nay của SV ghi "gửi lúc <giờ>" (SRS 4.3.2). */
export const JOIN_SENT_KEY = "join.sentAt";
/** Cùng một câu cho mọi trường hợp — không tiết lộ mã có tồn tại hay không (SRS mục 3). */
export const BAD_CODE = "Mã không hợp lệ hoặc đã hết hạn. Kiểm tra lại với giảng viên.";

/** Tìm lớp theo mã tham gia hiện hành (giảng viên có thể đã tạo lại mã). */
export function courseByJoinCode(code: string, members: MembersState) {
  const want = code.trim().toUpperCase();
  const entry = Object.entries(members.joinCodes).find(([, c]) => c.toUpperCase() === want);
  const id = entry?.[0] ?? COURSES.find((c) => c.joinCode.toUpperCase() === want)?.id;
  return COURSES.find((c) => c.id === id);
}

/** Nhập mã tham gia lớp (INTEGRATION mục 2). */
export function JoinScreen() {
  const router = useRouter();
  const { courses } = useSession();
  const [members] = useDemoSlice<MembersState>(KEYS.members, MEMBERS_SEED);
  const [fails, setFails] = useDemoSlice<number>(JOIN_FAILS_KEY, 0);
  const [code, setCode] = useState("");
  const [error, setError] = useState<string | null>(null);

  const locked = fails >= MAX_TRIES;

  function submit() {
    const course = courseByJoinCode(code, members);
    if (!course) {
      setFails(fails + 1);
      setError(BAD_CODE);
      return;
    }
    setError(null);
    router.push(`/join/${code.trim().toUpperCase()}`);
  }

  return (
    <Page>
      <PageHeader
        title="Tham gia lớp"
        description="Nhập mã tham gia giảng viên gửi cho bạn. Mã gồm 7 ký tự, không phân biệt chữ hoa chữ thường."
        back={courses.length > 0 ? { href: "/", label: "Hôm nay" } : undefined}
      />
      <PageState
        loading={<Skeleton lines={4} />}
        empty={<EmptyState title="Chưa có lớp nào chờ bạn">Khi giảng viên gửi mã tham gia, bạn nhập mã ở đây để vào lớp.</EmptyState>}
      >
        <Section>
          {locked ? (
            <InlineNotice tone="warning" title="Thử lại sau 10 phút">
              Bạn đã nhập sai {MAX_TRIES} lần. Vì lý do an toàn, ô nhập mã tạm khoá 10 phút. Nếu không chắc mã, hỏi lại giảng viên
              phụ trách lớp.
            </InlineNotice>
          ) : (
            <form
              className={s.form}
              onSubmit={(e) => {
                e.preventDefault();
                submit();
              }}
            >
              <Field
                label="Mã tham gia"
                error={error ?? undefined}
                helper={error ? undefined : `Còn ${MAX_TRIES - fails} lần thử trước khi ô nhập tạm khoá.`}
              >
                {(id, describedBy) => (
                  <Input
                    id={id}
                    aria-describedby={describedBy}
                    className={s.codeInput}
                    value={code}
                    invalid={Boolean(error)}
                    autoComplete="off"
                    spellCheck={false}
                    maxLength={12}
                    placeholder="AN7K2MQ"
                    onChange={(e) => {
                      setCode(e.target.value);
                      setError(null);
                    }}
                  />
                )}
              </Field>
              <Button type="submit" variant="primary" disabled={code.trim().length < 4}>
                Xem lớp
              </Button>
            </form>
          )}
        </Section>

        {courses.length > 0 && (
          <Section title="Lớp bạn đang học">
            <ul className={s.mine}>
              {courses.map((c) => (
                <li key={c.id}>
                  {c.label} · {c.schedule}
                </li>
              ))}
            </ul>
          </Section>
        )}
      </PageState>
    </Page>
  );
}
