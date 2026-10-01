package team._8.aitrogiang.integration;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.condition.EnabledIfEnvironmentVariable;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.web.servlet.MockMvc;
import team._8.aitrogiang.IntegrationTestBase;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

/**
 * Verifies that the Redis-backed Rate Limiter works as expected.
 * After exceeding the threshold, the system should return 429 Too Many Requests.
 */
@SpringBootTest
@AutoConfigureMockMvc
@EnabledIfEnvironmentVariable(named = "RUN_INTEGRATION_TESTS", matches = "true")
public class RateLimitIntegrationTest extends IntegrationTestBase {

    @Autowired
    private MockMvc mockMvc;

    @Test
    void testRateLimitTrigger() throws Exception {
        // Assuming the limit is 50 requests per minute for this demonstration
        // We simulate 51 requests
        for (int i = 0; i < 50; i++) {
            mockMvc.perform(get("/api/v1/health"))
                   .andExpect(status().isOk());
        }

        // The 51st request should be throttled
        mockMvc.perform(get("/api/v1/health"))
               .andExpect(status().isTooManyRequests());
    }
}
