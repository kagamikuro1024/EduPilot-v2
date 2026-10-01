package team._8.aitrogiang.controller;

import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import org.springframework.security.core.Authentication;
import org.springframework.web.bind.annotation.*;
import org.springframework.http.ResponseEntity;
import org.springframework.http.HttpStatus;
import team._8.aitrogiang.dto.CreateThreadRequest;
import team._8.aitrogiang.dto.SendThreadMessageRequest;
import team._8.aitrogiang.dto.PostDTO;
import team._8.aitrogiang.dto.ThreadListResponse;
import team._8.aitrogiang.model.User;
import team._8.aitrogiang.service.ForumService;

import java.util.List;
import java.util.Map;
import java.util.UUID;

/**
 * Handles forum CRUD operations (thread list, creation, message history).
 * The AI streaming endpoint (ask-ai) has been moved to ChatStreamingController
 * to enforce the Java SSE Gateway architectural boundary.
 */
@RestController
@RequestMapping("/api/v1/threads")
@RequiredArgsConstructor
public class ThreadController {

    private final ForumService forumService;

    @GetMapping
    public ResponseEntity<ThreadListResponse> getThreads(
            @RequestParam(required = false) String status,
            @RequestParam(required = false) String tag,
            @RequestParam(required = false) String search,
            @RequestParam(defaultValue = "1") int page,
            Authentication authentication) {
        User currentUser = authentication != null && authentication.getPrincipal() instanceof User
                ? (User) authentication.getPrincipal()
                : null;
        return ResponseEntity.ok(forumService.getThreads(status, tag, search, page, currentUser));
    }

    @PostMapping
    public ResponseEntity<Map<String, String>> createThread(
            @Valid @RequestBody CreateThreadRequest body,
            Authentication authentication
    ) {
        User currentUser = (User) authentication.getPrincipal();
        String threadId = forumService.createThread(body, currentUser);
        return ResponseEntity.status(HttpStatus.CREATED).body(Map.of("thread_id", threadId));
    }

    @GetMapping("/{thread_id}/messages")
    public ResponseEntity<List<PostDTO>> getMessages(
            @PathVariable("thread_id") UUID threadId,
            Authentication authentication
    ) {
        User currentUser = (User) authentication.getPrincipal();
        return ResponseEntity.ok(forumService.getMessages(threadId, currentUser));
    }

    @PostMapping("/{thread_id}/messages")
    public ResponseEntity<Map<String, String>> sendMessage(
            @PathVariable("thread_id") UUID threadId,
            @Valid @RequestBody SendThreadMessageRequest body,
            Authentication authentication
    ) {
        User currentUser = (User) authentication.getPrincipal();
        String id = forumService.createHumanReply(threadId, body.getContent().trim(), currentUser);
        return ResponseEntity.ok(Map.of("id", id));
    }

    // NOTE: POST /{threadId}/ask-ai is intentionally removed from this controller.
    // It is now handled by ChatStreamingController, which enforces the full
    // Java SSE Gateway contract (JWT → DB save → gRPC → stream → DB save).
}
