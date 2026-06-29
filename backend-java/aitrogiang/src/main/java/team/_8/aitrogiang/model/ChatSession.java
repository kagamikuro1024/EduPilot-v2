package team._8.aitrogiang.model;

import jakarta.persistence.*;
import lombok.*;

import java.time.Clock;
import java.time.LocalDateTime;
import java.util.UUID;

/**
 * Maps to PostgreSQL table: chat_sessions
 * Owned by Java (State Management Domain) — Python is STATELESS and never touches this.
 */
@Entity
@Table(name = "chat_sessions")
@Getter
@Setter
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class ChatSession {

    @Id
    @GeneratedValue(strategy = GenerationType.AUTO)
    private UUID id;

    /**
     * FK → users.id (not enforced at JPA level to match current schema).
     * Java resolves the User from SecurityContext and binds it here.
     */
    @Column(name = "student_id", nullable = false)
    private UUID studentId;

    @Column(name = "created_at", updatable = false)
    private LocalDateTime createdAt;

    @PrePersist
    protected void onCreate() {
        createdAt = LocalDateTime.now(Clock.systemUTC());
    }
}
