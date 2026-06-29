package team._8.aitrogiang.repository;

import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;
import team._8.aitrogiang.model.ForumThread;
import team._8.aitrogiang.model.ThreadStatus;

import java.time.LocalDateTime;
import java.util.UUID;

@Repository
public interface ForumThreadRepository extends JpaRepository<ForumThread, UUID> {

    @Query("""
            SELECT DISTINCT t
            FROM ForumThread t
            LEFT JOIN ForumThreadTag ftt ON ftt.threadId = t.id
            LEFT JOIN Tag tag ON tag.id = ftt.tagId
            WHERE (cast(:status as string) IS NULL OR cast(t.status as string) = cast(:status as string))
              AND (cast(:tag as string) IS NULL OR LOWER(tag.name) = LOWER(cast(:tag as string)))
              AND (cast(:search as string) IS NULL OR LOWER(t.title) LIKE LOWER(CONCAT('%', cast(:search as string), '%')))
              AND (cast(t.status as string) != 'ESCALATED' OR t.authorId = :userId OR :isStaff = true)
            """)
    Page<ForumThread> findWithFilters(
            @Param("status") ThreadStatus status,
            @Param("tag") String tag,
            @Param("search") String search,
            @Param("userId") UUID userId,
            @Param("isStaff") boolean isStaff,
            Pageable pageable
    );

    @Query("SELECT COUNT(ft) FROM ForumThread ft WHERE ft.createdAt >= :since")
    long countSince(@Param("since") LocalDateTime since);
}
