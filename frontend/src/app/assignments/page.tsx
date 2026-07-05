"use client";

import { useEffect, useMemo, useState } from "react";
import {
  BookOpen,
  Calendar as CalendarIcon,
  ChevronLeft,
  ChevronRight,
  ChevronDown,
  Clock,
  Download,
  MapPin,
} from "lucide-react";
import { WorkspaceLayout } from "@/components/layout/WorkspaceLayout";
import { useAuthStore } from "@/store/authStore";
import javaClient from "@/services/javaClient";

// -----------------------------------------------------------------------------
// Data models (mirror future API payloads)
// -----------------------------------------------------------------------------

export type CalendarEventVariant = "warning" | "critical" | "success";

export interface CalendarEvent {
  id: string;
  /** Day within the displayed month (1–31) */
  dayOfMonth: number;
  timeLabel: string;
  shortTitle: string;
  variant: CalendarEventVariant;
}

export interface StudentProfileHeader {
  displayName: string;
  courseLabel: string;
  avatarInitials: string;
}

export interface QuizAlertBanner {
  id: string;
  title: string;
  body: string;
}

export interface FaqItem {
  id: string;
  question: string;
  answer: string;
}

export type QuickResourceKind = "syllabus" | "lab" | "toolchain";

export interface QuickResource {
  id: string;
  kind: QuickResourceKind;
  label: string;
}

export interface StudentProgressSnapshot {
  completionPercent: number;
  averageGradeLabel: string;
  rankingLabel: string;
}

export type CalendarViewMode = "month" | "week";

export interface CalendarMonthMeta {
  label: string;
  year: number;
  /** 0 = January */
  monthIndex: number;
  /** Highlighted “today” in mock */
  todayDayOfMonth: number;
}

// -----------------------------------------------------------------------------
// Mock data (module scope)
// -----------------------------------------------------------------------------

// TODO: API INTEGRATION - Fetch this data from GET /api/v1/students/me/upcoming-assessments
const MOCK_QUIZ_ALERT: QuizAlertBanner = {
  id: "alert-1",
  title: "Next Quiz Starts in 2 Hours.",
  body: "Quiz 2: Operating Systems Basics will open at 10:00 AM. Ensure your browser is updated.",
};

// TODO: API INTEGRATION - Fetch this data from GET /api/v1/courses/{courseId}/calendar?month=2024-10
const MOCK_CALENDAR_META: CalendarMonthMeta = {
  label: "October",
  year: 2024,
  monthIndex: 9,
  todayDayOfMonth: 24,
};

// TODO: API INTEGRATION - Fetch this data from GET /api/v1/courses/{courseId}/calendar/events?month=2024-10
const MOCK_CALENDAR_EVENTS: CalendarEvent[] = [
  {
    id: "e1",
    dayOfMonth: 18,
    timeLabel: "11:59 PM",
    shortTitle: "Lab 3: S...",
    variant: "warning",
  },
  {
    id: "e2",
    dayOfMonth: 24,
    timeLabel: "11:59 PM",
    shortTitle: "Lab 4: M...",
    variant: "warning",
  },
  {
    id: "e3",
    dayOfMonth: 25,
    timeLabel: "10:00 AM",
    shortTitle: "Quiz 2: O...",
    variant: "critical",
  },
  {
    id: "e4",
    dayOfMonth: 25,
    timeLabel: "05:00 PM",
    shortTitle: "PS 3: Ne...",
    variant: "success",
  },
];

// TODO: API INTEGRATION - Fetch this data from GET /api/v1/courses/{courseId}/logistics-faq
const MOCK_FAQ_ITEMS: FaqItem[] = [
  {
    id: "faq-1",
    question: "Where is the physical lab located?",
    answer:
      "All in-person labs meet in Building 4, Room 102. Please bring your student ID and arrive 5 minutes early for attendance.",
  },
  {
    id: "faq-2",
    question: "How do I access the remote compiler?",
    answer: "Use the VPN link on the course portal, then SSH to compiler.cs101.edu with your university credentials.",
  },
  {
    id: "faq-3",
    question: "What is the late submission policy?",
    answer: "Submissions within 24 hours after the deadline receive a 10% penalty unless prior approval exists.",
  },
  {
    id: "faq-4",
    question: "How is the final grade calculated?",
    answer: "40% assignments, 25% quizzes, 25% final project, 10% participation.",
  },
];

