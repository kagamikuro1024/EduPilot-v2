package team._8.aitrogiang.util;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;

import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;
import java.nio.charset.StandardCharsets;
import java.util.Base64;
import java.util.LinkedHashMap;
import java.util.Map;

/** Builds a signed, server-owned identity context for the internal gRPC call. */
public final class AuthenticatedUserTagBuilder {

    public static final String PREFIX = "authenticated_user:v1:";
    private static final ObjectMapper MAPPER = new ObjectMapper();

    private AuthenticatedUserTagBuilder() {
    }

    public static String buildTag(String userId, String studentCode, String role, String secret) {
        if (userId == null || userId.isBlank() || secret == null || secret.isBlank()) {
            return null;
        }
        Map<String, Object> payload = new LinkedHashMap<>();
        payload.put("user_id", userId);
        payload.put("student_code", studentCode == null ? "" : studentCode.trim().toUpperCase());
        payload.put("role", role == null ? "" : role.trim().toUpperCase());
        try {
            String encodedPayload = Base64.getUrlEncoder().withoutPadding()
                    .encodeToString(MAPPER.writeValueAsBytes(payload));
            Mac mac = Mac.getInstance("HmacSHA256");
            mac.init(new SecretKeySpec(secret.getBytes(StandardCharsets.UTF_8), "HmacSHA256"));
            String signature = Base64.getUrlEncoder().withoutPadding()
                    .encodeToString(mac.doFinal(encodedPayload.getBytes(StandardCharsets.US_ASCII)));
            return PREFIX + encodedPayload + "." + signature;
        } catch (JsonProcessingException e) {
            throw new IllegalStateException("Authenticated user JSON encode failed", e);
        } catch (Exception e) {
            throw new IllegalStateException("Authenticated user tag signing failed", e);
        }
    }
}
