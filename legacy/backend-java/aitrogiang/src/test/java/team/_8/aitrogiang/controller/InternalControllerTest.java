package team._8.aitrogiang.controller;

import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.http.MediaType;
import org.springframework.test.context.TestPropertySource;
import org.springframework.test.web.servlet.MockMvc;
import team._8.aitrogiang.model.DocumentStatus;
import team._8.aitrogiang.service.AdminService;

import java.util.UUID;

import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.verifyNoInteractions;
import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors.csrf;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.patch;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@WebMvcTest(InternalController.class)
@AutoConfigureMockMvc(addFilters = false)
@TestPropertySource(properties = "app.internal.token=internal-secret")
class InternalControllerTest {

    @Autowired
    private MockMvc mockMvc;

    @MockBean
    private AdminService adminService;

    @MockBean
    private team._8.aitrogiang.security.JwtService jwtService;

    @Test
    void shouldRejectWhenTokenMissing() throws Exception {
        UUID id = UUID.randomUUID();

        mockMvc.perform(patch("/api/v1/internal/documents/{id}/status", id)
                        .with(csrf())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {"status":"READY"}
                                """))
                .andExpect(status().isForbidden())
                .andExpect(jsonPath("$.error").value("FORBIDDEN"));

        verifyNoInteractions(adminService);
    }

    @Test
    void shouldRejectWhenTokenInvalid() throws Exception {
        UUID id = UUID.randomUUID();

        mockMvc.perform(patch("/api/v1/internal/documents/{id}/status", id)
                        .with(csrf())
                        .header("X-Internal-Token", "wrong-token")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {"status":"READY"}
                                """))
                .andExpect(status().isForbidden())
                .andExpect(jsonPath("$.error").value("FORBIDDEN"));

        verifyNoInteractions(adminService);
    }

    @Test
    void shouldRejectInvalidStatus() throws Exception {
        UUID id = UUID.randomUUID();

        mockMvc.perform(patch("/api/v1/internal/documents/{id}/status", id)
                        .with(csrf())
                        .header("X-Internal-Token", "internal-secret")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {"status":"UNKNOWN_STATUS"}
                                """))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.error").value("VALIDATION_ERROR"));

        verifyNoInteractions(adminService);
    }

    @Test
    void shouldRejectBlankStatus() throws Exception {
        UUID id = UUID.randomUUID();

        mockMvc.perform(patch("/api/v1/internal/documents/{id}/status", id)
                        .with(csrf())
                        .header("X-Internal-Token", "internal-secret")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {"status":"   "}
                                """))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.error").value("VALIDATION_ERROR"));

        verifyNoInteractions(adminService);
    }

    @Test
    void shouldAcceptValidCallbackStatus() throws Exception {
        UUID id = UUID.randomUUID();

        mockMvc.perform(patch("/api/v1/internal/documents/{id}/status", id)
                        .with(csrf())
                        .header("X-Internal-Token", "internal-secret")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {"status":"DUPLICATE","reason":"Duplicate hash"}
                                """))
                .andExpect(status().isNoContent());

        verify(adminService).updateDocumentStatus(eq(id), eq(DocumentStatus.DUPLICATE));
    }
}
