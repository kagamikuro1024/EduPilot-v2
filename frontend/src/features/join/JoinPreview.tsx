"use client";

import { useState } from "react";
import { SUBJECT, TERM } from "@/mock/core";
import { KEYS, MEMBERS_SEED, type MembersState } from "@/mock/state";
import { useSession } from "@/shared/session/session";
import { useDemoSlice } from "@/shared/state/demo";
import {
  Button,
  ButtonLink,
  DefinitionList,
  EmptyState,
  InlineNotice,
  Page,
  PageHeader,
  PageState,
  Section,
  Skeleton,
  StatusText,
} from "@/shared/ui";
import { BAD_CODE, courseByJoinCode } from "./JoinScreen";
import s from "./Join.module.css";

/** Xem trước lớp trước khi tham gia; lớp bật duyệt thì gửi yêu cầu chờ giảng viên. */
export function JoinPreview({ code }: { code: string }) {
  const { studentId, courses } = useSession();
  const [members, setMembers] = useDemoSlice<MembersState>(KEYS.members, MEMBERS_SEED);
  const [justSent, setJustSent] = useState(false);
  const course = courseByJoinCode(code, members);
  const me = studentId ?? "";

  if (!course) {
    return (
      <Page>
        <PageHeader title="Tham gia lớp" back={{ href: "/join", label: "Nhập mã khác" }} />
        <InlineNotice tone="warning" title={BAD_CODE} action={<ButtonLink href="/join">Nhập mã khác</ButtonLink>}>
          Mã tham gia có thể đã được tạo lại. Hỏi giảng viên phụ trách lớp để lấy mã mới nhất.
        </InlineNotice>
      </Page>
    );
  }

  const already = courses.some((c) => c.id === course.id);
  const pending = (members.pending[course.id] ?? []).includes(me);

  function join() {
    if (course!.requireApproval) {
      setMembers({ ...members, pending: { ...members.pending, [course!.id]: [...(members.pending[course!.id] ?? []), me] } });
    } else {
      setMembers({ ...members, joined: { ...members.joined, [course!.id]: [...(members.joined[course!.id] ?? []), me] } });
    }
    setJustSent(true);
  }

  return (
    <Page>
      <PageHeader title="Tham gia lớp" back={{ href: "/join", label: "Nhập mã khác" }} description="Kiểm tra thông tin lớp trước khi gửi yêu cầu." />
      <PageState
        loading={<Skeleton lines={5} />}
        empty={<EmptyState title="Mã này chưa gắn với lớp nào" action={<ButtonLink href="/join" variant="primary">Nhập mã khác</ButtonLink>} />}
      >
        <Section title={course.name}>
          <DefinitionList
            items={[
              { term: "Học phần", value: `${SUBJECT.name} (${SUBJECT.code})` },
              { term: "Mã lớp", value: course.code },
              { term: "Giảng viên", value: course.teacher },
              { term: "Học kỳ", value: TERM },
              { term: "Lịch học", value: `${course.schedule} · ${course.room}` },
              { term: "Sĩ số", value: `${course.size} chỗ` },
            ]}
          />

          <div className={s.actions}>
            {already && !justSent ? (
              <>
                <StatusText tone="green">Bạn đã là thành viên của lớp này</StatusText>
                <ButtonLink href="/" variant="primary">
                  Về Hôm nay
                </ButtonLink>
              </>
            ) : pending || (justSent && course.requireApproval) ? (
              <>
                <StatusText tone="amber">Đã gửi yêu cầu, chờ giảng viên duyệt</StatusText>
                <p className={s.note}>
                  Lớp này yêu cầu giảng viên duyệt thành viên. Khi được duyệt, lớp sẽ hiện trong bộ chọn lớp và màn Hôm nay của bạn.
                </p>
              </>
            ) : justSent ? (
              <>
                <StatusText tone="green">Bạn đã vào lớp</StatusText>
                <ButtonLink href="/" variant="primary">
                  Về Hôm nay
                </ButtonLink>
              </>
            ) : (
              <>
                <Button variant="primary" onClick={join}>
                  Tham gia lớp
                </Button>
                <p className={s.note}>
                  {course.requireApproval
                    ? "Lớp này cần giảng viên duyệt. Bạn sẽ vào lớp sau khi được duyệt."
                    : "Bạn vào lớp ngay sau khi bấm."}
                </p>
              </>
            )}
          </div>
        </Section>
      </PageState>
    </Page>
  );
}
