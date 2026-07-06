package team._8.aitrogiang.integration;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.condition.EnabledIfEnvironmentVariable;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import team._8.aitrogiang.IntegrationTestBase;
import team._8.aitrogiang.grpc.ClassifyResponse;
import team._8.aitrogiang.grpc.IntentType;
import team._8.aitrogiang.model.ForumThread;
import team._8.aitrogiang.model.ThreadStatus;
import team._8.aitrogiang.model.User;
import team._8.aitrogiang.repository.ForumThreadRepository;
import team._8.aitrogiang.service.ChatChannel;
import team._8.aitrogiang.service.ChatStreamingService;
import team._8.aitrogiang.service.PythonAiOrchestratorClient;

import java.util.UUID;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.Mockito.when;

/**
 * Verifies the Graceful Fallback (Dual-channel) logic.
 * If AI confidence is low, the thread status should be set to ESCALATED.
 */
@SpringBootTest
@EnabledIfEnvironmentVariable(named = "RUN_INTEGRATION_TESTS", matches = "true")
public class AiFallbackIntegrationTest extends IntegrationTestBase {

    @Autowired
    private ChatStreamingService chatStreamingService;

    @Autowired
    private ForumThreadRepository forumThreadRepository;

    @MockBean
    private PythonAiOrchestratorClient pythonClient;

    @Test
    void testLowConfidenceEscalation() {
        // 1. Setup a test thread
        UUID threadId = UUID.randomUUID();
        ForumThread thread = ForumThread.builder()
                .id(threadId)
                .title("Test Question")
                .status(ThreadStatus.OPEN)
                .build();
        forumThreadRepository.save(thread);

        // 2. Mock low confidence response from Python (TD-05)
        ClassifyResponse mockResponse = ClassifyResponse.newBuilder()
                .setConfidence(0.6f) // Below threshold 0.8
                .setSuggestedChannel(IntentType.PUBLIC)
                .build();
        when(pythonClient.classifyIntent(anyString(), anyString())).thenReturn(mockResponse);
        
        // Mock the streamResponse to return empty (we are testing fallback, not the stream result)
        when(pythonClient.streamResponse(anyString(), anyString(), anyString(), any(), anyString(), any(), anyString(), any(), anyString()))
                .thenReturn(reactor.core.publisher.Flux.empty());

        // 3. Trigger the process
        User mockUser = new User();
        mockUser.setId(UUID.randomUUID());
        mockUser.setFullName("Test Student");
        
        try {
            chatStreamingService.processAndStream(threadId, "A very difficult question", mockUser, ChatChannel.FORUM)
                    .blockLast();
        } catch (Exception ignored) {}

        // 4. Verify thread is ESCALATED (Scenario 3)
        ForumThread updatedThread = forumThreadRepository.findById(threadId).orElseThrow();
        assertThat(updatedThread.getStatus()).isEqualTo(ThreadStatus.ESCALATED);
    }
}
