package team._8.aitrogiang.repository;

import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;
import team._8.aitrogiang.model.ForumPost;

import java.time.LocalDateTime;
import java.util.List;
import java.util.UUID;

@Repository
public interface ForumPostRepository extends JpaRepository<ForumPost, UUID> {

    List<ForumPost> findByThreadIdOrderByCreatedAtAsc(UUID threadId);

    long countByThreadId(UUID threadId);

    ForumPost findFirstByThreadIdOrderByCreatedAtDesc(UUID threadId);

    @Query(
            value = """
                    SELECT CAST(
                        SUM(CASE WHEN fp.author_type::text = 'AI' THEN 1 ELSE 0 END)
                        AS double precision
                    ) / NULLIF(COUNT(fp.id), 0)
                    FROM forum_posts fp
                    WHERE fp.created_at >= :since
                    """,
            nativeQuery = true
    )
    Double aiResolutionRateSince(@Param("since") LocalDateTime since);

    @Query(
            value = """
                    SELECT CAST(
                        SUM(CASE WHEN fp.verification_status::text = 'VERIFIED' THEN 1 ELSE 0 END)
                        AS double precision
                    ) / NULLIF(
                        SUM(CASE WHEN fp.verification_status::text IN ('VERIFIED', 'CORRECTED', 'REJECTED') THEN 1 ELSE 0 END), 0
                    )
                    FROM forum_posts fp
                    WHERE fp.author_type::text = 'AI'
                      AND fp.created_at >= :since
                    """,
            nativeQuery = true
    )
    Double taVerificationAccuracySince(@Param("since") LocalDateTime since);

    @Query(
            value = """
                    SELECT COUNT(*)::int
                    FROM forum_posts fp
                    WHERE fp.author_type::text = 'AI'
                      AND fp.verification_status::text = 'UNVERIFIED'
                      AND fp.created_at >= :since
                    """,
            nativeQuery = true
    )
    Integer pendingVerificationCountSince(@Param("since") LocalDateTime since);

    @Query(
            value = """
                    SELECT COUNT(*)::int
                    FROM forum_posts fp
                    WHERE fp.author_type::text = 'AI'
                      AND fp.verification_status::text = 'VERIFIED'
                      AND fp.created_at >= :since
                    """,
            nativeQuery = true
    )
    Integer verifiedCountSince(@Param("since") LocalDateTime since);

    @Query(
            value = """
                    SELECT COUNT(*)::int
                    FROM forum_posts fp
                    WHERE fp.author_type::text = 'AI'
                      AND fp.verification_status::text = 'CORRECTED'
                      AND fp.created_at >= :since
                    """,
            nativeQuery = true
    )
    Integer correctedCountSince(@Param("since") LocalDateTime since);

    @Query(
            value = """
                    SELECT COUNT(*)::int
                    FROM forum_posts fp
                    WHERE fp.author_type::text = 'AI'
                      AND fp.verification_status::text = 'REJECTED'
                      AND fp.created_at >= :since
                    """,
            nativeQuery = true
    )
    Integer rejectedCountSince(@Param("since") LocalDateTime since);

    @Query(
            value = """
                    SELECT CAST(
                        SUM(CASE WHEN fp.verification_status::text = 'CORRECTED' THEN 1 ELSE 0 END)
                        AS double precision
                    ) / NULLIF(
                        SUM(CASE WHEN fp.verification_status::text IN ('VERIFIED', 'CORRECTED', 'REJECTED') THEN 1 ELSE 0 END), 0
                    )
                    FROM forum_posts fp
                    WHERE fp.author_type::text = 'AI'
                      AND fp.created_at >= :since
                    """,
            nativeQuery = true
    )
    Double correctionRateSince(@Param("since") LocalDateTime since);

    @Query(
            value = """
                    SELECT CAST(
                        SUM(CASE WHEN fp.verification_status::text = 'REJECTED' THEN 1 ELSE 0 END)
                        AS double precision
                    ) / NULLIF(
                        SUM(CASE WHEN fp.verification_status::text IN ('VERIFIED', 'CORRECTED', 'REJECTED') THEN 1 ELSE 0 END), 0
                    )
                    FROM forum_posts fp
                    WHERE fp.author_type::text = 'AI'
                      AND fp.created_at >= :since
                    """,
            nativeQuery = true
    )
    Double rejectionRateSince(@Param("since") LocalDateTime since);
}
