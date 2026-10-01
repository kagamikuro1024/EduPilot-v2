import { afterAll, beforeEach, describe, expect, it, vi } from "vitest";
import { streamChat } from "@/services/aiClient";
import { useAuthStore } from "@/store/authStore";
import { toast } from "sonner";

vi.mock("sonner", () => ({
  toast: {
    warning: vi.fn(),
    error: vi.fn(),
  },
}));

const originalFetch = global.fetch;

describe("aiClient streamChat", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    useAuthStore.setState({ token: "fake-token", role: "STUDENT" });
    process.env.NEXT_PUBLIC_JAVA_API_URL = "http://localhost:8080";
  });

  it("does not call onError when aborted by user", async () => {
    const controller = new AbortController();
    const onError = vi.fn();
    const onChunk = vi.fn();
    const onFinish = vi.fn();

    global.fetch = vi.fn(async () => {
      throw new DOMException("Aborted", "AbortError");
    }) as unknown as typeof global.fetch;

    controller.abort();
    await streamChat("thread-1", "hello", { onChunk, onError, onFinish }, controller.signal);

    expect(onError).not.toHaveBeenCalled();
  });

  it("returns rate-limit error message for 429", async () => {
    const onError = vi.fn();
    const onRateLimit = vi.fn();
    const onChunk = vi.fn();
    const onFinish = vi.fn();

    global.fetch = vi.fn(async () =>
      new Response(null, { status: 429, statusText: "Too Many Requests" })
    ) as unknown as typeof global.fetch;

    await streamChat("thread-1", "hello", { onChunk, onError, onFinish, onRateLimit });
    expect(onRateLimit).toHaveBeenCalledWith(
      "Bạn đã hết lượt hỏi AI hôm nay. Vui lòng đợi TA hỗ trợ hoặc thử lại vào ngày mai."
    );
    expect(onError).not.toHaveBeenCalled();
  });

  it("shows privacy toast for 403 private-channel classifier errors without calling onError", async () => {
    const onError = vi.fn();
    const onSecurityBlock = vi.fn();
    const onChunk = vi.fn();

    global.fetch = vi.fn(async () =>
      new Response(
        JSON.stringify({
          error_code: "ERR_PII_DETECTED",
          message: "Switch to private",
          suggested_channel: "PRIVATE",
        }),
        { status: 403, headers: { "Content-Type": "application/json" } },
      )
    ) as unknown as typeof global.fetch;

    await streamChat("thread-1", "hello", { onChunk, onError, onSecurityBlock });

    expect(toast.warning).toHaveBeenCalledWith(
      "Câu hỏi chứa thông tin cá nhân. Vui lòng chuyển sang kênh Chat 1v1."
    );
    expect(onSecurityBlock).toHaveBeenCalled();
    expect(onError).not.toHaveBeenCalled();
  });

  it("shows red policy toast for malicious injection errors", async () => {
    const onError = vi.fn();
    const onSecurityBlock = vi.fn();
    const onChunk = vi.fn();

    global.fetch = vi.fn(async () =>
      new Response(
        JSON.stringify({
          error_code: "MALICIOUS_INJECTION",
          message: "Policy violation",
          suggested_channel: null,
        }),
        { status: 403, headers: { "Content-Type": "application/json" } },
      )
    ) as unknown as typeof global.fetch;

    await streamChat("thread-1", "hello", { onChunk, onError, onSecurityBlock });

    expect(toast.error).toHaveBeenCalledWith(
      "Yêu cầu bị từ chối do vi phạm chính sách an toàn của hệ thống.",
    );
    expect(onSecurityBlock).toHaveBeenCalled();
    expect(onError).not.toHaveBeenCalled();
  });

  it("shows policy rejection toast for INTENT_VIOLATION even when suggested_channel is PRIVATE", async () => {
    const onError = vi.fn();
    const onSecurityBlock = vi.fn();
    const onChunk = vi.fn();

    global.fetch = vi.fn(async () =>
      new Response(
        JSON.stringify({
          error_code: "INTENT_VIOLATION",
          message: "Some backend detail",
          suggested_channel: "PRIVATE",
        }),
        { status: 403, headers: { "Content-Type": "application/json" } },
      )
    ) as unknown as typeof global.fetch;

    await streamChat("thread-1", "hello", { onChunk, onError, onSecurityBlock });

    expect(toast.warning).not.toHaveBeenCalled();
    expect(toast.error).toHaveBeenCalledWith(
      "Yêu cầu bị từ chối do vi phạm chính sách an toàn của hệ thống.",
    );
    expect(onSecurityBlock).toHaveBeenCalled();
    expect(onError).not.toHaveBeenCalled();
  });

  it("returns service unavailable message for 503", async () => {
    const onError = vi.fn();
    const onChunk = vi.fn();
    const onFinish = vi.fn();

    global.fetch = vi.fn(async () =>
      new Response(null, { status: 503, statusText: "Service Unavailable" })
    ) as unknown as typeof global.fetch;

    await streamChat("thread-1", "hello", { onChunk, onError, onFinish });
    expect(onError).toHaveBeenCalledWith(
      "Trợ giảng AI đang tạm thời không hoạt động. Câu hỏi của bạn đã được ghi nhận, TA sẽ hỗ trợ sớm."
    );
  });

  it("calls onFinish when SSE stream is finished", async () => {
    const onError = vi.fn();
    const onChunk = vi.fn();
    const onFinish = vi.fn();
    const payload = 'data: {"chunk":"Hi"}\n\ndata: {"chunk":"","is_finished":true,"metadata":{"agent_used":"QA_AGENT"}}\n\n';

    global.fetch = vi.fn(async () => {
      const stream = new ReadableStream<Uint8Array>({
        start(controller) {
          controller.enqueue(new TextEncoder().encode(payload));
          controller.close();
        },
      });
      return new Response(stream, { status: 200 });
    }) as unknown as typeof global.fetch;

    await streamChat("thread-1", "hello", { onChunk, onError, onFinish });
    expect(onChunk).toHaveBeenCalledWith("Hi");
    expect(onFinish).toHaveBeenCalled();
    expect(onError).not.toHaveBeenCalled();
  });

  it("sends autoTriggered flag for contextual first answer", async () => {
    const onError = vi.fn();
    const onChunk = vi.fn();
    const onFinish = vi.fn();

    global.fetch = vi.fn(async () =>
      new Response(
        new ReadableStream<Uint8Array>({
          start(controller) {
            controller.enqueue(new TextEncoder().encode('data: {"chunk":"","is_finished":true}\n\n'));
            controller.close();
          },
        }),
        { status: 200 },
      )
    ) as unknown as typeof global.fetch;

    await streamChat("thread-1", "Explain recursion", { onChunk, onError, onFinish }, undefined, {
      autoTriggered: true,
    });

    const [, init] = vi.mocked(global.fetch).mock.calls[0];
    expect(JSON.parse(init.body)).toEqual({ message: "Explain recursion", autoTriggered: true });
  });
});

afterAll(() => {
  global.fetch = originalFetch;
});
