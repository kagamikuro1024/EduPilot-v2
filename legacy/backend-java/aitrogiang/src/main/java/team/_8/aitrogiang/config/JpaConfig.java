package team._8.aitrogiang.config;

import org.springframework.context.annotation.Configuration;
import org.springframework.data.jpa.repository.config.EnableJpaRepositories;

@Configuration
@EnableJpaRepositories(basePackages = "team._8.aitrogiang.repository")
public class JpaConfig {
}
