package team._8.aitrogiang.service;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import team._8.aitrogiang.dto.AnalyticsSummaryDTO;
import team._8.aitrogiang.dto.AdminDocumentStatsResponse;
import team._8.aitrogiang.model.DocumentStatus;
import team._8.aitrogiang.repository.*;

import java.util.Collections;
import java.util.List;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
class AdminServiceTest {

    @Mock
    private DocumentRepository documentRepository;
    @Mock
    private AnalyticsAtRiskStudentRepository atRiskStudentRepository;
    @Mock
    private AnalyticsTopicDifficultyRepository topicDifficultyRepository;
    @Mock
    private ForumThreadRepository forumThreadRepository;
    @Mock
    private ForumPostRepository forumPostRepository;
    @Mock
    private PythonAiOrchestratorClient pythonClient;
    @Mock
    private AnalyticsDailySummaryRepository dailySummaryRepository;

    private AdminService adminService;

    @BeforeEach
    void setUp() {
        adminService = new AdminService(
                documentRepository,
                atRiskStudentRepository,
                topicDifficultyRepository,
                forumThreadRepository,
                forumPostRepository,
                pythonClient
        );
    }

    @Test
    void getAnalyticsSummary_ShouldReturnRealDataFromRepositories() {
        // Arrange
        when(forumThreadRepository.countSince(any())).thenReturn(10L);
        when(forumPostRepository.aiResolutionRateSince(any())).thenReturn(0.85);
        when(topicDifficultyRepository.findTop10ByOrderByQueryCountDesc()).thenReturn(Collections.emptyList());
        when(atRiskStudentRepository.findTop10ByOrderByReportDateDesc()).thenReturn(Collections.emptyList());

        // Act
        AnalyticsSummaryDTO summary = adminService.getAnalyticsSummary();

        // Assert
        assertThat(summary.getTotal_questions_this_week()).isEqualTo(10L);
        assertThat(summary.getAi_resolution_rate()).isEqualTo(0.85);
        assertThat(summary.getTop_tags()).isEmpty();
        assertThat(summary.getAt_risk_students()).isEmpty();
    }

    @Test
    void getDocumentStats_ShouldReturnAggregatedMetrics() {
        // Arrange
        when(documentRepository.count()).thenReturn(5L);
        when(documentRepository.countByStatus(DocumentStatus.FAILED)).thenReturn(0L);
        when(documentRepository.countByStatus(DocumentStatus.READY)).thenReturn(4L);
        when(documentRepository.findAll()).thenReturn(List.of());

        // Act
        AdminDocumentStatsResponse stats = adminService.getDocumentStats();

        // Assert
        assertThat(stats.getTotal_documents()).isEqualTo(5L);
        assertThat(stats.getIndex_health()).isEqualTo("80%");
        assertThat(stats.getKnowledge_volume()).isEqualTo("0 B");
    }
}
