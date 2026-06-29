package team._8.aitrogiang.model;

import jakarta.persistence.*;
import lombok.*;
import org.hibernate.annotations.JdbcTypeCode;
import org.hibernate.type.SqlTypes;

import java.time.LocalDate;
import java.util.UUID;

@Entity
@Table(name = "analytics_at_risk_students")
@Getter
@Setter
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class AnalyticsAtRiskStudent {

    @Id
    @GeneratedValue(strategy = GenerationType.AUTO)
    private UUID id;

    @Column(name = "report_date")
    private LocalDate reportDate;

    @Column(name = "student_id")
    private UUID studentId;

    @Enumerated(EnumType.STRING)
    @Column(name = "risk_level", columnDefinition = "risk_level_enum")
    @JdbcTypeCode(SqlTypes.NAMED_ENUM)
    private RiskLevel riskLevel;

    @Column(name = "reason", columnDefinition = "TEXT")
    private String reason;
}
