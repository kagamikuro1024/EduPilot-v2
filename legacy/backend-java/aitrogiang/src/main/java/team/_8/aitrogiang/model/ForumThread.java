package team._8.aitrogiang.model;

import jakarta.persistence.*;
import lombok.*;
import org.hibernate.annotations.JdbcTypeCode;
import org.hibernate.type.SqlTypes;

import java.time.Clock;
import java.time.LocalDateTime;
import java.util.UUID;

@Entity
@Table(name = "forum_threads")
@Getter
@Setter
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class ForumThread {

    @Id
    @GeneratedValue(strategy = GenerationType.AUTO)
    private UUID id;

    @Column(name = "author_id")
    private UUID authorId;

    @Column(name = "title")
    private String title;

    @Enumerated(EnumType.STRING)
    @Column(name = "status", columnDefinition = "thread_status")
    @JdbcTypeCode(SqlTypes.NAMED_ENUM)
    private ThreadStatus status;

    @Column(name = "created_at", updatable = false)
    private LocalDateTime createdAt;

    @PrePersist
    protected void onCreate() {
        createdAt = LocalDateTime.now(Clock.systemUTC());
    }
}