// TODO: API INTEGRATION - Fetch this data from GET /api/v1/courses/{courseId}/quick-resources
const MOCK_QUICK_RESOURCES: QuickResource[] = [
  { id: "qr-1", kind: "syllabus", label: "Course Syllabus PDF" },
  { id: "qr-2", kind: "lab", label: "Find My Lab Room" },
  { id: "qr-3", kind: "toolchain", label: "Download Toolchain" },
];

// TODO: API INTEGRATION - Fetch this data from GET /api/v1/students/me/progress?courseId=...
const MOCK_STUDENT_PROGRESS: StudentProgressSnapshot = {
  completionPercent: 65,
  averageGradeLabel: "A-",
  rankingLabel: "#12",
};

// TODO: API INTEGRATION - Fetch this data from GET /api/v1/meta/client-version (or static config)
const MOCK_CLIENT_STATUS = {
  label: "System Online",
  version: "Version 1.2.4-stable",
} as const;

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

function eventBadgeClasses(variant: CalendarEventVariant): string {
  switch (variant) {
    case "warning":
      return "bg-amber-50 text-amber-900 ring-amber-200";
    case "critical":
      return "bg-rose-50 text-rose-900 ring-rose-200";
    case "success":
      return "bg-emerald-50 text-emerald-900 ring-emerald-200";
  }
}

function buildMonthCells(year: number, monthIndex: number, today: Date | null): { day: number | null; isToday: boolean }[] {
  const firstWeekday = new Date(year, monthIndex, 1).getDay();
  const daysInMonth = new Date(year, monthIndex + 1, 0).getDate();
  const cells: { day: number | null; isToday: boolean }[] = [];
  
  for (let i = 0; i < firstWeekday; i++) {
    cells.push({ day: null, isToday: false });
  }

  for (let d = 1; d <= daysInMonth; d++) {
    const isCurrentToday = today && 
      today.getFullYear() === year && 
      today.getMonth() === monthIndex && 
      today.getDate() === d;
    cells.push({ day: d, isToday: !!isCurrentToday });
  }
  while (cells.length % 7 !== 0) {
    cells.push({ day: null, isToday: false });
  }
  while (cells.length < 42) {
    cells.push({ day: null, isToday: false });
  }
  return cells;
}

function eventsForDay(day: number | null, events: CalendarEvent[]): CalendarEvent[] {
  if (day == null) return [];
  return events.filter((e) => e.dayOfMonth === day);
}

function QuickResourceIcon({ kind }: { kind: QuickResourceKind }) {
  const cls = "h-5 w-5 text-blue-600";
  switch (kind) {
    case "syllabus":
      return <BookOpen className={cls} aria-hidden />;
    case "lab":
      return <MapPin className={cls} aria-hidden />;
    case "toolchain":
      return <Download className={cls} aria-hidden />;
  }
}

// -----------------------------------------------------------------------------
// Page
// -----------------------------------------------------------------------------

