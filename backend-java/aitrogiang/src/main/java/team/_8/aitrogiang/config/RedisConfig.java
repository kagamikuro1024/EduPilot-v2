package team._8.aitrogiang.config;

import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.data.redis.core.script.DefaultRedisScript;
import org.springframework.data.redis.core.script.RedisScript;

import java.util.List;

@Configuration
public class RedisConfig {

    @Bean
    public RedisScript<List> rateLimitScript() {
        DefaultRedisScript<List> script = new DefaultRedisScript<>();
        script.setScriptText(
                "local limit = tonumber(ARGV[2]) " +
                "local c = tonumber(redis.call('GET', KEYS[1]) or '0') " +
                "if c >= limit then " +
                "  return {c + 1, redis.call('TTL', KEYS[1])} " +
                "end " +
                "c = redis.call('INCR', KEYS[1]) " +
                "if c == 1 then redis.call('EXPIRE', KEYS[1], ARGV[1]) end " +
                "return {c, redis.call('TTL', KEYS[1])}"
        );
        script.setResultType(List.class);
        return script;
    }
}
