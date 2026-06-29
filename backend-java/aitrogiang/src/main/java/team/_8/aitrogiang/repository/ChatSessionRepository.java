package team._8.aitrogiang.repository;

import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.stereotype.Repository;
import team._8.aitrogiang.model.ChatSession;

import java.util.Optional;
import java.util.List;
import java.util.UUID;

@Repository
public interface ChatSessionRepository extends JpaRepository<ChatSession, UUID> {

    /**
     * Find the most recent session for a given student.
     * For MVP: one active session per student.
     */
    Optional<ChatSession> findFirstByStudentIdOrderByCreatedAtDesc(UUID studentId);

    /**
     * Load all sessions for a student, newest first.
     */
    List<ChatSession> findByStudentIdOrderByCreatedAtDesc(UUID studentId);
}
