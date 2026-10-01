package team._8.aitrogiang.controller;

import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.Authentication;
import org.springframework.web.bind.annotation.*;
import team._8.aitrogiang.dto.ClassifyIntentRequestDTO;
import team._8.aitrogiang.dto.ClassifyIntentResponseDTO;
import team._8.aitrogiang.grpc.ClassifyResponse;
import team._8.aitrogiang.model.User;
import team._8.aitrogiang.service.IntentClassifierClient;
import team._8.aitrogiang.service.PrivacyAnalyticsService;
import team._8.aitrogiang.service.RateLimitType;
import team._8.aitrogiang.service.RateLimiterService;

@RestController
@RequestMapping("/api/v1/classify-intent")
@RequiredArgsConstructor
public class IntentController {

    private final IntentClassifierClient classifier;
    private final RateLimiterService rateLimiterService;
    private final PrivacyAnalyticsService privacyAnalyticsService;

    @PostMapping
    public ResponseEntity<ClassifyIntentResponseDTO> classify(
            @RequestBody ClassifyIntentRequestDTO body,
            Authentication authentication
    ) {
        User currentUser = (User) authentication.getPrincipal();
        rateLimiterService.checkOrThrow(currentUser.getId(), RateLimitType.CLASSIFY);
        ClassifyResponse resp = classifier.classify(body.getContent(), body.getChannelHint());
        String hint = body.getChannelHint() == null ? "UNKNOWN" : body.getChannelHint().toUpperCase();
        String suggested = resp.getSuggestedChannel().name();
        if (!"UNCERTAIN".equals(suggested) && !suggested.equals(hint)) {
            privacyAnalyticsService.trackChannelSwitchSuggested(hint, suggested, resp.getConfidence());
        }
        if ("PRIVATE".equals(suggested) || resp.getIsViolation()) {
            privacyAnalyticsService.trackSensitiveDetection(hint, "CLASSIFY_PRIVATE_OR_VIOLATION", resp.getConfidence());
        }
        return ResponseEntity.ok(ClassifyIntentResponseDTO.builder()
                .suggestedChannel(resp.getSuggestedChannel().name())
                .confidence(resp.getConfidence())
                .reasoning(resp.getReasoning())
                .build());
    }
}
