import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useThreadsList } from "@/hooks/useThreadsList";

const listThreadsMock = vi.fn();
const createThreadMock = vi.fn();
const classifyMock = vi.fn();

vi.mock("@/services/threads.service", () => ({
  threadsService: {
    listThreads: (...args: unknown[]) => listThreadsMock(...args),
    createThread: (...args: unknown[]) => createThreadMock(...args),
  },
}));

vi.mock("@/services/intent.service", () => ({
  intentService: {
    classify: (...args: unknown[]) => classifyMock(...args),
  },
}));

describe("useThreadsList privacy gate", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    listThreadsMock.mockResolvedValue({ items: [], total: 0 });
    createThreadMock.mockResolvedValue("new-thread-id");
    classifyMock.mockResolvedValue({ suggestedChannel: "PUBLIC", confidence: 0.9, reasoning: "" });
  });

  it("blocks public thread creation when local sensitive signals are detected", async () => {
    const { result } = renderHook(() => useThreadsList());
    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      await result.current.submitCreate({
        title: "Need help",
        content: "MSSV 22123456 diem giua ky cua em",
      });
    });

    expect(createThreadMock).not.toHaveBeenCalled();
    expect(result.current.createError).toContain("Khong the dang thread cong khai");
  });

  it("allows benign public thread creation", async () => {
    const { result } = renderHook(() => useThreadsList());
    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      await result.current.submitCreate({
        title: "Question about recursion",
        content: "How to debug recursive base case?",
      });
    });

    expect(createThreadMock).toHaveBeenCalledTimes(1);
    expect(result.current.createError).toBeNull();
  });
});
