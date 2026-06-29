package team._8.aitrogiang.service;

import lombok.Getter;

@Getter
public enum RateLimitType {
    ASK_AI("ask-ai"),
    CLASSIFY("classify");

    private final String keyPrefix;

    RateLimitType(String keyPrefix) {
        this.keyPrefix = keyPrefix;
    }
}
