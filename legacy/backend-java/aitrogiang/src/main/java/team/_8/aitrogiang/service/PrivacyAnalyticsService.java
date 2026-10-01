package team._8.aitrogiang.service;

import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;
import team._8.aitrogiang.model.AnalyticsPrivacyEvent;
import team._8.aitrogiang.repository.AnalyticsPrivacyEventRepository;

@Service
@RequiredArgsConstructor
public class PrivacyAnalyticsService {

    private final AnalyticsPrivacyEventRepository repository;

    public void trackSensitiveDetection(String channel, String reasonCode, Float confidence) {
        save("SENSITIVE_DETECTION", channel, reasonCode, confidence);
    }

    public void trackPublicLeakPrevention(String reasonCode, Float confidence) {
        save("PUBLIC_LEAK_PREVENTION", "PUBLIC", reasonCode, confidence);
    }

    public void trackChannelSwitchSuggested(String fromChannel, String toChannel, Float confidence) {
        save("CHANNEL_SWITCH_SUGGESTED", fromChannel, toChannel, confidence);
    }

    private void save(String eventType, String channel, String reasonCode, Float confidence) {
        repository.save(AnalyticsPrivacyEvent.builder()
                .eventType(eventType)
                .channel(channel == null || channel.isBlank() ? "UNKNOWN" : channel.toUpperCase())
                .reasonCode(reasonCode == null || reasonCode.isBlank() ? null : reasonCode)
                .confidence(confidence)
                .build());
    }
}
