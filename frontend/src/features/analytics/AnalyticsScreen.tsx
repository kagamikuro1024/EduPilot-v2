"use client";

import { useState } from "react";
import { analyticsFor, fmtVnd, type AnalyticsRange } from "@/mock/analytics";
import { fmtTime } from "@/mock/core";
import { ticketStats } from "@/mock/derive";
import { KEYS, type Ticket } from "@/mock/state";
import { mergeTickets } from "@/mock/support";
import { useSession } from "@/shared/session/session";
import { useSimNow } from "@/shared/state/clock";
import { useDemoSlice } from "@/shared/state/demo";
import { BarList, DefinitionList, EmptyState, Page, PageHeader, PageState, Panel, Section, SegmentedControl, Skeleton, TrendChart } from "@/shared/ui";
import s from "./analytics.module.css";

/** "1.424" — nhóm nghìn giống `fmtVnd` để server và trình duyệt cho ra cùng một chuỗi. */
const fmtNum = (n: number) => n.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ".");

export function AnalyticsScreen() {
  const { role, course, courses, isAll } = useSession();
  const [stored] = useDemoSlice<Ticket[]>(KEYS.tickets, []);
  const [range, setRange] = useState<AnalyticsRange>("7");
  const nowMs = useSimNow();
  const courseIds = isAll ? courses.map((c) => c.id) : [course.id];
  const a = analyticsFor(course.id, range, ticketStats(mergeTickets(stored), courseIds, nowMs));
  const days = range === "7" ? "7 ngày" : "30 ngày";

  return (
    <Page>
      <PageHeader
        title="Số liệu lớp học"
        description="Xu hướng của cả lớp để điều chỉnh cách dạy — không phải bảng theo dõi từng sinh viên."
        meta={
          <>
            <span>{course.label}</span>
            <span>Tính tới hôm nay, {fmtTime(new Date(nowMs))}</span>
          </>
        }
        actions={
          <SegmentedControl
            label="Khoảng thời gian"
            value={range}
            onChange={setRange}
            options={[
              { value: "7", label: "7 ngày" },
              { value: "30", label: "30 ngày" },
            ]}
          />
        }
      />

      <PageState
        loading={
          <Panel>
            <div className={s.loading}>
              <Skeleton lines={2} />
              <Skeleton lines={4} />
              <Skeleton lines={4} />
            </div>
          </Panel>
        }
        empty={
          <Panel><EmptyState title={`Lớp ${course.code} chưa đủ dữ liệu để vẽ xu hướng`}>
            Số liệu xuất hiện sau khi lớp có câu hỏi và bài nộp đầu tiên. Mời sinh viên vào lớp bằng mã tham gia, hoặc mở một bài tập để bắt đầu.
          </EmptyState></Panel>
        }
        error={{
          problem: "Không tải được số liệu của lớp.",
          recovery: "Việc tổng hợp số liệu chạy mỗi giờ một lần và đang chậm. Thử lại sau vài phút; dữ liệu lớp không bị ảnh hưởng.",
        }}
      >
        <Section panel title="Hoạt động học" description={`Câu hỏi sinh viên gửi trong ${days}`}>
          <p className={s.lead}>
            {`${fmtNum(a.questions)} câu hỏi trong ${days}, trung bình ${a.questionsPerDay} câu mỗi ngày. ${a.activeStudents}/${a.students} sinh viên có hỏi ít nhất một câu; ${a.students - a.activeStudents} bạn chưa hỏi câu nào.`}
          </p>
          <TrendChart points={a.trend} label={`Câu hỏi mỗi ngày trong ${days}`} format={(v) => `${v} câu`} />
          <h3 className={s.subhead}>Chủ đề được hỏi nhiều nhất</h3>
          <BarList items={a.topics.map((t) => ({ label: t.label, value: t.value }))} format={(v) => `${v} câu`} />
        </Section>

        <Section panel title="Hỗ trợ" description="Câu AI không tự trả lời được và thời gian lớp chờ bạn">
          <p className={s.lead}>
            {`AI tự trả lời ${a.aiSharePct}% trong ${fmtNum(a.questions)} câu hỏi; ${a.escalated} câu chuyển sang giảng viên vì không đủ chắc chắn hoặc sinh viên tự yêu cầu.`}
          </p>
          <DefinitionList
            items={[
              { term: "Thời gian trả lời (trung vị)", value: a.medianReply },
              { term: "Câu chờ quá 24 giờ", value: a.overdue24 === 0 ? "Không có" : `${a.overdue24} câu` },
              { term: "Câu đã chuyển giảng viên", value: `${a.escalated} câu trong ${days}` },
            ]}
          />
        </Section>

        <Section panel title="Bảo vệ thông tin cá nhân" description="Thông tin cá nhân bị che trước khi câu hỏi đi tới model">
          <p className={s.lead}>
            <strong>{a.piiDetected} lần</strong> hệ thống phát hiện thông tin cá nhân trong {days} và che lại trước khi gửi đi. <strong>{a.piiLeaked} lần</strong> lọt ra ngoài.
          </p>
          {a.piiChannels.length > 0 && <BarList items={a.piiChannels.map((c) => ({ label: c.label, value: c.value, tone: "amber" as const }))} format={(v) => `${v} lần`} />}
        </Section>

        <Section panel title="Chất lượng chấm" description="AI chỉ ra bản nháp; con số dưới đây cho biết bạn phải sửa nhiều hay ít">
          {a.submissions === 0 ? (
            <p className={s.lead}>Lớp chưa có bài nộp nào được chấm nên chưa đo được chất lượng bản nháp.</p>
          ) : (
            <>
              <p className={s.lead}>
                <strong>{a.submissions} bài</strong> có bản nháp của AI; bạn sửa điểm <strong>{a.edited} bài</strong> ({a.editRate}%). Chênh lệch trung bình giữa nháp và điểm công bố: {a.avgGap}.
              </p>
              <DefinitionList
                items={[
                  { term: "Bài có bản nháp", value: `${a.submissions} bài` },
                  { term: "Bài giảng viên sửa điểm", value: `${a.edited} bài · ${a.editRate}%` },
                  { term: "Chênh lệch trung bình", value: a.avgGap },
                ]}
              />
            </>
          )}
        </Section>

        {role === "teacher" && (
          <Section panel title="Chi phí" description="Tiền trả cho model trong khoảng đang xem">
            <p className={s.lead}>
              <strong>{fmtVnd(a.cost)}</strong> trong {days}, khoảng {fmtVnd(a.costPerStudent)} cho mỗi sinh viên. Trần chi tiêu do quản trị viên hệ thống đặt.
            </p>
          </Section>
        )}
      </PageState>
    </Page>
  );
}
