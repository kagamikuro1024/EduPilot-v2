package team._8.aitrogiang.service;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.jdbc.core.simple.JdbcClient;
import team._8.aitrogiang.repository.*;
import team._8.aitrogiang.dto.AnalyticsMetricsDTO;

import java.time.LocalDateTime;
import java.util.Collections;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
public class AnalyticsServiceTest {

    @Mock
    private JdbcClient jdbcClient;
    @Mock
    private ForumPostRepository forumPostRepository;
    @Mock
    private ChatMessageRepository chatMessageRepository;
    @Mock
    private AnalyticsTopicDifficultyRepository topicDifficultyRepository;
    @Mock
    private AnalyticsAtRiskStudentRepository atRiskStudentRepository;
    @Mock
    private AnalyticsDailySummaryRepository dailySummaryRepository;

    @InjectMocks
    private AnalyticsService analyticsService;

    @Test
    public void getTopDifficultTopics_ShouldUseRepository() {
        // Arrange
        when(topicDifficultyRepository.findTop5ByOrderByDifficultyScoreDesc()).thenReturn(Collections.emptyList());
        
        // Act
        analyticsService.getTopDifficultTopics();

        // Assert
        verify(topicDifficultyRepository).findTop5ByOrderByDifficultyScoreDesc();
    }

    @Test
    public void getMetrics_ShouldAggregateFromDailySummary() {
        // Arrange
        when(dailySummaryRepository.findRecent(any())).thenReturn(Collections.emptyList());
        when(forumPostRepository.taVerificationAccuracySince(any())).thenReturn(0.9);
        when(chatMessageRepository.helpfulFeedbackRateSince(any())).thenReturn(0.8);

        // Act
        AnalyticsMetricsDTO metrics = analyticsService.getMetrics(LocalDateTime.now().minusDays(7));

        // Assert
        assertThat(metrics.getTa_verification_accuracy()).isEqualTo(0.9);
        assertThat(metrics.getHelpful_feedback_rate()).isEqualTo(0.8);
        assertThat(metrics.getSensitive_detections()).isEqualTo(0);
        verify(dailySummaryRepository).findRecent(any());
    }
}
