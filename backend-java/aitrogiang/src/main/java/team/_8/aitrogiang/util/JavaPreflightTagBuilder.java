package team._8.aitrogiang.util;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import team._8.aitrogiang.grpc.ClassifyResponse;

import java.nio.charset.StandardCharsets;
import java.util.Base64;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

/**
 * Encodes ClassifyIntent results as {@code java_preflight:v1:<base64url(json))} for {@link team._8.aitrogiang.service.PythonAiOrchestratorClient#streamResponse}
 * so Python can skip a duplicate {@code classify_and_guard} LLM call (no .proto change).
 */
public final class JavaPreflightTagBuilder {

    public static final String PREFIX = "java_preflight:v1:";

    private static final ObjectMapper MAPPER = new ObjectMapper();
    private static final Pattern TASK_INTENT = Pattern.compile("\\|\\|TASK_INTENT:([A-Z_]+)\\|\\|");

    private JavaPreflightTagBuilder() {
    }

    /**
     * Parses {@code ||TASK_INTENT:ACADEMIC||} suffix appended by Python ClassifyIntent and builds the tag payload.
     */
    public static String buildTag(ClassifyResponse r) {
        if (r == null) {
            return null;
        }
        String rawReasoning = r.getReasoning() != null ? r.getReasoning() : "";
        Matcher m = TASK_INTENT.matcher(rawReasoning);
        String taskIntent = "UNCERTAIN";
        String humanReasoning = rawReasoning.trim();
        if (m.find()) {
            taskIntent = m.group(1);
            humanReasoning = rawReasoning.substring(0, m.start()).trim();
        }
        Map<String, Object> payload = new LinkedHashMap<>();
        payload.put("v", 1);
        payload.put("ch", r.getSuggestedChannel().name());
        payload.put("conf", r.getConfidence());
        payload.put("reason", humanReasoning);
        payload.put("viol", r.getIsViolation());
        payload.put("task", taskIntent);
        try {
            String json = MAPPER.writeValueAsString(payload);
            return PREFIX + Base64.getUrlEncoder().withoutPadding()
                    .encodeToString(json.getBytes(StandardCharsets.UTF_8));
        } catch (JsonProcessingException e) {
            throw new IllegalStateException("java_preflight JSON encode failed", e);
        }
    }
}
