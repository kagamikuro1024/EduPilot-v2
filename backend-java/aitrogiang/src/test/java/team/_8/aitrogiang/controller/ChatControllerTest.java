package team._8.aitrogiang.controller;

import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.http.MediaType;
import org.springframework.security.test.context.support.WithMockUser;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.MvcResult;
import org.springframework.web.servlet.mvc.method.annotation.SseEmitter;
import team._8.aitrogiang.service.GrpcChatClientService;

import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.doAnswer;
import static org.mockito.Mockito.verify;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.asyncDispatch;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.request;

@WebMvcTest(ChatStreamController.class)
public class ChatControllerTest {

    @Autowired
    private MockMvc mockMvc;

    @MockBean
    private team._8.aitrogiang.security.JwtService jwtService;

    @MockBean
    private GrpcChatClientService grpcChatClientService;

    @Test
    @WithMockUser
    public void testGracefulFallbackOnGuardrailViolation() throws Exception {
        // Arrange
        String message = "test sensitive message";
        String fallbackMsg = "Vấn đề này vượt ngoài phạm vi trả lời của tôi, thông tin sẽ được chuyển cho TA, bạn chú ý theo dõi nhé.";

        // Simulate gRPC error FAILED_PRECONDITION being handled by the service and pushing to emitter
        doAnswer(invocation -> {
            SseEmitter emitter = invocation.getArgument(1);
            emitter.send(fallbackMsg);
            emitter.complete();
            return null;
        }).when(grpcChatClientService).streamChat(eq(message), any(SseEmitter.class));

        // Act
        MvcResult result = mockMvc.perform(get("/api/v1/chat/stream")
                        .param("message", message)
                        .accept(MediaType.TEXT_EVENT_STREAM))
                .andExpect(request().asyncStarted())
                .andReturn();

        MvcResult dispatched = mockMvc.perform(asyncDispatch(result))
                .andExpect(status().isOk())
                .andReturn();

        // Assert: stream endpoint remains healthy and delegates to gRPC client.
        verify(grpcChatClientService).streamChat(eq(message), any(SseEmitter.class));
    }
}
