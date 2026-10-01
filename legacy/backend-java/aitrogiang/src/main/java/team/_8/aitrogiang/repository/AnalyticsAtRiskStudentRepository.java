package team._8.aitrogiang.repository;

import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.stereotype.Repository;
import team._8.aitrogiang.model.AnalyticsAtRiskStudent;

import java.util.List;
import java.util.UUID;

@Repository
public interface AnalyticsAtRiskStudentRepository extends JpaRepository<AnalyticsAtRiskStudent, UUID> {
    List<AnalyticsAtRiskStudent> findTop10ByOrderByReportDateDesc();
}
