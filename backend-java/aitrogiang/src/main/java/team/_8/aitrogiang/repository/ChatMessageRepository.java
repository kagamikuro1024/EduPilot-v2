package team._8.aitrogiang.repository;

import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;
import team._8.aitrogiang.model.ChatMessage;

import java.time.LocalDateTime;
import java.util.List;
import java.util.UUID;

@Repository
public interface ChatMessageRepository extends JpaRepository<ChatMessage, UUID> {

    /**
     * Load full conversation history for a session, ordered chronologically.
     * Used to build the history payload sent to the Python AI service via gRPC.
     */
    List<ChatMessage> findBySessionIdOrderByCreatedAtAsc(UUID sessionId);

    /**
     * Delete all messages for a session (used when deleting a session).
     */
    void deleteBySessionId(UUID sessionId);

    @Query(
            value = """
                    SELECT CAST(
                        SUM(CASE WHEN cm.feedback::text = 'LIKE' THEN 1 ELSE 0 END)
                        AS double precision
                    ) / NULLIF(
                        SUM(CASE WHEN cm.feedback IS NOT NULL THEN 1 ELSE 0 END), 0
                    )
                    FROM chat_messages cm
                    WHERE cm.sender::text = 'AI'
                      AND cm.created_at >= :since
                    """,
            nativeQuery = true
    )
    Double helpfulFeedbackRateSince(@Param("since") LocalDateTime since);

    @Query(
            value = """
                    SELECT COUNT(*)::int
                    FROM chat_messages cm
                    WHERE cm.sender::text = 'AI'
                      AND cm.feedback::text = 'NEEDS_TA'
                      AND cm.created_at >= :since
                    """,
            nativeQuery = true
    )
    Integer needsTaCountSince(@Param("since") LocalDateTime since);
}
