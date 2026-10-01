package team._8.aitrogiang.repository;

import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;
import team._8.aitrogiang.model.AnalyticsDailySummary;

import java.time.LocalDate;
import java.util.Optional;
import java.util.UUID;
import java.util.List;

@Repository
public interface AnalyticsDailySummaryRepository extends JpaRepository<AnalyticsDailySummary, UUID> {
    Optional<AnalyticsDailySummary> findByReportDate(LocalDate reportDate);

    @Query("SELECT a FROM AnalyticsDailySummary a WHERE a.reportDate >= :since ORDER BY a.reportDate DESC")
    List<AnalyticsDailySummary> findRecent(@Param("since") LocalDate since);
}
