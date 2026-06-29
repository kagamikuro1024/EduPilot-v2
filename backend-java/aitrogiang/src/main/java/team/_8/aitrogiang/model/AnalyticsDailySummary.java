package team._8.aitrogiang.model;

import jakarta.persistence.*;
import lombok.*;
import java.time.LocalDate;
import java.time.LocalDateTime;
import java.util.UUID;

@Entity
@Table(name = "analytics_daily_summary")
@Getter
@Setter
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class AnalyticsDailySummary {

    @Id
    @GeneratedValue(strategy = GenerationType.AUTO)
    private UUID id;

    @Column(name = "report_date", unique = true, nullable = false)
    private LocalDate reportDate;

    @Column(name = "sensitive_detections")
    private Integer sensitiveDetections;

    @Column(name = "channel_switch_rate")
    private Float channelSwitchRate;

    @Column(name = "public_leak_prevent_count")
    private Integer publicLeakPreventCount;

    @Column(name = "false_positive_rate")
    private Float falsePositiveRate;

    @Column(name = "created_at", insertable = false, updatable = false)
    private LocalDateTime createdAt;
}
