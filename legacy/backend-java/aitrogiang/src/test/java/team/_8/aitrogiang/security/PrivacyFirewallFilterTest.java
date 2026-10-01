package team._8.aitrogiang.security;

import com.fasterxml.jackson.databind.ObjectMapper;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.MockMvc;
import team._8.aitrogiang.dto.AskAiRequest;
import team._8.aitrogiang.IntegrationTestBase;

import static org.assertj.core.api.Assertions.assertThat;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

@SpringBootTest
@AutoConfigureMockMvc
public class PrivacyFirewallFilterTest extends IntegrationTestBase {

    @Autowired
    private MockMvc mockMvc;

    @Autowired
    private ObjectMapper objectMapper;

    private AskAiRequest request;

    @BeforeEach
    void setUp() {
        request = new AskAiRequest();
    }

    @Test
    void shouldBlockRequestContainingMSSV() throws Exception {
        request.setMessage("My Student ID is 20210001. Please help.");

        mockMvc.perform(post("/api/v1/threads/123/ask-ai")
                .contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(request)))
                .andExpect(status().isForbidden())
                .andExpect(jsonPath("$.error_code").value("ERR_PII_DETECTED"))
                .andExpect(jsonPath("$.error").value("ERR_PII_DETECTED"))
                .andExpect(jsonPath("$.message").value("Vui lòng chuyển sang khung chat Private để gửi thông tin cá nhân."))
                .andExpect(jsonPath("$.suggested_channel").value("PRIVATE"));
    }

    @Test
    void shouldBlockRequestContainingPhoneNumber() throws Exception {
        request.setMessage("Call me at 0912345678");

        mockMvc.perform(post("/api/v1/threads/123/ask-ai")
                .contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(request)))
                .andExpect(status().isForbidden())
                .andExpect(jsonPath("$.error_code").value("ERR_PII_DETECTED"))
                .andExpect(jsonPath("$.suggested_channel").value("PRIVATE"))
                .andExpect(jsonPath("$.error").value("ERR_PII_DETECTED"));
    }

    @Test
    void shouldAllowSafeRequest() throws Exception {
        request.setMessage("Tell me about memory management in C++");

        // Note: For this to work in a real Spring Boot test with Security, 
        // we might need to mock authentication or use @WithMockUser
        // But the firewall filter runs BEFORE authentication in our config.
        
        var result = mockMvc.perform(post("/api/v1/threads/123/ask-ai")
                .contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(request)))
                .andReturn();
        // Not blocked by PII firewall (unlike 403 + ERR_PII_DETECTED)
        assertThat(result.getResponse().getContentAsString()).doesNotContain("ERR_PII_DETECTED");
    }

    @Test
    void shouldBlockLargePayload() throws Exception {
        // Generate 150KB of data
        String largeData = "A".repeat(153600); 
        AskAiRequest largeRequest = new AskAiRequest();
        largeRequest.setMessage(largeData);

        mockMvc.perform(post("/api/v1/threads/123/ask-ai")
                .contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(largeRequest)))
                .andExpect(status().isPayloadTooLarge());
    }

    @Test
    void shouldBlockChunkedLargePayload() throws Exception {
        // Simulating a large payload that might be chunked
        byte[] largeBytes = new byte[150000];
        java.util.Arrays.fill(largeBytes, (byte) 'A');

        mockMvc.perform(post("/api/v1/threads/123/ask-ai")
                .contentType(MediaType.APPLICATION_JSON)
                .content(largeBytes))
                .andExpect(status().isPayloadTooLarge());
    }
}