export default function AssignmentsPage() {
  const [calendarMode, setCalendarMode] = useState<CalendarViewMode>("month");
  const [openFaqId, setOpenFaqId] = useState<string | null>("faq-1");
  const fullName = useAuthStore((s) => s.fullName);
  const role = useAuthStore((s) => s.role);
  const canCreate = role === "TA" || role === "ADMIN";

  const [viewDate, setViewDate] = useState(() => new Date()); // Default to current date
  const [events, setEvents] = useState<CalendarEvent[]>(MOCK_CALENDAR_EVENTS);
  const [showModal, setShowModal] = useState(false);
  const [todayRef] = useState(() => new Date());

  useEffect(() => {
    javaClient.get("/api/v1/assignments")
      .then((res) => res.data)
      .then((data) => {
        if (Array.isArray(data)) {
          const mapped: CalendarEvent[] = data.map((item: any, idx: number) => {
            const date = new Date(item.due_date);
            const day = isNaN(date.getDate()) ? 24 : date.getDate();
            const timeStr = isNaN(date.getHours()) ? "11:59 PM" : date.toLocaleTimeString("en-US", { hour: "2-digit", minute: "2-digit" });
            // We should probably also parse month/year to properly place on correct monthly grid view
            // For now, map current month logic or parse the real full timestamp:
            return {
              id: item.id || `db-${idx}`,
              dayOfMonth: day,
              // Storing raw full date helps filter by currently viewed month
              fullDate: date,
              timeLabel: timeStr,
              shortTitle: item.title,
              variant: "warning",
            };
          });
          setEvents(mapped);
        }
      })
      .catch((err) => {
        console.warn("Failed to fetch database assignments, using mock calendar events:", err);
      });
  }, []);

  const handlePrevMonth = () => {
    setViewDate(prev => new Date(prev.getFullYear(), prev.getMonth() - 1, 1));
  };

  const handleNextMonth = () => {
    setViewDate(prev => new Date(prev.getFullYear(), prev.getMonth() + 1, 1));
  };

  const handleCreateEvent = async (title: string, dateStr: string, timeStr: string, rule: string) => {
    const date = new Date(`${dateStr}T${timeStr}`);
    const day = isNaN(date.getDate()) ? 24 : date.getDate();
    const isoString = isNaN(date.getTime()) ? new Date().toISOString() : date.toISOString();
    const fullDate = isNaN(date.getTime()) ? new Date() : date;

    const payload = {
      title,
      description: "TA Created Assignment",
      due_date: isoString,
      late_penalty_rule: rule || "Trừ 10% mỗi ngày",
    };

    const newEvent: CalendarEvent = {
      id: `local-created-${Date.now()}`,
      dayOfMonth: day,
      fullDate,
      timeLabel: timeStr,
      shortTitle: title,
      variant: "warning",
    } as any; // Extend calendar interface internally
    setEvents((prev) => [...prev, newEvent]);

    try {
      await javaClient.post("/api/v1/assignments", payload);
    } catch (err) {
      console.error("Failed to post assignment to database:", err);
    }
  };

  // TODO: API INTEGRATION - Replace mocks with hooks calling GET /api/v1/...
  const profile = useMemo(() => {
    const name = fullName || "Student";
    const initials = name
      .trim()
      .split(/\s+/)
      .slice(0, 2)
      .map((n) => n[0]?.toUpperCase())
      .filter(Boolean)
      .join("");
    
    return {
      displayName: name,
      courseLabel: "CS101: Intro to IT", // Keeping this for now as there's no backend for course selection yet
      avatarInitials: initials || "ST",
    };
  }, [fullName]);
  const quizAlert = MOCK_QUIZ_ALERT;
  const currentYear = viewDate.getFullYear();
  const currentMonthIndex = viewDate.getMonth();
  const currentMonthLabel = viewDate.toLocaleString("en-US", { month: "long" });

  const calendarEvents = useMemo(() => {
    // Filter events applicable to the currently viewed month/year
    return events.filter(ev => {
      // Mock events (e1..e4) exist only for fallback display in October 2024 mock view
      if (["e1","e2","e3","e4"].includes(ev.id)) {
        const now = new Date();
        return currentYear === now.getFullYear() && currentMonthIndex === now.getMonth();
      }
      // For DB retrieved events, verify they fall into the selected month
      const evtAny = ev as any;
      if (evtAny.fullDate instanceof Date && !isNaN(evtAny.fullDate.getTime())) {
        return evtAny.fullDate.getFullYear() === currentYear && evtAny.fullDate.getMonth() === currentMonthIndex;
      }
      // Fallback fallback
      return true;
    });
  }, [events, currentYear, currentMonthIndex]);

  const faqItems = MOCK_FAQ_ITEMS;
  const quickResources = MOCK_QUICK_RESOURCES;
  const progress = MOCK_STUDENT_PROGRESS;
  const clientStatus = MOCK_CLIENT_STATUS;

  const monthCells = useMemo(() => {
    return buildMonthCells(currentYear, currentMonthIndex, todayRef);
  }, [currentYear, currentMonthIndex, todayRef]);

  return (
    <WorkspaceLayout
      footerLine2="rights"
      searchPlaceholder="Search courses, assignments…"
      studentProfile={profile}
      footerFloating={
        <div className="pointer-events-none fixed bottom-4 right-4 flex items-center gap-2 rounded-full border border-slate-200 bg-white/95 px-3 py-1.5 text-xs font-medium text-slate-600 shadow-md backdrop-blur">
          <span className="h-2 w-2 rounded-full bg-emerald-500" aria-hidden />
          <span>{clientStatus.label}</span>
          <span className="text-slate-400">·</span>
          <span className="text-slate-500">{clientStatus.version}</span>
        </div>
      }
      sidePanel={
        <div className="space-y-5 p-5">
          <div>
            <h2 className="text-sm font-semibold text-slate-900">Logistics FAQ</h2>
            <p className="mt-0.5 text-xs text-slate-500">Answers update automatically when policies change.</p>
          </div>
          <div className="space-y-2">
            {faqItems.map((item) => {
              const open = openFaqId === item.id;
              return (
                <div
                  key={item.id}
                  className="overflow-hidden rounded-lg border border-slate-200 bg-white shadow-sm"
                >
                  <button
                    type="button"
                    onClick={() => setOpenFaqId(open ? null : item.id)}
                    className="flex w-full items-center justify-between gap-2 px-3 py-2.5 text-left text-sm font-semibold text-slate-800 hover:bg-slate-50"
                  >
                    <span>{item.question}</span>
                    <ChevronDown
                      className={`h-4 w-4 shrink-0 text-slate-400 transition ${open ? "rotate-180" : ""}`}
                    />
                  </button>
                  {open && (
                    <div className="border-t border-slate-100 px-3 py-2.5 text-sm leading-relaxed text-slate-600">
                      {item.answer}
                    </div>
                  )}
                </div>
              );
            })}
          </div>

          <div>
            <h2 className="text-sm font-semibold text-slate-900">Quick Resources</h2>
            <div className="mt-3 space-y-2">
              {quickResources.map((r) => (
                <button
                  key={r.id}
                  type="button"
                  className="flex w-full items-center gap-3 rounded-xl border border-slate-200 bg-slate-50/80 px-4 py-3 text-left text-sm font-semibold text-slate-800 shadow-sm transition hover:border-blue-200 hover:bg-blue-50/50"
                >
                  <span className="flex h-10 w-10 items-center justify-center rounded-lg bg-white shadow-sm ring-1 ring-slate-100">
                    <QuickResourceIcon kind={r.kind} />
                  </span>
                  {r.label}
                </button>
              ))}
            </div>
          </div>

          <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
            <p className="text-[11px] font-bold uppercase tracking-wide text-blue-600">Your Progress</p>
            <p className="mt-3 text-xs font-medium text-slate-500">Course Completion</p>
            <div className="mt-2 h-2 w-full overflow-hidden rounded-full bg-slate-100">
              <div
                className="h-full rounded-full bg-blue-600 transition-all"
                style={{ width: `${progress.completionPercent}%` }}
              />
            </div>
            <div className="mt-4 grid grid-cols-2 gap-3">
              <div className="rounded-lg border border-slate-100 bg-slate-50 px-3 py-2 text-center">
                <p className="text-[10px] font-bold uppercase tracking-wide text-slate-500">Average</p>
                <p className="mt-1 text-lg font-bold text-slate-900">{progress.averageGradeLabel}</p>
              </div>
              <div className="rounded-lg border border-slate-100 bg-slate-50 px-3 py-2 text-center">
                <p className="text-[10px] font-bold uppercase tracking-wide text-slate-500">Ranking</p>
                <p className="mt-1 text-lg font-bold text-slate-900">{progress.rankingLabel}</p>
              </div>
            </div>
          </div>
        </div>
      }
    >
      <div className="px-6 py-6">
        <div className="mx-auto max-w-4xl space-y-5">
              <div>
                <h1 className="text-2xl font-bold tracking-tight text-slate-900">
                  Assignments & Course Assistant
                </h1>
                <p className="mt-1 text-sm text-slate-600">
                  Manage your deadlines and access course logistics from one central hub.
                </p>
              </div>

              <div className="flex gap-3 rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
                <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-slate-50 ring-1 ring-slate-100">
                  <Clock className="h-5 w-5 text-slate-600" aria-hidden />
                </div>
                <div>
                  <p className="text-sm font-semibold text-slate-900">{quizAlert.title}</p>
                  <p className="mt-1 text-sm leading-relaxed text-slate-600">{quizAlert.body}</p>
                </div>
              </div>

              <section className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
                <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                  <div className="flex items-center gap-2">
                    <CalendarIcon className="h-5 w-5 text-slate-500" />
                    <h2 className="text-lg font-semibold text-slate-900">Assignment Calendar</h2>
                    {canCreate && (
                      <button
                        type="button"
                        onClick={() => setShowModal(true)}
                        className="ml-3 rounded-lg bg-blue-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-blue-700 active:scale-95 transition-all cursor-pointer"
                      >
                        + Create Deadline
                      </button>
                    )}
                  </div>
                  <div className="flex flex-wrap items-center gap-2">
                    <div className="inline-flex rounded-lg border border-slate-200 bg-slate-50 p-0.5">
                      <button
                        type="button"
                        onClick={() => setCalendarMode("month")}
                        className={`rounded-md px-3 py-1.5 text-xs font-semibold transition ${
                          calendarMode === "month"
                            ? "bg-slate-900 text-white shadow-sm"
                            : "text-slate-600 hover:text-slate-900"
                        }`}
                      >
                        Month
                      </button>
                      <button
                        type="button"
                        onClick={() => setCalendarMode("week")}
                        className={`rounded-md px-3 py-1.5 text-xs font-semibold transition ${
                          calendarMode === "week"
                            ? "bg-slate-900 text-white shadow-sm"
                            : "text-slate-600 hover:text-slate-900"
                        }`}
                      >
                        Week
                      </button>
                    </div>
                    <div className="flex items-center gap-1 rounded-lg border border-slate-200 bg-white px-2 py-1 shadow-sm">
                      <button
                        type="button"
                        onClick={handlePrevMonth}
                        className="rounded p-1 text-slate-500 hover:bg-slate-50 hover:text-slate-800 transition-colors"
                        aria-label="Previous month"
                      >
                        <ChevronLeft className="h-4 w-4" />
                      </button>
                      <span className="min-w-[8rem] text-center text-sm font-semibold text-slate-800 select-none">
                        {currentMonthLabel} {currentYear}
                      </span>
                      <button
                        type="button"
                        onClick={handleNextMonth}
                        className="rounded p-1 text-slate-500 hover:bg-slate-50 hover:text-slate-800 transition-colors"
                        aria-label="Next month"
                      >
                        <ChevronRight className="h-4 w-4" />
                      </button>
                    </div>
                  </div>
                </div>

                {calendarMode === "week" ? (
                  <p className="mt-6 rounded-lg border border-dashed border-slate-200 bg-slate-50 py-8 text-center text-sm text-slate-500">
                    Week view placeholder — wire to API when course schedule endpoints are ready.
                  </p>
                ) : (
                  <div className="mt-4">
                    <div className="grid grid-cols-7 gap-px overflow-hidden rounded-lg border border-slate-200 bg-slate-200 text-center text-[11px] font-semibold uppercase tracking-wide text-slate-500">
                      {["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"].map((d) => (
                        <div key={d} className="bg-slate-50 py-2">
                          {d}
                        </div>
                      ))}
                    </div>
                    <div className="grid grid-cols-7 gap-px rounded-b-lg border border-t-0 border-slate-200 bg-slate-200">
                      {monthCells.map((cell, idx) => {
                        const dayEvents = eventsForDay(cell.day, calendarEvents);
                        return (
                          <div
                            key={idx}
                            className={`min-h-[5.5rem] bg-white p-1.5 text-left ${
                              cell.day ? "hover:bg-slate-50/80" : ""
                            }`}
                          >
                            {cell.day != null && (
                              <>
                                <div className="flex items-center justify-between gap-1">
                                  <span
                                    className={`flex h-7 w-7 items-center justify-center rounded-full text-xs font-semibold ${
                                      cell.isToday
                                        ? "bg-blue-600 text-white shadow-sm"
                                        : "text-slate-700"
                                    }`}
                                  >
                                    {cell.day}
                                  </span>
                                </div>
                                <div className="mt-1 space-y-1">
                                  {dayEvents.map((ev) => (
                                    <div
                                      key={ev.id}
                                      className={`truncate rounded px-1 py-0.5 text-[10px] font-semibold ring-1 ${eventBadgeClasses(
                                        ev.variant,
                                      )}`}
                                      title={`${ev.timeLabel} ${ev.shortTitle}`}
                                    >
                                      {ev.timeLabel} {ev.shortTitle}
                                    </div>
                                  ))}
                                </div>
                              </>
                            )}
                          </div>
                        );
                      })}
                    </div>
                  </div>
                )}
              </section>
        </div>
      </div>
      {showModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/60 p-4 backdrop-blur-sm animate-in fade-in duration-200">
          <div className="w-full max-w-md rounded-2xl border border-slate-200 bg-white p-6 shadow-2xl animate-in zoom-in-95 duration-200">
            <h3 className="text-lg font-bold text-slate-900">Create New Deadline</h3>
            <p className="mt-1 text-xs text-slate-500">Add a course assignment deadline. Chatbot will reference this immediately.</p>
            
            <form onSubmit={(e) => {
              e.preventDefault();
              const formData = new FormData(e.currentTarget);
              const title = formData.get("title") as string;
              const dateStr = formData.get("date") as string;
              const timeStr = formData.get("time") as string;
              const rule = formData.get("rule") as string;
              
              if (title && dateStr && timeStr) {
                void handleCreateEvent(title, dateStr, timeStr, rule);
                setShowModal(false);
              }
            }} className="mt-4 space-y-4">
              <div>
                <label className="block text-xs font-bold uppercase tracking-wider text-slate-500">Title</label>
                <input required name="title" type="text" placeholder="e.g. Lab 5: System Calls" className="mt-1.5 w-full rounded-xl border border-slate-300 px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500" />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-bold uppercase tracking-wider text-slate-500">Date</label>
                  <input required name="date" type="date" className="mt-1.5 w-full rounded-xl border border-slate-300 px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500" />
                </div>
                <div>
                  <label className="block text-xs font-bold uppercase tracking-wider text-slate-500">Time</label>
                  <input required name="time" type="time" defaultValue="23:59" className="mt-1.5 w-full rounded-xl border border-slate-300 px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500" />
                </div>
              </div>
              <div>
                <label className="block text-xs font-bold uppercase tracking-wider text-slate-500">Late Penalty Policy</label>
                <input name="rule" type="text" placeholder="e.g. 10% penalty per day late" className="mt-1.5 w-full rounded-xl border border-slate-300 px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500" />
              </div>
              
              <div className="mt-6 flex items-center justify-end gap-2 pt-2 border-t border-slate-100">
                <button type="button" onClick={() => setShowModal(false)} className="rounded-lg px-4 py-2 text-sm font-semibold text-slate-600 hover:bg-slate-50 cursor-pointer">Cancel</button>
                <button type="submit" className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-semibold text-white hover:bg-blue-700 active:scale-95 transition-all cursor-pointer">Create</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </WorkspaceLayout>
  );
}
