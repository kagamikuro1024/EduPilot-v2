package team._8.aitrogiang.model;

import jakarta.persistence.*;
import lombok.*;

import java.time.LocalDate;
import java.util.UUID;

@Entity
@Table(name = "analytics_topic_difficulty")
@Getter
@Setter
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class AnalyticsTopicDifficulty {

    @Id
    @GeneratedValue(strategy = GenerationType.AUTO)
    private UUID id;

    @Column(name = "report_date")
    private LocalDate reportDate;

    @Column(name = "topic_name")
    private String topicName;

    @Column(name = "query_count")
    private Integer queryCount;

    @Column(name = "difficulty_score")
    private Float difficultyScore;
}
