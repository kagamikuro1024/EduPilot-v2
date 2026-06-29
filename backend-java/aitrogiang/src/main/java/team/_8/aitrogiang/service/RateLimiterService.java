package team._8.aitrogiang.service;

import lombok.RequiredArgsConstructor;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.data.redis.core.script.RedisScript;
import org.springframework.stereotype.Service;
import team._8.aitrogiang.config.RateLimitProperties;
import team._8.aitrogiang.exception.RateLimitException;

import java.time.LocalDateTime;
import java.time.format.DateTimeFormatter;
import java.util.List;
import java.util.UUID;
import java.util.concurrent.TimeUnit;

@Service
@RequiredArgsConstructor
public class RateLimiterService {

    private final StringRedisTemplate redisTemplate;
    private final RedisScript<List> rateLimitScript;
    private final RateLimitProperties rateLimitProperties;

    public void checkOrThrow(UUID userId, RateLimitType type) {
        RateLimitProperties.Rule rule = resolveRule(type);
        String key = buildKey(userId, type);
        List<Long> result = redisTemplate.execute(
                rateLimitScript,
                List.of(key),
                String.valueOf(rule.getTtlSeconds()),
                String.valueOf(rule.getLimit())
        );

        if (result != null && result.size() >= 2) {
            Long count = result.get(0);
            Long ttl = result.get(1);
            if (count != null && count > rule.getLimit()) {
                throw new RateLimitException(ttl != null && ttl > 0 ? ttl : rule.getTtlSeconds());
            }
        }
    }

    private String buildKey(UUID userId, RateLimitType type) {
        RateLimitProperties.Rule rule = resolveRule(type);
        String suffix = LocalDateTime.now().format(DateTimeFormatter.ofPattern(rule.getKeyDateFormat()));
        return "rate:" + type.getKeyPrefix() + ":" + userId + ":" + suffix;
    }

    private RateLimitProperties.Rule resolveRule(RateLimitType type) {
        if (type == RateLimitType.ASK_AI) {
            return rateLimitProperties.getAskAi();
        }
        if (type == RateLimitType.CLASSIFY) {
            return rateLimitProperties.getClassify();
        }
        throw new IllegalArgumentException("Unsupported rate limit type: " + type);
    }
}
