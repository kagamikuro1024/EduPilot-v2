package team._8.aitrogiang.service;

import org.springframework.stereotype.Service;
import team._8.aitrogiang.exception.IntentViolationException;
import team._8.aitrogiang.grpc.ClassifyResponse;
import team._8.aitrogiang.grpc.IntentType;

import java.util.regex.Pattern;

@Service
public class PublicPrivacyFirewallService {

    private static final Pattern SENSITIVE_PATTERN = Pattern.compile(
            "\\b(\\d{8,12}|student\\s*id|mssv|grade|gpa|password|phone|email|cccd|social\\s*security)\\b",
            Pattern.CASE_INSENSITIVE
    );

    private final IntentClassifierClient classifier;
    private final PrivacyAnalyticsService privacyAnalyticsService;

    public PublicPrivacyFirewallService(
            IntentClassifierClient classifier,
            PrivacyAnalyticsService privacyAnalyticsService
    ) {
        this.classifier = classifier;
        this.privacyAnalyticsService = privacyAnalyticsService;
    }

    /**
     * Enforce privacy policy for public/forum channel:
     * - block if regex signals sensitive information
     * - block if classifier routes to PRIVATE
     * - block if classifier marks explicit violation
     */
    public void enforcePublicSafeOrThrow(String content) {
        String safe = content == null ? "" : content.trim();
        if (safe.isBlank()) {
            return;
        }

        boolean localSensitive = SENSITIVE_PATTERN.matcher(safe).find();
        ClassifyResponse classification = classifier.classify(safe, "PUBLIC");

        boolean classifierPrivate = classification.getSuggestedChannel() == IntentType.PRIVATE;
        boolean classifierViolation = classification.getIsViolation();
        if (localSensitive || classifierPrivate || classifierViolation) {
            String reasonCode = localSensitive ? "LOCAL_SENSITIVE_PATTERN" : "CLASSIFIER_PRIVATE_OR_VIOLATION";
            privacyAnalyticsService.trackSensitiveDetection("PUBLIC", reasonCode, classification.getConfidence());
            privacyAnalyticsService.trackPublicLeakPrevention(reasonCode, classification.getConfidence());
            String reason = classification.getReasoning();
            if (reason == null || reason.isBlank()) {
                reason = "Public channel blocked: sensitive/private content detected.";
            }
            throw new IntentViolationException(reason, classification.getConfidence());
        }
    }
}
