"use client";

import { useEffect, useState, type FormEvent } from "react";
import type { CreateThreadRequestBody } from "@/types/threads";

export interface ThreadCreateModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  submitting: boolean;
  error: string | null;
  privacyHint: string | null;
  onDraftChange: (title: string, content: string) => Promise<void> | void;
  onSubmit: (body: CreateThreadRequestBody) => Promise<void> | Promise<string | null>;
}

function parseTags(raw: string): string[] | undefined {
  const parts = raw
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);
  return parts.length ? parts : undefined;
}

export function ThreadCreateModal({
  open,
  onOpenChange,
  submitting,
  error,
  privacyHint,
  onDraftChange,
  onSubmit,
}: ThreadCreateModalProps) {
  const [title, setTitle] = useState("");
  const [content, setContent] = useState("");
  const [tagsRaw, setTagsRaw] = useState("");

  useEffect(() => {
    if (!open) {
      setTitle("");
      setContent("");
      setTagsRaw("");
    }
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const timer = window.setTimeout(() => {
      void onDraftChange(title, content);
    }, 800);
    return () => window.clearTimeout(timer);
  }, [open, title, content, onDraftChange]);

  if (!open) return null;

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    await onSubmit({
      title: title.trim(),
      content: content.trim(),
      tags: parseTags(tagsRaw),
    });
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <button
        type="button"
        className="absolute inset-0 bg-slate-900/40"
        aria-label="Close dialog"
        onClick={() => !submitting && onOpenChange(false)}
      />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="thread-create-title"
        className="relative z-10 w-full max-w-md rounded-xl border border-slate-200 bg-white p-5 shadow-xl"
      >
        <h2 id="thread-create-title" className="text-lg font-bold text-slate-900">
          New thread
        </h2>
        <p className="mt-1 text-sm text-slate-600">Creates a public forum thread (same contract as the Java API).</p>
        <form onSubmit={handleSubmit} className="mt-4 space-y-3">
          <div>
            <label htmlFor="thread-title" className="text-xs font-semibold uppercase tracking-wide text-slate-500">
              Title
            </label>
            <input
              id="thread-title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              required
              maxLength={500}
              className="mt-1 w-full rounded-lg border border-slate-200 px-3 py-2 text-sm outline-none ring-blue-100 focus:border-blue-400 focus:ring-4"
              placeholder="Short question title"
              disabled={submitting}
            />
          </div>
          <div>
            <label htmlFor="thread-content" className="text-xs font-semibold uppercase tracking-wide text-slate-500">
              First message
            </label>
            <textarea
              id="thread-content"
              value={content}
              onChange={(e) => setContent(e.target.value)}
              required
              rows={5}
              className="mt-1 w-full resize-y rounded-lg border border-slate-200 px-3 py-2 text-sm outline-none ring-blue-100 focus:border-blue-400 focus:ring-4"
              placeholder="Describe your question or topic…"
              disabled={submitting}
            />
          </div>
          <div>
            <label htmlFor="thread-tags" className="text-xs font-semibold uppercase tracking-wide text-slate-500">
              Tags (optional)
            </label>
            <input
              id="thread-tags"
              value={tagsRaw}
              onChange={(e) => setTagsRaw(e.target.value)}
              className="mt-1 w-full rounded-lg border border-slate-200 px-3 py-2 text-sm outline-none ring-blue-100 focus:border-blue-400 focus:ring-4"
              placeholder="lecture, midterm (comma-separated)"
              disabled={submitting}
            />
          </div>
          {error ? (
            <p className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-800 ring-1 ring-red-100" role="alert">
              {error}
            </p>
          ) : null}
          {privacyHint ? (
            <p className="rounded-lg bg-amber-50 px-3 py-2 text-sm text-amber-800 ring-1 ring-amber-100" role="status">
              Privacy suggestion: {privacyHint}
            </p>
          ) : null}
          <div className="flex justify-end gap-2 pt-1">
            <button
              type="button"
              className="rounded-lg border border-slate-200 px-3 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50 disabled:opacity-50"
              onClick={() => onOpenChange(false)}
              disabled={submitting}
            >
              Cancel
            </button>
            <button
              type="submit"
              className="rounded-lg bg-blue-600 px-3 py-2 text-sm font-semibold text-white hover:bg-blue-700 disabled:opacity-60"
              disabled={submitting}
            >
              {submitting ? "Creating…" : "Create"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
