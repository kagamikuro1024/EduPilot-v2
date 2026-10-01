package team._8.aitrogiang.controller;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.http.MediaType;
import org.springframework.security.authentication.UsernamePasswordAuthenticationToken;
import org.springframework.security.core.Authentication;
import org.springframework.test.web.servlet.MockMvc;
import team._8.aitrogiang.dto.*;
import team._8.aitrogiang.model.*;
import team._8.aitrogiang.security.JwtService;
import team._8.aitrogiang.service.UserProfileService;

import java.time.LocalDateTime;
import java.util.UUID;

import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.*;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.*;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@WebMvcTest(UserController.class)
@AutoConfigureMockMvc(addFilters = false)
public class UserControllerTest {

    @Autowired
    private MockMvc mockMvc;

    @MockBean
    private UserProfileService userProfileService;

    @MockBean
    private JwtService jwtService;

    private User studentUser;

    @BeforeEach
    public void setUp() {
        studentUser = User.builder()
                .id(UUID.randomUUID())
                .email("student@example.com")
                .password("hashedpwd")
                .fullName("Student Name")
                .studentCode("22123456")
                .role(UserRole.STUDENT)
                .createdAt(LocalDateTime.now())
                .build();
    }

    private Authentication mockAuth() {
        return new UsernamePasswordAuthenticationToken(studentUser, null, studentUser.getAuthorities());
    }

    @Test
    public void testGetProfile_Success() throws Exception {
        UserProfileResponse profileResponse = UserProfileResponse.builder()
                .id(studentUser.getId())
                .full_name(studentUser.getFullName())
                .email(studentUser.getEmail())
                .student_code(studentUser.getStudentCode())
                .role(studentUser.getRole())
                .avatar_available(false)
                .build();

        when(userProfileService.getProfile(any(User.class))).thenReturn(profileResponse);

        mockMvc.perform(get("/api/v1/users/me")
                        .principal(mockAuth()))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.full_name").value("Student Name"))
                .andExpect(jsonPath("$.email").value("student@example.com"))
                .andExpect(jsonPath("$.student_code").value("22123456"))
                .andExpect(jsonPath("$.role").value("STUDENT"));
    }

    @Test
    public void testPatchProfile_Success() throws Exception {
        UserProfileResponse profileResponse = UserProfileResponse.builder()
                .id(studentUser.getId())
                .full_name("New Name")
                .email(studentUser.getEmail())
                .role(studentUser.getRole())
                .build();

        when(userProfileService.updateProfile(any(User.class), any(UserProfileUpdateRequest.class)))
                .thenReturn(profileResponse);

        mockMvc.perform(patch("/api/v1/users/me")
                        .principal(mockAuth())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"full_name\":\"New Name\"}"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.full_name").value("New Name"));
    }

    @Test
    public void testPatchProfile_ValidationError_ShouldReturn400() throws Exception {
        mockMvc.perform(patch("/api/v1/users/me")
                        .principal(mockAuth())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"full_name\":\"\"}")) // empty name
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.error").value("VALIDATION_ERROR"));
    }

    @Test
    public void testGetPreferences_Success() throws Exception {
        UserPreferencesResponse preferencesResponse = UserPreferencesResponse.builder()
                .theme(ThemePreference.DARK)
                .font_size(FontSizePreference.DEFAULT)
                .reduce_motion(false)
                .default_student_page(DefaultPagePreference.CHAT)
                .build();

        when(userProfileService.getPreferences(any(User.class))).thenReturn(preferencesResponse);

        mockMvc.perform(get("/api/v1/users/me/preferences")
                        .principal(mockAuth()))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.theme").value("DARK"))
                .andExpect(jsonPath("$.font_size").value("DEFAULT"))
                .andExpect(jsonPath("$.reduce_motion").value(false))
                .andExpect(jsonPath("$.default_student_page").value("CHAT"));
    }

    @Test
    public void testPatchPreferences_Success() throws Exception {
        UserPreferencesResponse preferencesResponse = UserPreferencesResponse.builder()
                .theme(ThemePreference.DARK)
                .font_size(FontSizePreference.DEFAULT)
                .reduce_motion(true)
                .default_student_page(DefaultPagePreference.ASSIGNMENTS)
                .build();

        when(userProfileService.updatePreferences(any(User.class), any(UserPreferencesUpdateRequest.class)))
                .thenReturn(preferencesResponse);

        mockMvc.perform(patch("/api/v1/users/me/preferences")
                        .principal(mockAuth())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"theme\":\"DARK\",\"reduce_motion\":true}"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.theme").value("DARK"))
                .andExpect(jsonPath("$.reduce_motion").value(true));
    }

    @Test
    public void testChangePassword_Success() throws Exception {
        doNothing().when(userProfileService).changePassword(any(User.class), any(PasswordChangeRequest.class));

        mockMvc.perform(patch("/api/v1/users/me/password")
                        .principal(mockAuth())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"current_password\":\"oldPassword\",\"new_password\":\"NewPassword123\"}"))
                .andExpect(status().isNoContent());

        verify(userProfileService, times(1)).changePassword(any(User.class), any(PasswordChangeRequest.class));
    }
}
