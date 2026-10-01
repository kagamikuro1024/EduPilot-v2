package team._8.aitrogiang.config;

import lombok.Getter;
import lombok.Setter;
import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.stereotype.Component;

@Getter
@Setter
@Component
@ConfigurationProperties(prefix = "app.rate-limit")
public class RateLimitProperties {

    private Rule askAi = new Rule(15L, 86400L, "yyyy-MM-dd");
    private Rule classify = new Rule(30L, 60L, "yyyy-MM-dd-HH-mm");

    @Getter
    @Setter
    public static class Rule {
        private long limit;
        private long ttlSeconds;
        private String keyDateFormat;

        public Rule() {
        }

        public Rule(long limit, long ttlSeconds, String keyDateFormat) {
            this.limit = limit;
            this.ttlSeconds = ttlSeconds;
            this.keyDateFormat = keyDateFormat;
        }
    }
}

