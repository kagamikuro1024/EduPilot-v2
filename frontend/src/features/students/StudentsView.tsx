"use client";

import { Search } from "lucide-react";
import { fmtScore, type Student } from "@/mock/core";
import { attendanceStats, qtOf, type AttendanceStats } from "@/mock/grades";
import { STUDENT_FILTERS, matchesFilter, rosterOf, type StudentFilter } from "@/mock/roster";
import { BT03_SEED } from "@/mock/assess";
import {
  ATTENDANCE_SEED,
  KEYS,
  MEMBERS_SEED,
  SCHEMES_SEED,
  type AttendanceState,
  type Bt03State,
  type MembersState,
  type SchemesState,
} from "@/mock/state";
import { useSession } from "@/shared/session/session";
import { useDemoSlice } from "@/shared/state/demo";
import {
  Button,
  DataTable,
  EmptyState,
  FilterChips,
  InlineNotice,
  Input,
  Page,
  PageHeader,
  PageState,
  PrivateMark,
  Skeleton,
  StatusText,
  Toolbar,
  useRouteState,
  type Column,
} from "@/shared/ui";
import s from "./StudentsView.module.css";

type Row = { student: Student; stats: AttendanceStats; qt: number | null };

/** Lát riêng của màn: giữ ô tìm kiếm và chip khi quay lại từ hồ sơ (DESIGN §14.6). */
const FILTER_KEY = "students.filter";
type FilterState = { q: string; chips: StudentFilter[] };

export function StudentsView() {
  const { course } = useSession();
  const [attendance] = useDemoSlice<AttendanceState>(KEYS.attendance, ATTENDANCE_SEED);
  const [members] = useDemoSlice<MembersState>(KEYS.members, MEMBERS_SEED);
  const [bt03] = useDemoSlice<Bt03State>(KEYS.bt03, BT03_SEED);
  const [schemes] = useDemoSlice<SchemesState>(KEYS.schemes, SCHEMES_SEED);
  const [{ q, chips }, setFilter] = useDemoSlice<FilterState>(FILTER_KEY, { q: "", chips: [] });
  const routeState = useRouteState();

  const hasScheme = schemes[course.id]?.status === "confirmed";
  const rows: Row[] = rosterOf(course.id, members).map((student) => {
    const stats = attendanceStats(student, attendance);
    return { student, stats, qt: hasScheme ? qtOf(student.id, attendance, bt03).qt : null };
  });

  const needle = q.trim().toLowerCase();
  const shown = rows.filter(
    (r) =>
      (needle === "" || r.student.name.toLowerCase().includes(needle) || r.student.code.includes(needle)) &&
      chips.every((c) => matchesFilter(c, r.student, r.stats)),
  );

  const columns: Column<Row>[] = [
    {
      key: "name",
      header: "Sinh viên",
      frozen: true,
      width: "26%",
      render: (r) => (
        <span className={s.name}>
          <span className="ep-item-title">{r.student.name}</span>
          <span className="ep-meta">{r.student.code}</span>
        </span>
      ),
    },
    { key: "att", header: "Chuyên cần", render: (r) => `${r.stats.recorded - r.stats.absences}/${r.stats.recorded} buổi` },
    { key: "qt", header: "Điểm hiện tại", align: "end", render: (r) => (r.qt === null ? "—" : fmtScore(r.qt)) },
    { key: "speak", header: "Phát biểu", align: "end", hideOnMobile: true, render: (r) => `${r.stats.speaks} lần` },
    { key: "act", header: "Hoạt động học", align: "end", hideOnMobile: true, render: (r) => `${r.student.activityMin} phút/tuần` },
    {
      key: "risk",
      header: "Rủi ro",
      render: (r) =>
        r.student.risk === "none" ? (
          <span className="ep-meta">Không</span>
        ) : (
          <StatusText tone={r.student.risk === "high" ? "red" : "amber"}>{r.student.risk === "high" ? "Cần chú ý" : "Theo dõi"}</StatusText>
        ),
    },
  ];

  return (
    <Page width="wide">
      <PageHeader
        title="Sinh viên"
        description={`${rows.length} sinh viên · ${course.label}`}
        meta={<PrivateMark>Chuyên cần, điểm và nhãn rủi ro chỉ giảng viên/TA thấy</PrivateMark>}
      />
      <PageState
        state={routeState}
        loading={<Skeleton lines={10} />}
        empty={<EmptyState title="Lớp này chưa có sinh viên nào">Chia sẻ mã tham gia để sinh viên vào lớp, danh sách sẽ hiện ở đây.</EmptyState>}
        error={{ problem: "Không tải được danh sách sinh viên.", recovery: "Dữ liệu lớp vẫn an toàn. Thử lại, hoặc mở lại sau ít phút." }}
      >
        <Toolbar
          end={
            <FilterChips
              label="Lọc nhanh"
              value={chips}
              onChange={(next) => setFilter((p) => ({ ...p, chips: next }))}
              options={STUDENT_FILTERS.map((f) => ({ value: f.value, label: f.label, count: rows.filter((r) => matchesFilter(f.value, r.student, r.stats)).length }))}
            />
          }
        >
          <label className={s.search}>
            <Search aria-hidden />
            <span className="ep-sr-only">Tìm theo tên hoặc mã số sinh viên</span>
            <Input
              type="search"
              value={q}
              placeholder="Tìm theo tên hoặc MSSV"
              onChange={(e) => setFilter((p) => ({ ...p, q: e.target.value }))}
            />
          </label>
        </Toolbar>

        {!hasScheme && (
          <InlineNotice tone="info" compact>
            Lớp này chưa có công thức điểm chính thức nên cột “Điểm hiện tại” còn trống.
          </InlineNotice>
        )}

        <DataTable
          caption={`Sinh viên lớp ${course.code}`}
          columns={columns}
          rows={shown}
          rowKey={(r) => r.student.id}
          rowHref={(r) => `/students/${r.student.id}`}
          empty={
            <EmptyState
              title="Không có sinh viên nào khớp"
              action={
                <Button onClick={() => setFilter({ q: "", chips: [] })}>Xoá bộ lọc</Button>
              }
            >
              Thử tìm bằng một phần tên hoặc mã số sinh viên, hoặc bỏ bớt chip lọc.
            </EmptyState>
          }
        />
      </PageState>
    </Page>
  );
}
