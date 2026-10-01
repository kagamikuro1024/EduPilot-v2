package team._8.aitrogiang.controller;

import lombok.RequiredArgsConstructor;
import org.springframework.http.MediaType;
import org.springframework.http.codec.ServerSentEvent;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.Authentication;
import org.springframework.web.bind.annotation.*;
import reactor.core.publisher.Flux;
import team._8.aitrogiang.dto.AskAiRequest;
import team._8.aitrogiang.dto.ChatMessageDTO;
import team._8.aitrogiang.dto.CreateChatSessionResponse;
import team._8.aitrogiang.model.ChatSession;
import team._8.aitrogiang.model.User;
import team._8.aitrogiang.service.ChatChannel;
import team._8.aitrogiang.service.ChatStreamingService;
import team._8.aitrogiang.service.PrivateChatService;
import team._8.aitrogiang.service.RateLimitType;
import team._8.aitrogiang.service.RateLimiterService;
import lombok.extern.slf4j.Slf4j;
import java.util.List;
import java.util.Map;
import java.util.UUID;

@RestController
@RequestMapping("/api/v1/chat/sessions")
@RequiredArgsConstructor
@Slf4j
public class ChatController {

    private final PrivateChatService privateChatService;
    private final ChatStreamingService chatStreamingService;
    private final RateLimiterService rateLimiterService;

    @GetMapping
    public ResponseEntity<List<Map<String, String>>> listSessions(Authentication authentication) {
        if (authentication == null || !(authentication.getPrincipal() instanceof User)) {
            return ResponseEntity.status(401).build();
        }
        User currentUser = (User) authentication.getPrincipal();
        List<Map<String, String>> sessions = privateChatService.listSessions(currentUser).stream()
                .map(session -> Map.of("session_id", session.getId().toString()))
                .toList();
        return ResponseEntity.ok(sessions);
    }

    @PostMapping
    public ResponseEntity<CreateChatSessionResponse> createSession(Authentication authentication) {
        if (authentication == null || !(authentication.getPrincipal() instanceof User)) {
            return ResponseEntity.status(401).build();
        }
        User currentUser = (User) authentication.getPrincipal();
        ChatSession session = privateChatService.createSession(currentUser);
        return ResponseEntity.ok(CreateChatSessionResponse.builder()
                .session_id(session.getId().toString())
                .build());
    }

    @GetMapping("/{id}/messages")
    public ResponseEntity<List<ChatMessageDTO>> getMessages(@PathVariable UUID id, Authentication authentication) {
        if (authentication == null || !(authentication.getPrincipal() instanceof User)) {
            return ResponseEntity.status(401).build();
        }
        User currentUser = (User) authentication.getPrincipal();
        return ResponseEntity.ok(privateChatService.getMessages(id, currentUser));
    }

    @PostMapping(value = "/{id}/ask-ai", produces = MediaType.TEXT_EVENT_STREAM_VALUE)
    public Flux<ServerSentEvent<Object>> askAi(
            @PathVariable UUID id,
            @RequestBody(required = false) AskAiRequest body,
            Authentication authentication
    ) {
        if (authentication == null || !(authentication.getPrincipal() instanceof User)) {
            return Flux.just(ServerSentEvent.builder().event("error").data("Unauthorized").build());
        }
        User currentUser = (User) authentication.getPrincipal();

        rateLimiterService.checkOrThrow(currentUser.getId(), RateLimitType.ASK_AI);
        privateChatService.validateOwnership(id, currentUser);

        String message = (body != null && body.getMessage() != null) ? body.getMessage().trim() : "";
        return chatStreamingService.processAndStream(id, message, currentUser, ChatChannel.CHAT);
    }

    @DeleteMapping("/{id}")
    public ResponseEntity<Void> deleteSession(@PathVariable UUID id, Authentication authentication) {
        if (authentication == null || !(authentication.getPrincipal() instanceof User)) {
            return ResponseEntity.status(401).build();
        }
        User currentUser = (User) authentication.getPrincipal();
        privateChatService.deleteSession(id, currentUser);
        return ResponseEntity.ok().build();
    }
}
