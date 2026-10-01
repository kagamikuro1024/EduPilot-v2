"use client";

import { useRouter } from "next/navigation";
import { WorkspaceLayout } from "@/components/layout/WorkspaceLayout";
import { ThreadCreateModal } from "@/components/threads/ThreadCreateModal";
import { ThreadsStubView } from "@/components/threads/ThreadsStubView";
import { useThreadsList } from "@/hooks/useThreadsList";
import { useAuthStore } from "@/store/authStore";

export default function ThreadsPage() {
  const router = useRouter();
  const role = useAuthStore((s) => s.role);
  const staff = role === "TA" || role === "ADMIN";

  const {
    threads,
    total,
    loading,
    error,
    reload,
    searchInput,
    setSearchInput,
    filter,
    setFilter,
    selectedThreadId,
    setSelectedThreadId,
    createOpen,
    setCreateOpen,
    createSubmitting,
    createError,
    createPrivacyHint,
    analyzeCreateDraft,
    submitCreate,
  } = useThreadsList();

  const handleSelectThread = (id: string) => {
    setSelectedThreadId(id);
    router.push(`/threads/${id}`);
  };

  const handleCreateThread = async (body: Parameters<typeof submitCreate>[0]) => {
    const id = await submitCreate(body);
    if (!id) return;
    if (typeof window !== "undefined") {
      window.sessionStorage.setItem(`thread:${id}:autoAiQuestion`, body.content);
    }
    router.push(`/threads/${id}?auto_ai=1`);
  };

  return (
    <WorkspaceLayout footerLine2="rights" {...(staff ? { avatarInitialsOverride: "AR" } : {})}>
      <ThreadsStubView
        threads={threads}
        total={total}
        loading={loading}
        error={error}
        onRetry={reload}
        searchQuery={searchInput}
        onSearchChange={setSearchInput}
        activeFilter={filter}
        onFilterChange={setFilter}
        selectedThreadId={selectedThreadId}
        onSelectThread={handleSelectThread}
        onOpenCreate={() => setCreateOpen(true)}
      />
      <ThreadCreateModal
        open={createOpen}
        onOpenChange={setCreateOpen}
        submitting={createSubmitting}
        error={createError}
        privacyHint={createPrivacyHint}
        onDraftChange={analyzeCreateDraft}
        onSubmit={handleCreateThread}
      />
    </WorkspaceLayout>
  );
}
