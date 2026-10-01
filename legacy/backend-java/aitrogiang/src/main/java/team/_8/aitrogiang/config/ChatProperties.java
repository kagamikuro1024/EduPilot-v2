package team._8.aitrogiang.config;

import lombok.Getter;
import lombok.Setter;
import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.stereotype.Component;

@Getter
@Setter
@Component
@ConfigurationProperties(prefix = "app.chat")
public class ChatProperties {

    /**
     * Java-side short-circuit threshold for intent confidence.
     * - Production: keep >= 0.4 for safer fallback behavior.
     * - Benchmark profile: can override to lower values for measurement.
     */
    private double confidenceThreshold = 0.4;
}

