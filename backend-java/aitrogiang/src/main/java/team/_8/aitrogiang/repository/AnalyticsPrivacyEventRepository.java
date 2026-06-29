package team._8.aitrogiang.repository;

import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.stereotype.Repository;
import team._8.aitrogiang.model.AnalyticsPrivacyEvent;

import java.util.UUID;

@Repository
public interface AnalyticsPrivacyEventRepository extends JpaRepository<AnalyticsPrivacyEvent, UUID> {
}
