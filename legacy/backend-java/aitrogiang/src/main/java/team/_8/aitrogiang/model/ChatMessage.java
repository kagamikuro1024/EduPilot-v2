package team._8.aitrogiang.model;

import jakarta.persistence.*;
import lombok.*;
import org.hibernate.annotations.ColumnTransformer;
import org.hibernate.annotations.JdbcTypeCode;
import org.hibernate.type.SqlTypes;

import java.time.Clock;
import java.time.LocalDateTime;
import java.util.UUID;

/**
 * Maps to PostgreSQL table: chat_messages
 * Java persists both the user's input and the AI's complete response.
 * Python is STATELESS — it never reads or writes to this table.
 */
@Entity
@Table(name = "chat_messages")
@Getter
@Setter
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class ChatMessage {

    @Id
    @GeneratedValue(strategy = GenerationType.AUTO)
    private UUID id;

    /** FK → chat_sessions.id */
    @Column(name = "session_id", nullable = false)
    private UUID sessionId;

    /** STUDENT | AI | TA — stored as VARCHAR to match PostgreSQL ENUM */
    @Enumerated(EnumType.STRING)
    @Column(name = "sender", columnDefinition = "message_sender")
    @org.hibernate.annotations.JdbcTypeCode(org.hibernate.type.SqlTypes.NAMED_ENUM)
    private MessageSenderType sender;

    @Column(columnDefinition = "TEXT")
    private String content;

    /**
     * User feedback on an AI message. NULL until user interacts.
     * Stored as VARCHAR to match PostgreSQL ENUM: feedback_enum ('LIKE','DISLIKE','NEEDS_TA')
     */
    @Column(name = "feedback")
    @ColumnTransformer(write = "?::feedback_enum")
    private String feedback;

    /** RAG citation metadata as JSONB. NULL for non-AI messages. */
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
