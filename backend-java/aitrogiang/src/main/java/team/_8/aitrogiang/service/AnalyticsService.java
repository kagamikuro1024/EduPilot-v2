package team._8.aitrogiang.service;

import lombok.RequiredArgsConstructor;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Service;
import team._8.aitrogiang.dto.AnalyticsMetricsDTO;
import team._8.aitrogiang.dto.AtRiskStudentRecord;
import team._8.aitrogiang.dto.StudentActivityPointRecord;
import team._8.aitrogiang.dto.TopicDifficultyRecord;
import team._8.aitrogiang.repository.ChatMessageRepository;
import team._8.aitrogiang.repository.ForumPostRepository;
import team._8.aitrogiang.repository.AnalyticsTopicDifficultyRepository;
import team._8.aitrogiang.repository.AnalyticsAtRiskStudentRepository;
import team._8.aitrogiang.repository.AnalyticsDailySummaryRepository;
import team._8.aitrogiang.model.AnalyticsDailySummary;

import java.time.LocalDateTime;
import java.util.List;

@Service
@RequiredArgsConstructor
public class AnalyticsService {

    private final JdbcClient jdbcClient;
    private final ForumPostRepository forumPostRepository;
    private final ChatMessageRepository chatMessageRepository;
    private final AnalyticsTopicDifficultyRepository topicDifficultyRepository;
    private final AnalyticsAtRiskStudentRepository atRiskStudentRepository;
    private final AnalyticsDailySummaryRepository dailySummaryRepository;

    /**
     * Retrieves the top 5 most difficult topics based on difficulty score.
     */
    public List<TopicDifficultyRecord> getTopDifficultTopics() {
        return topicDifficultyRepository.findTop5ByOrderByDifficultyScoreDesc().stream()
                .map(item -> new TopicDifficultyRecord(
                        item.getTopicName(),
                        item.getDifficultyScore() == null ? 0f : item.getDifficultyScore(),
                        item.getQueryCount() == null ? 0 : item.getQueryCount()
                ))
                .toList();
    }

    /**
     * Retrieves students flagged as CRITICAL or WARNING.
     */
    public List<AtRiskStudentRecord> getAtRiskStudents() {
        String sql = """
            SELECT u.full_name, u.email, a.risk_level, a.reason 
            FROM analytics_at_risk_students a 
            JOIN users u ON a.student_id = u.id 
            WHERE a.risk_level IN ('CRITICAL', 'WARNING')
            ORDER BY a.report_date DESC
            LIMIT 100
            """;

        return jdbcClient.sql(sql)
                .query((rs, rowNum) -> new AtRiskStudentRecord(
                        rs.getString("full_name"),
                        rs.getString("email"),
                        rs.getString("risk_level"),
                        rs.getString("reason")
                ))
                .list();
    }

    public AnalyticsMetricsDTO getMetrics(LocalDateTime since) {
        List<AnalyticsDailySummary> summaries = dailySummaryRepository.findRecent(since.toLocalDate());
        
        int sensitiveDetections = summaries.stream().mapToInt(s -> s.getSensitiveDetections() == null ? 0 : s.getSensitiveDetections()).sum();
        int publicLeakPreventCount = summaries.stream().mapToInt(s -> s.getPublicLeakPreventCount() == null ? 0 : s.getPublicLeakPreventCount()).sum();
        double avgChannelSwitchRate = summaries.stream().mapToDouble(s -> s.getChannelSwitchRate() == null ? 0.0 : s.getChannelSwitchRate()).average().orElse(0.0);

        return AnalyticsMetricsDTO.builder()
                .ta_verification_accuracy(forumPostRepository.taVerificationAccuracySince(since))
                .correction_rate(forumPostRepository.correctionRateSince(since))
                .rejection_rate(forumPostRepository.rejectionRateSince(since))
                .helpful_feedback_rate(chatMessageRepository.helpfulFeedbackRateSince(since))
                .needs_ta_count(chatMessageRepository.needsTaCountSince(since))
                .pending_verification_count(forumPostRepository.pendingVerificationCountSince(since))
                .verified_count(forumPostRepository.verifiedCountSince(since))
                .corrected_count(forumPostRepository.correctedCountSince(since))
                .rejected_count(forumPostRepository.rejectedCountSince(since))
                .sensitive_detections(sensitiveDetections)
                .public_leak_prevention_count(publicLeakPreventCount)
                .channel_switch_rate(avgChannelSwitchRate)
                .build();
    }

    public List<StudentActivityPointRecord> getStudentActivity(LocalDateTime since) {
        String sql = """
            WITH days AS (
                SELECT generate_series(
                    date_trunc('day', :since::timestamp),
                    date_trunc('day', now()),
                    interval '1 day'
                )::date AS day
            ),
            threads_per_day AS (
                SELECT
                    date_trunc('day', ft.created_at)::date AS day,
                    COUNT(*)::int AS total_threads
                FROM forum_threads ft
                WHERE ft.created_at >= :since
                  AND ft.created_at <= now()
                GROUP BY 1
            ),
            ai_resolved_per_day AS (
                SELECT
                    date_trunc('day', fp.created_at)::date AS day,
                    COUNT(DISTINCT fp.thread_id)::int AS ai_resolved
                FROM forum_posts fp
                WHERE fp.author_type::text = 'AI'
                  AND fp.created_at >= :since
                  AND fp.created_at <= now()
                GROUP BY 1
            )
            SELECT
                to_char(d.day, 'YYYY-MM-DD') AS day,
                COALESCE(t.total_threads, 0) AS total_threads,
                COALESCE(a.ai_resolved, 0) AS ai_resolved
            FROM days d
            LEFT JOIN threads_per_day t ON t.day = d.day
            LEFT JOIN ai_resolved_per_day a ON a.day = d.day
            ORDER BY d.day
            """;

        return jdbcClient.sql(sql)
                .param("since", since)
                .query((rs, rowNum) -> new StudentActivityPointRecord(
                        rs.getString("day"),
                        rs.getInt("total_threads"),
                        rs.getInt("ai_resolved")
                ))
                .list();
    }
}
