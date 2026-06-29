package team._8.aitrogiang.controller;

import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.http.MediaType;
import org.springframework.http.codec.ServerSentEvent;
import org.springframework.security.authentication.UsernamePasswordAuthenticationToken;
import org.springframework.security.core.Authentication;
import org.springframework.test.web.servlet.MockMvc;
import reactor.core.publisher.Flux;
import team._8.aitrogiang.exception.IntentViolationException;
import team._8.aitrogiang.service.ChatChannel;
import team._8.aitrogiang.model.User;
import team._8.aitrogiang.model.UserRole;
import team._8.aitrogiang.security.JwtService;
import team._8.aitrogiang.service.ChatStreamingService;
import team._8.aitrogiang.service.ForumService;
import team._8.aitrogiang.service.PublicPrivacyFirewallService;
import team._8.aitrogiang.service.RateLimiterService;

import java.util.Map;
import java.util.UUID;

import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.doThrow;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@WebMvcTest(controllers = {ThreadController.class, ChatStreamingController.class})
@AutoConfigureMockMvc(addFilters = false)
class PublicPrivacyFirewallControllerTest {

    private static final String PRIVACY_MSG =
            "Noi dung co dau hieu thong tin rieng tu. Vui long su dung kenh chat private.";

    @Autowired
    private MockMvc mockMvc;

    @MockBean
    private ForumService forumService;

    @MockBean
    private ChatStreamingService chatStreamingService;

    @MockBean
    private RateLimiterService rateLimiterService;

    @MockBean
    private PublicPrivacyFirewallService privacyFirewallService;

    @MockBean
    private JwtService jwtService;

