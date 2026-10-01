package team._8.aitrogiang.dto;

import lombok.Builder;
import lombok.Value;

@Value
@Builder
public class AnalyticsMetricsDTO {
    Double ta_verification_accuracy;
    Double correction_rate;
    Double rejection_rate;
    Double helpful_feedback_rate;
    Integer needs_ta_count;
    Integer pending_verification_count;
    Integer verified_count;
    Integer corrected_count;
    Integer rejected_count;
    Integer sensitive_detections;
    Integer public_leak_prevention_count;
    Double channel_switch_rate;
}
