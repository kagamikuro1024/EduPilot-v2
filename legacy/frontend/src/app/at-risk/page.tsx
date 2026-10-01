"use client";

import { useAnalytics } from "@/hooks/useAnalytics";
import { useAuthStore } from "@/store/authStore";
import { WorkspaceLayout } from "@/components/layout/WorkspaceLayout";
import { 
  AlertTriangle, 
  ChevronRight, 
  MessageSquareWarning, 
  Search, 
  UserPlus, 
  Mail, 
  ExternalLink 
} from "lucide-react";
import { EngagementBand } from "@/hooks/useAnalytics";

function engagementPill(band: EngagementBand): { label: string; className: string } {
  switch (band) {
    case "critical":
      return { label: "Critical", className: "bg-red-100 text-red-700 ring-red-200" };
    case "warning":
      return { label: "Warning", className: "bg-amber-100 text-amber-800 ring-amber-200" };
    case "at_risk":
      return {
        label: "At Risk",
        className: "bg-slate-100 text-slate-600 ring-slate-200",
      };
  }
}

export default function AtRiskPage() {
  const role = useAuthStore((s) => s.role);
  const isStaff = role === "TA" || role === "ADMIN";
  const { atRiskStudents, loading, error } = useAnalytics("7d");

  if (!isStaff) {
    return (
      <WorkspaceLayout footerLine2="operational">
        <div className="mx-auto max-w-3xl px-6 py-10">
          <div className="rounded-xl border border-amber-200 bg-amber-50 p-6">
            <h1 className="text-xl font-semibold text-amber-900">Access restricted</h1>
            <p className="mt-2 text-sm text-amber-800">
              This dashboard is only for TA/ADMIN accounts.
            </p>
          </div>
        </div>
      </WorkspaceLayout>
    );
  }

  return (
    <WorkspaceLayout footerLine2="operational">
      <div className="mx-auto max-w-6xl space-y-6 px-6 py-8">
        {/* Page Header */}
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-slate-900">
              At-Risk Students
            </h1>
            <p className="mt-1 text-sm text-slate-600">
              Identify and support students who may be falling behind.
            </p>
          </div>
          <div className="flex items-center gap-2">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
              <input 
                type="text" 
                placeholder="Search students..." 
                className="rounded-lg border border-slate-200 bg-white py-2 pl-10 pr-4 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500/20"
              />
            </div>
            <button className="inline-flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-blue-700 transition">
              <UserPlus className="h-4 w-4" />
              Add Student
            </button>
          </div>
        </div>

        {error && (
          <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
            {error}
          </div>
        )}

        {/* Stats Summary */}
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
            <p className="text-xs font-bold uppercase tracking-wider text-slate-500">Total Flagged</p>
            <p className="mt-2 text-3xl font-bold text-slate-900">{atRiskStudents.length}</p>
          </div>
          <div className="rounded-xl border border-red-50 bg-red-50/30 p-5 shadow-sm ring-1 ring-red-100">
            <p className="text-xs font-bold uppercase tracking-wider text-red-600">Critical Cases</p>
            <p className="mt-2 text-3xl font-bold text-red-700">
              {atRiskStudents.filter(s => s.engagement === "critical").length}
            </p>
          </div>
          <div className="rounded-xl border border-amber-50 bg-amber-50/30 p-5 shadow-sm ring-1 ring-amber-100">
            <p className="text-xs font-bold uppercase tracking-wider text-amber-600">Warnings</p>
            <p className="mt-2 text-3xl font-bold text-amber-700">
              {atRiskStudents.filter(s => s.engagement === "warning").length}
            </p>
          </div>
        </div>

        {/* Students Table */}
        <div className="rounded-xl border border-slate-200 bg-white shadow-sm overflow-hidden">
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-slate-200 text-left text-sm">
              <thead className="bg-slate-50 text-xs font-semibold uppercase tracking-wide text-slate-500">
                <tr>
                  <th className="px-6 py-4">Student Name</th>
                  <th className="px-6 py-4">Risk Reason</th>
                  <th className="px-6 py-4">Status</th>
                  <th className="px-6 py-4 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 bg-white">
                {loading ? (
                  Array.from({ length: 3 }).map((_, i) => (
                    <tr key={i}>
                      <td className="px-6 py-6" colSpan={4}>
                        <div className="h-10 animate-pulse rounded-lg bg-slate-100" />
                      </td>
                    </tr>
                  ))
                ) : atRiskStudents.length === 0 ? (
                  <tr>
                    <td className="px-6 py-12 text-center text-slate-500" colSpan={4}>
                      No at-risk students found. Great job!
                    </td>
                  </tr>
                ) : (
                  atRiskStudents.map((student) => {
                    const pill = engagementPill(student.engagement);
                    return (
                      <tr key={student.id} className="group hover:bg-slate-50/50 transition">
                        <td className="px-6 py-4">
                          <div className="flex items-center gap-3">
                            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-blue-500 to-indigo-600 text-xs font-bold text-white shadow-inner">
                              {student.avatarInitials}
                            </div>
                            <div>
                              <p className="font-bold text-slate-900">{student.displayName}</p>
                              <p className="text-xs text-slate-500">{student.studentId}</p>
                            </div>
                          </div>
                        </td>
                        <td className="px-6 py-4">
                          <div className="flex items-start gap-2">
                            <MessageSquareWarning className="mt-0.5 h-4 w-4 shrink-0 text-amber-500" />
                            <span className="text-slate-600 leading-snug">{student.lastActiveLabel}</span>
                          </div>
                        </td>
                        <td className="px-6 py-4">
                          <span className={`inline-flex items-center rounded-full px-2.5 py-1 text-xs font-bold ring-1 ${pill.className}`}>
                            {pill.label}
                          </span>
                        </td>
                        <td className="px-6 py-4 text-right">
                          <div className="flex items-center justify-end gap-2">
                            <button 
                              className="rounded-lg p-2 text-slate-400 hover:bg-blue-50 hover:text-blue-600 transition"
                              title="Email Student"
                            >
                              <Mail className="h-4 w-4" />
                            </button>
                            <button 
                              className="rounded-lg p-2 text-slate-400 hover:bg-blue-50 hover:text-blue-600 transition"
                              title="View Threads"
                            >
                              <ExternalLink className="h-4 w-4" />
                            </button>
                            <button className="rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-xs font-bold text-slate-700 hover:bg-slate-50 transition">
                              Details
                            </button>
                          </div>
                        </td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>
        </div>

        {/* Action Suggestion */}
        <div className="rounded-2xl bg-gradient-to-r from-blue-600 to-indigo-700 p-8 text-white shadow-lg relative overflow-hidden">
          <div className="absolute top-0 right-0 p-4 opacity-10">
             <AlertTriangle size={120} />
          </div>
          <div className="relative z-10 max-w-2xl">
            <h2 className="text-xl font-bold">Proactive Support Needed</h2>
            <p className="mt-2 text-blue-100">
              Students flagged as "Critical" have not participated in the last 3 assignments. 
              We recommend scheduling a 1-on-1 session or sending a supportive nudge via email.
            </p>
            <div className="mt-6 flex flex-wrap gap-3">
              <button className="rounded-xl bg-white px-6 py-3 text-sm font-bold text-blue-700 shadow-sm hover:bg-blue-50 transition">
                Send Bulk Notification
              </button>
              <button className="rounded-xl bg-blue-500/30 px-6 py-3 text-sm font-bold text-white border border-white/20 hover:bg-blue-500/40 transition">
                Export Data for Lecturer
              </button>
            </div>
          </div>
        </div>
      </div>
    </WorkspaceLayout>
  );
}
