package team._8.aitrogiang.util;

import org.junit.jupiter.api.Test;
import team._8.aitrogiang.grpc.ClassifyResponse;
import team._8.aitrogiang.grpc.IntentType;

import java.nio.charset.StandardCharsets;
import java.util.Base64;
import java.util.Map;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;

import static org.assertj.core.api.Assertions.assertThat;

class JavaPreflightTagBuilderTest {

    private static final ObjectMapper MAPPER = new ObjectMapper();

    @Test
    void buildTag_encodesJson_and_stripsTaskIntentSuffixFromReasoning() throws Exception {
        ClassifyResponse response = ClassifyResponse.newBuilder()
                .setSuggestedChannel(IntentType.PUBLIC)
                .setConfidence(0.91f)
                .setReasoning("Hello||TASK_INTENT:PROCEDURAL||")
                .setIsViolation(false)
                .build();

        String tag = JavaPreflightTagBuilder.buildTag(response);
        assertThat(tag).startsWith(JavaPreflightTagBuilder.PREFIX);

        String b64 = tag.substring(JavaPreflightTagBuilder.PREFIX.length());
        byte[] jsonBytes = Base64.getUrlDecoder().decode(b64);
        Map<String, Object> map = MAPPER.readValue(jsonBytes, new TypeReference<>() {});

        assertThat(map.get("v")).isEqualTo(1);
        assertThat(map.get("ch")).isEqualTo("PUBLIC");
        assertThat(map.get("conf")).isEqualTo(0.91);
        assertThat(map.get("reason")).isEqualTo("Hello");
        assertThat(map.get("viol")).isEqualTo(false);
        assertThat(map.get("task")).isEqualTo("PROCEDURAL");
    }

    @Test
    void buildTag_defaultsTaskToUncertain_whenNoSuffix() {
        ClassifyResponse response = ClassifyResponse.newBuilder()
                .setSuggestedChannel(IntentType.PRIVATE)
                .setConfidence(0.5f)
                .setReasoning("No machine suffix here")
                .setIsViolation(false)
                .build();

        String tag = JavaPreflightTagBuilder.buildTag(response);
        String b64 = tag.substring(JavaPreflightTagBuilder.PREFIX.length());
        String json = new String(Base64.getUrlDecoder().decode(b64), StandardCharsets.UTF_8);
        assertThat(json).contains("\"task\":\"UNCERTAIN\"");
        assertThat(json).contains("No machine suffix here");
    }
}
