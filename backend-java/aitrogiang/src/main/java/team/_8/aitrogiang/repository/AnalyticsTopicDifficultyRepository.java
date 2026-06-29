package team._8.aitrogiang.repository;

import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.stereotype.Repository;
import team._8.aitrogiang.model.AnalyticsTopicDifficulty;

import java.util.List;
import java.util.UUID;

@Repository
public interface AnalyticsTopicDifficultyRepository extends JpaRepository<AnalyticsTopicDifficulty, UUID> {
    List<AnalyticsTopicDifficulty> findTop10ByOrderByQueryCountDesc();
    List<AnalyticsTopicDifficulty> findTop5ByOrderByDifficultyScoreDesc();
}
