package team._8.aitrogiang.model;

import jakarta.persistence.*;
import lombok.*;
import org.hibernate.annotations.JdbcTypeCode;
import org.hibernate.type.SqlTypes;

import java.time.Clock;
import java.time.LocalDateTime;
import java.util.UUID;

@Entity
@Table(name = "forum_posts")
@Getter
@Setter
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class ForumPost {

    @Id
    @GeneratedValue(strategy = GenerationType.AUTO)
    private UUID id;

    @Column(name = "thread_id")
    private UUID threadId;

    @Column(name = "author_id")
    private UUID authorId;

    @Enumerated(EnumType.STRING)
    @Column(name = "author_type", columnDefinition = "post_author_type")
    @JdbcTypeCode(SqlTypes.NAMED_ENUM)
    private PostAuthorType authorType;

    @Column(name = "content", columnDefinition = "TEXT")
    private String content;

    @Column(name = "original_ai_content", columnDefinition = "TEXT")
    private String originalAiContent;

    @Enumerated(EnumType.STRING)
    @Column(name = "verification_status", columnDefinition = "verification_status")
    @JdbcTypeCode(SqlTypes.NAMED_ENUM)
    private VerificationStatus verificationStatus;

    @Column(name = "verified_by_ta_id")
    private UUID verifiedByTaId;

    @Column(name = "citations", columnDefinition = "jsonb")
    @JdbcTypeCode(SqlTypes.JSON)
    private String citations;

    @Column(name = "created_at", updatable = false)
    private LocalDateTime createdAt;

    @PrePersist
    protected void onCreate() {
        createdAt = LocalDateTime.now(Clock.systemUTC());
    }
}
