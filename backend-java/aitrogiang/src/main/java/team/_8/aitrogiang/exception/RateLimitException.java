package team._8.aitrogiang.exception;

import lombok.Getter;

@Getter
public class RateLimitException extends RuntimeException {
    private final long retryAfterSeconds;

    public RateLimitException(long retryAfterSeconds) {
        super("Rate limit exceeded. Please retry later.");
        this.retryAfterSeconds = retryAfterSeconds;
    }
}