    @Test
    void createThread_shouldReturnForbidden_whenIntentViolationTriggered() throws Exception {
        doThrow(new IntentViolationException("Classifier temporarily unavailable", 0.0f))
                .when(forumService).createThread(any(), any());

        mockMvc.perform(post("/api/v1/threads")
                        .principal(authAsStudent())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "title": "MSSV 22123456 diem giua ky",
                                  "content": "MSSV 22123456 diem giua ky cua em la bao nhieu"
                                }
                                """))
                .andExpect(status().isForbidden())
                .andExpect(jsonPath("$.error_code").value("INTENT_VIOLATION"))
                .andExpect(jsonPath("$.error").value("INTENT_VIOLATION"))
                .andExpect(jsonPath("$.message").value(PRIVACY_MSG))
                .andExpect(jsonPath("$.suggested_channel").value("PRIVATE"));
    }

    @Test
    void sendMessage_shouldReturnForbidden_whenIntentViolationTriggered() throws Exception {
        doThrow(new IntentViolationException("Internal classifier detail", 0.2f))
                .when(forumService).createHumanReply(any(), anyString(), any());

        mockMvc.perform(post("/api/v1/threads/{thread_id}/messages", UUID.randomUUID())
                        .principal(authAsStudent())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "content": "MSSV 22123456 diem cua em"
                                }
                                """))
                .andExpect(status().isForbidden())
                .andExpect(jsonPath("$.error_code").value("INTENT_VIOLATION"))
                .andExpect(jsonPath("$.error").value("INTENT_VIOLATION"))
                .andExpect(jsonPath("$.message").value(PRIVACY_MSG))
                .andExpect(jsonPath("$.suggested_channel").value("PRIVATE"));
    }

    @Test
    void askAiForum_shouldReturnForbidden_whenPrivacyFirewallBlocks() throws Exception {
        doThrow(new IntentViolationException("Sensitive content", 0.9f))
                .when(privacyFirewallService).enforcePublicSafeOrThrow(eq("MSSV 22123456 diem giua ky"));

        mockMvc.perform(post("/api/v1/threads/{threadId}/ask-ai", UUID.randomUUID())
                        .principal(authAsStudent())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "message": "MSSV 22123456 diem giua ky"
                                }
                                """))
                .andExpect(status().isForbidden())
                .andExpect(jsonPath("$.error_code").value("INTENT_VIOLATION"))
                .andExpect(jsonPath("$.error").value("INTENT_VIOLATION"))
                .andExpect(jsonPath("$.message").value(PRIVACY_MSG))
                .andExpect(jsonPath("$.suggested_channel").value("PRIVATE"));
    }

    @Test
    void askAiForum_shouldReturnForbidden_forGradeAndGpaContent() throws Exception {
        doThrow(new IntentViolationException("Sensitive grade data", 0.9f))
                .when(privacyFirewallService).enforcePublicSafeOrThrow(eq("Diem GPA cua em hien tai la 3.8"));

        mockMvc.perform(post("/api/v1/threads/{threadId}/ask-ai", UUID.randomUUID())
                        .principal(authAsStudent())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "message": "Diem GPA cua em hien tai la 3.8"
                                }
                                """))
                .andExpect(status().isForbidden())
                .andExpect(jsonPath("$.error_code").value("INTENT_VIOLATION"))
                .andExpect(jsonPath("$.error").value("INTENT_VIOLATION"))
                .andExpect(jsonPath("$.message").value(PRIVACY_MSG))
                .andExpect(jsonPath("$.suggested_channel").value("PRIVATE"));
    }

    @Test
    void askAiForum_shouldReturnForbidden_forPersonalContactContent() throws Exception {
        doThrow(new IntentViolationException("Personal contact leak", 0.85f))
                .when(privacyFirewallService).enforcePublicSafeOrThrow(eq("So phone cua em la 0901234567, vui long lien he"));

        mockMvc.perform(post("/api/v1/threads/{threadId}/ask-ai", UUID.randomUUID())
                        .principal(authAsStudent())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "message": "So phone cua em la 0901234567, vui long lien he"
                                }
                                """))
                .andExpect(status().isForbidden())
                .andExpect(jsonPath("$.error_code").value("INTENT_VIOLATION"))
                .andExpect(jsonPath("$.error").value("INTENT_VIOLATION"))
                .andExpect(jsonPath("$.message").value(PRIVACY_MSG))
                .andExpect(jsonPath("$.suggested_channel").value("PRIVATE"));
    }

    @Test
    void askAiForum_shouldReturnForbidden_forMixedBenignAndSensitiveContent() throws Exception {
        doThrow(new IntentViolationException("Mixed sensitive payload", 0.88f))
                .when(privacyFirewallService).enforcePublicSafeOrThrow(eq("Cho em hoi recursion voi MSSV 22123456 va diem giua ky"));

        mockMvc.perform(post("/api/v1/threads/{threadId}/ask-ai", UUID.randomUUID())
                        .principal(authAsStudent())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "message": "Cho em hoi recursion voi MSSV 22123456 va diem giua ky"
                                }
                                """))
                .andExpect(status().isForbidden())
                .andExpect(jsonPath("$.error_code").value("INTENT_VIOLATION"))
                .andExpect(jsonPath("$.error").value("INTENT_VIOLATION"))
                .andExpect(jsonPath("$.message").value(PRIVACY_MSG))
                .andExpect(jsonPath("$.suggested_channel").value("PRIVATE"));
    }

    @Test
    void askAiForum_shouldNotPersistDuplicateQuestion_whenAutoTriggered() throws Exception {
        UUID threadId = UUID.randomUUID();
        when(chatStreamingService.processAndStream(any(), anyString(), any(), any(), eq(false)))
                .thenReturn(Flux.just(ServerSentEvent.builder()
                        .data(Map.of("chunk", "ok"))
                        .build()));

        mockMvc.perform(post("/api/v1/threads/{threadId}/ask-ai", threadId)
                        .principal(authAsStudent())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "message": "Explain recursion base cases",
                                  "autoTriggered": true
                                }
                                """))
                .andExpect(status().isOk());

        verify(chatStreamingService).processAndStream(
                eq(threadId),
                eq("Explain recursion base cases"),
                any(),
                eq(ChatChannel.FORUM),
                eq(false)
        );
    }

    private Authentication authAsStudent() {
        User principal = User.builder()
                .id(UUID.randomUUID())
                .email("student@example.com")
                .password("hashed")
                .fullName("Student A")
                .role(UserRole.STUDENT)
                .build();
        Authentication auth = new UsernamePasswordAuthenticationToken(
                principal,
                null,
                principal.getAuthorities()
        );
        return auth;
    }
}
