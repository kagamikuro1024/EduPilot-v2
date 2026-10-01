package team._8.aitrogiang.controller;

import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import org.springframework.security.core.Authentication;
import org.springframework.web.bind.annotation.*;
import org.springframework.http.ResponseEntity;
import team._8.aitrogiang.dto.CorrectMessageRequest;
import team._8.aitrogiang.model.User;
import team._8.aitrogiang.service.ForumService;

import java.util.Map;
import java.util.UUID;

@RestController
@RequestMapping("/api/v1/messages")
@RequiredArgsConstructor
public class MessageController {

    private final ForumService forumService;

    @PutMapping("/{message_id}/verify")
    public ResponseEntity<Map<String, Boolean>> verifyMessage(
            @PathVariable("message_id") UUID messageId,
            Authentication authentication
    ) {
        User currentUser = (User) authentication.getPrincipal();
        forumService.verifyMessage(messageId, currentUser);
        return ResponseEntity.ok(Map.of("success", true));
    }

    @PutMapping("/{message_id}/correct")
    public ResponseEntity<Map<String, Boolean>> correctMessage(
            @PathVariable("message_id") UUID messageId,
            @Valid @RequestBody CorrectMessageRequest body,
            Authentication authentication
    ) {
        User currentUser = (User) authentication.getPrincipal();
        forumService.correctMessage(messageId, body.getContent().trim(), currentUser);
        return ResponseEntity.ok(Map.of("success", true));
    }

    @PutMapping("/{message_id}/reject")
    public ResponseEntity<Map<String, Boolean>> rejectMessage(
            @PathVariable("message_id") UUID messageId,
            Authentication authentication
    ) {
        User currentUser = (User) authentication.getPrincipal();
        forumService.rejectMessage(messageId, currentUser);
        return ResponseEntity.ok(Map.of("success", true));
    }
}
