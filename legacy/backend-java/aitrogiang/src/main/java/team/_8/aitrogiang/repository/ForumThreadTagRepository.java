package team._8.aitrogiang.repository;

import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.stereotype.Repository;
import team._8.aitrogiang.model.ForumThreadTag;

import java.util.List;
import java.util.UUID;

@Repository
public interface ForumThreadTagRepository extends JpaRepository<ForumThreadTag, UUID> {

    List<ForumThreadTag> findByThreadId(UUID threadId);
}
