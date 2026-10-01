package team._8.aitrogiang.controller;

import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.Authentication;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;
import team._8.aitrogiang.dto.SubmitFeedbackRequest;
import team._8.aitrogiang.model.User;
import team._8.aitrogiang.service.PrivateChatService;

import java.util.Map;
import java.util.UUID;

@RestController
@RequestMapping("/api/v1/chat/messages")
@RequiredArgsConstructor
public class ChatFeedbackController {

    private final PrivateChatService privateChatService;

    @PutMapping("/{message_id}/feedback")
    public ResponseEntity<Map<String, Boolean>> submitFeedback(
            @PathVariable("message_id") UUID messageId,
            @Valid @RequestBody SubmitFeedbackRequest body,
            Authentication authentication
    ) {
        User currentUser = (User) authentication.getPrincipal();
        privateChatService.submitFeedback(messageId, body.getFeedback(), currentUser);
        return ResponseEntity.ok(Map.of("success", true));
    }
}
