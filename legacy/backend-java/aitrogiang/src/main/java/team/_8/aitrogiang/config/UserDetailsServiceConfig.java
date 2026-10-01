package team._8.aitrogiang.config;

import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.security.core.userdetails.UserDetailsService;
import org.springframework.security.core.userdetails.UsernameNotFoundException;
import team._8.aitrogiang.repository.UserRepository;

/**
 * Defines {@link UserDetailsService} outside {@link SecurityConfig} so that
 * {@code JwtAuthFilter} can be constructed without first initializing {@code SecurityConfig},
 * breaking the circular dependency: SecurityConfig -> JwtAuthFilter -> UserDetailsService -> SecurityConfig.
 */
@Configuration
public class UserDetailsServiceConfig {

    @Bean
    public UserDetailsService userDetailsService(UserRepository userRepository) {
        return username -> userRepository.findByEmail(username)
                .orElseThrow(() -> new UsernameNotFoundException("User not found"));
    }
}
