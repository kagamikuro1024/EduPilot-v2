package team._8.aitrogiang.repository;

import org.springframework.data.jpa.repository.JpaRepository;
import team._8.aitrogiang.model.User;
import java.util.Optional;
import java.util.UUID;

public interface UserRepository extends JpaRepository<User, UUID> {
    Optional<User> findByEmail(String email);
}
