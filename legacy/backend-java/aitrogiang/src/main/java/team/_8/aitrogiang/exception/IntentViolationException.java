package team._8.aitrogiang.exception;

public class IntentViolationException extends RuntimeException {
    private final Float confidence;

    public IntentViolationException(String message) {
        super(message);
        this.confidence = null;
    }

    public IntentViolationException(String message, Float confidence) {
        super(message);
        this.confidence = confidence;
    }

    public Float getConfidence() {
        return confidence;
    }
}
