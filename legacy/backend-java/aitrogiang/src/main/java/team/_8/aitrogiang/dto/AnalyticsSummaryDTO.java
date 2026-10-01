package team._8.aitrogiang.dto;

import lombok.Builder;
import lombok.Value;

import java.util.List;

@Value
@Builder
public class AnalyticsSummaryDTO {
    long total_questions_this_week;
    double ai_resolution_rate;
    List<TopTagDTO> top_tags;
    List<AtRiskStudentDTO> at_risk_students;

    @Value
    @Builder
    public static class TopTagDTO {
        String name;
        int query_count;
        float difficulty_score;
    }

    @Value
    @Builder
    public static class AtRiskStudentDTO {
        String student_id;
        String risk_level;
        String reason;
    }
}
