"use client";

import { Copy, Link2 } from "lucide-react";
import { useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";
import { PENDING_STUDENT_IDS, studentById, type Student } from "@/mock/core";
import { rosterOf } from "@/mock/roster";
import { noteJoinDecided } from "@/mock/notes";
import { KEYS, MEMBERS_SEED, type MembersState } from "@/mock/state";
import { useUndoLine } from "@/shared/lib/useUndoLine";
import { useSession } from "@/shared/session/session";
import { useDemoSlice } from "@/shared/state/demo";
import {
  ActionList,
  ActionRow,
  Button,
  ConfirmIrreversible,
  DataTable,
  EmptyState,
  Field,
  Input,
  OverflowMenu,
  Page,
  PageHeader,
  PageState,
  Section,
  Skeleton,
  useRouteState,
} from "@/shared/ui";
import s from "./MembersView.module.css";

const CODE_CHARS = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789";

/** Thành viên lớp và yêu cầu vào lớp (INTEGRATION mục 2). TA xem và duyệt; chỉ giảng viên đổi mã. */
export function MembersView() {
  const { role, course } = useSession();
  const [members, setMembers] = useDemoSlice<MembersState>(KEYS.members, MEMBERS_SEED);
  const tab = useSearchParams().get("tab");
  const [inviting, setInviting] = useState(tab === "staff");
  const [email, setEmail] = useState("");
  const [regenerating, setRegenerating] = useState(false);
  const [removing, setRemoving] = useState<Student | null>(null);
  const [removed, setRemoved] = useState<string[]>([]);
  const undo = useUndoLine();
  const routeState = useRouteState();

  // Liên kết sâu `?tab=pending` từ thẻ "yêu cầu vào lớp chờ duyệt" và từ chuông (SRS 4.9).
  useEffect(() => {
    if (tab === "pending") document.getElementById("pending")?.scrollIntoView({ block: "start" });
  }, [tab]);

  const isTeacher = role === "teacher";
  const code = members.joinCodes[course.id] ?? course.joinCode;
  const pending = (members.pending[course.id] ?? []).map(studentById).filter((x): x is Student => Boolean(x));
  const roster = rosterOf(course.id, members).filter((st) => !removed.includes(st.id));

  function decide(st: Student, approve: boolean) {
    const before = members;
    setMembers((prev) => ({
      ...prev,
      pending: { ...prev.pending, [course.id]: (prev.pending[course.id] ?? []).filter((id) => id !== st.id) },
      joined: approve ? { ...prev.joined, [course.id]: [...(prev.joined[course.id] ?? []), st.id] } : prev.joined,
      rejected: approve ? prev.rejected : { ...prev.rejected, [course.id]: [...(prev.rejected[course.id] ?? []), st.id] },
    }));
    noteJoinDecided(st.id, course, approve);
    undo.push(`${approve ? "Đã duyệt" : "Đã từ chối"} ${st.name}`, () => setMembers(before));
  }

  function copyLink() {
    void navigator.clipboard?.writeText(`https://edupilot.test/join/${code}`).catch(() => undefined);
    undo.push("Đã sao chép link mời vào lớp (mô phỏng)");
  }

  return (
    <Page width="wide">
      <PageHeader
        title="Thành viên lớp"
        back={{ href: "/", label: "Hôm nay" }}
        description={`${course.label} · ${roster.length}/${course.size} chỗ đã dùng`}
        actions={
          isTeacher && (
            <>
              <Button onClick={() => setInviting((v) => !v)}>Mời trợ giảng</Button>
              <Button onClick={() => setRegenerating(true)}>Tạo lại mã</Button>
            </>
          )
        }
      />
      <PageState
        state={routeState}
        loading={<Skeleton lines={8} />}
        empty={<EmptyState title="Lớp này chưa có thành viên nào">Chia sẻ mã tham gia để sinh viên vào lớp.</EmptyState>}
        error={{ problem: "Không tải được danh sách thành viên.", recovery: "Yêu cầu vào lớp vẫn được giữ. Thử lại, hoặc mở lại sau ít phút." }}
      >
        {undo.node}

        <Section title="Mã tham gia" description={course.requireApproval ? "Sinh viên nhập mã rồi chờ bạn duyệt." : "Sinh viên nhập mã là vào lớp ngay."}>
          <div className={s.codeRow}>
            <span className={s.code}>{code}</span>
            <Button variant="text" size="sm" icon={<Link2 aria-hidden />} onClick={copyLink}>
              Sao chép link
            </Button>
            <Button
              variant="text"
              size="sm"
              icon={<Copy aria-hidden />}
              onClick={() => {
                void navigator.clipboard?.writeText(code).catch(() => undefined);
                undo.push("Đã sao chép mã tham gia (mô phỏng)");
              }}
            >
              Sao chép mã
            </Button>
          </div>
          <p className="ep-meta">
            {course.requireApproval ? "Lớp bật duyệt trước khi vào" : "Lớp không bật duyệt"} · thiết lập này do giảng viên đổi ở phần cài đặt lớp.
          </p>
          {inviting && isTeacher && (
            <div className={s.invite}>
              <Field label="Email trợ giảng" helper="Trợ giảng xem được lớp này nhưng không sửa được điểm.">
                {(id, describedBy) => (
                  <Input id={id} aria-describedby={describedBy} type="email" value={email} placeholder="ten@edupilot.test" onChange={(e) => setEmail(e.target.value)} />
                )}
              </Field>
              <Button
                variant="primary"
                disabled={!email.includes("@")}
                onClick={() => {
                  undo.push(`Đã gửi lời mời trợ giảng tới ${email} (mô phỏng)`);
                  setEmail("");
                  setInviting(false);
                }}
              >
                Gửi lời mời
              </Button>
            </div>
          )}
        </Section>

        <Section id="pending" title={`Yêu cầu chờ duyệt${pending.length > 0 ? ` (${pending.length})` : ""}`}>
          {pending.length === 0 ? (
            <EmptyState title="Không có yêu cầu chờ">Khi sinh viên nhập mã {code}, yêu cầu sẽ xuất hiện ở đây để bạn duyệt.</EmptyState>
          ) : (
            <ActionList label="Yêu cầu vào lớp">
              {pending.map((st) => (
                <ActionRow
                  key={st.id}
                  tone="amber"
                  title={st.name}
                  context={st.email}
                  meta={`${st.code} · gửi yêu cầu ${PENDING_STUDENT_IDS.includes(st.id) ? "hôm qua" : "vừa xong"}`}
                  action={
                    <span className={s.rowActions}>
                      <Button size="sm" onClick={() => decide(st, true)}>
                        Duyệt
                      </Button>
                      <Button size="sm" variant="text" onClick={() => decide(st, false)}>
                        Từ chối
                      </Button>
                    </span>
                  }
                />
              ))}
            </ActionList>
          )}
        </Section>

        <Section title={`Thành viên (${roster.length})`}>
          <DataTable
            caption={`Thành viên lớp ${course.code}`}
            dense
            columns={[
              { key: "name", header: "Sinh viên", frozen: true, width: "30%", render: (st) => st.name },
              { key: "code", header: "MSSV", render: (st) => st.code },
              { key: "mail", header: "Email", render: (st) => st.email },
              {
                key: "act",
                header: "",
                align: "end",
                width: "56px",
                render: (st) =>
                  isTeacher ? <OverflowMenu label={`Hành động với ${st.name}`} items={[{ label: "Mời ra khỏi lớp", danger: true, onSelect: () => setRemoving(st) }]} /> : null,
              },
            ]}
            rows={roster}
            rowKey={(st) => st.id}
            rowAttrs={(st) => ({ "data-part": "student-row", "data-student-id": st.id })}
            empty={<EmptyState title="Chưa có sinh viên nào trong lớp">Chia sẻ mã {code} để sinh viên vào lớp.</EmptyState>}
          />
        </Section>
      </PageState>

      {isTeacher && (
        <>
          <ConfirmIrreversible
            open={regenerating}
            onClose={() => setRegenerating(false)}
            title="Tạo lại mã tham gia?"
            consequence={`Mã cũ ${code} vô hiệu ngay. Ai đang giữ link cũ sẽ không vào được lớp và cần mã mới.`}
            confirmLabel="Tạo lại mã"
            onConfirm={() => {
              const next = Array.from({ length: 7 }, () => CODE_CHARS[Math.floor(Math.random() * CODE_CHARS.length)]).join("");
              setMembers((prev) => ({ ...prev, joinCodes: { ...prev.joinCodes, [course.id]: next } }));
              undo.push(`Mã mới của lớp là ${next}`);
            }}
          />
          <ConfirmIrreversible
            open={Boolean(removing)}
            onClose={() => setRemoving(null)}
            title={`Mời ${removing?.name ?? ""} ra khỏi lớp?`}
            consequence="Sinh viên mất quyền xem tài liệu và bài tập của lớp ngay; điểm đã ghi vẫn được giữ."
            confirmLabel="Mời ra khỏi lớp"
            onConfirm={() => {
              if (removing) setRemoved((prev) => [...prev, removing.id]);
              setRemoving(null);
            }}
          />
        </>
      )}
    </Page>
  );
}
