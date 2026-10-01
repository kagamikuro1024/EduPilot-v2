package team._8.aitrogiang.controller;

import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.MediaType;
import org.springframework.http.codec.ServerSentEvent;
import org.springframework.security.core.Authentication;
import org.springframework.web.bind.annotation.*;
import reactor.core.publisher.Flux;
import team._8.aitrogiang.dto.AskAiRequest;
import team._8.aitrogiang.model.User;
import team._8.aitrogiang.service.ChatChannel;
import team._8.aitrogiang.service.ChatStreamingService;
import team._8.aitrogiang.service.PublicPrivacyFirewallService;
import team._8.aitrogiang.service.RateLimitType;
import team._8.aitrogiang.service.RateLimiterService;

import java.util.UUID;

/**
 * ┌──────────────────────────────────────────────────────────────────────┐
 * │  ARCHITECTURAL BOUNDARY ENFORCEMENT — SSE STREAMING GATEWAY          │
 * │                                                                      │
 * │  This controller is the SINGLE authorized entry point for all        │
 * │  AI interactions. The flow is strictly:                              │
 * │                                                                      │
 * │  [Frontend] ──POST──▶ [This Controller] ──gRPC──▶ [Python AI]       │
 * │                              │                                       │
 * │                    ┌─────────┴─────────┐                             │
 * │                    │  1. Auth (JWT ✅)  │                             │
 * │                    │  2. Save to DB ✅  │  ← Java's exclusive duty   │
 * │                    │  3. Stream SSE ✅  │                             │
 * │                    │  4. Save AI ans ✅ │  ← Java's exclusive duty   │
 * │                    └───────────────────┘                             │
 * │                                                                      │
 * │  JWT validation is handled automatically by JwtAuthFilter.           │
 * │  No explicit auth code is needed here.                               │
 * └──────────────────────────────────────────────────────────────────────┘
 */
@Slf4j
@RestController
@RequestMapping("/api/v1/threads")
@RequiredArgsConstructor
public class ChatStreamingController {

    private final ChatStreamingService chatStreamingService;
    private final RateLimiterService rateLimiterService;
    private final PublicPrivacyFirewallService privacyFirewallService;

    /**
     * SSE Streaming Endpoint — the canonical endpoint per API_CONTRACT.md
     *
     * Endpoint:  POST /api/v1/threads/{threadId}/ask-ai
     * Auth:      Bearer JWT (enforced by JwtAuthFilter before this method is called)
     * Produces:  text/event-stream (SSE)
     *
     * SSE Wire Format per chunk:
     *   data: {"chunk": "text token"}\n\n
     *
     * SSE Wire Format on completion:
     *   data: {"chunk": "", "is_finished": true, "metadata": {"agent_used": "...", "citations": [...]}}\n\n
     *
     * SSE Wire Format on error:
     *   event: error
     *   data: {"error": "human-readable message"}\n\n
     *
     * Rate Limiting is enforced at gateway/infrastructure layer.
     * If limit exceeded, Java returns HTTP 429 before this method is invoked.
     */
    @PostMapping(
            value = "/{threadId}/ask-ai",
            produces = MediaType.TEXT_EVENT_STREAM_VALUE
    )
    public Flux<ServerSentEvent<Object>> askAi(
            @PathVariable UUID threadId,
            @RequestBody(required = false) AskAiRequest body,
            Authentication authentication
    ) {
        User currentUser = (User) authentication.getPrincipal();
        rateLimiterService.checkOrThrow(currentUser.getId(), RateLimitType.ASK_AI);
        String message = (body != null && body.getMessage() != null)
                ? body.getMessage().trim()
                : "";
        boolean persistUserMessage = body == null || !body.isAutoTriggered();
        privacyFirewallService.enforcePublicSafeOrThrow(message);

        log.info("[Gateway] SSE request: user={}, thread={}, msg_len={}",
                currentUser.getEmail(), threadId, message.length());

        return chatStreamingService.processAndStream(threadId, message, currentUser, ChatChannel.FORUM, persistUserMessage);
    }
}
