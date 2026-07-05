package team._8.aitrogiang.service;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import org.mockito.Mock;
import org.mockito.MockitoAnnotations;
import org.springframework.mock.web.MockMultipartFile;
import org.springframework.security.crypto.password.PasswordEncoder;
import team._8.aitrogiang.dto.*;
import team._8.aitrogiang.exception.*;
import team._8.aitrogiang.model.*;
import team._8.aitrogiang.repository.UserPreferencesRepository;
import team._8.aitrogiang.repository.UserRepository;
import team._8.aitrogiang.security.JwtService;

import javax.imageio.ImageIO;
import java.awt.image.BufferedImage;
import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.time.LocalDateTime;
import java.util.Optional;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.*;

public class UserProfileServiceTest {

    private UserProfileService userProfileService;

    @Mock
    private UserRepository userRepository;

    @Mock
    private UserPreferencesRepository userPreferencesRepository;

    @Mock
    private PasswordEncoder passwordEncoder;

    @TempDir
    Path tempDir;

    private User testUser;

    @BeforeEach
    public void setUp() {
        MockitoAnnotations.openMocks(this);
        userProfileService = new UserProfileService(userRepository, userPreferencesRepository, passwordEncoder);
        
        // Inject temp directory as app.upload.dir
        org.springframework.test.util.ReflectionTestUtils.setField(
                userProfileService, "uploadDir", tempDir.toAbsolutePath().toString());

        testUser = User.builder()
                .id(UUID.randomUUID())
                .email("student@example.com")
                .password("hashed_current_password")
                .fullName("Nguyen Van A")
                .studentCode("22123456")
                .role(UserRole.STUDENT)
                .createdAt(LocalDateTime.now())
                .build();
    }

    @Test
    public void testGetProfile() {
        UserProfileResponse response = userProfileService.getProfile(testUser);
        assertEquals(testUser.getId(), response.getId());
        assertEquals(testUser.getFullName(), response.getFull_name());
        assertEquals(testUser.getEmail(), response.getEmail());
        assertEquals(testUser.getStudentCode(), response.getStudent_code());
        assertEquals(testUser.getRole(), response.getRole());
        assertFalse(response.isAvatar_available());
    }

    @Test
    public void testUpdateProfile_Success() {
        UserProfileUpdateRequest request = new UserProfileUpdateRequest("Nguyen Van B");
        when(userRepository.save(any(User.class))).thenAnswer(inv -> inv.getArgument(0));

        UserProfileResponse response = userProfileService.updateProfile(testUser, request);
        assertEquals("Nguyen Van B", response.getFull_name());
        assertEquals("Nguyen Van B", testUser.getFullName());
    }

    @Test
    public void testUpdateProfile_EmptyName_ShouldThrow() {
        UserProfileUpdateRequest request = new UserProfileUpdateRequest("   ");
        assertThrows(IllegalArgumentException.class, () -> {
            userProfileService.updateProfile(testUser, request);
        });
    }

    @Test
    public void testChangePassword_Success() {
        PasswordChangeRequest request = new PasswordChangeRequest("current_pwd", "NewPassword123");
        when(passwordEncoder.matches("current_pwd", testUser.getPassword())).thenReturn(true);
        when(passwordEncoder.encode("NewPassword123")).thenReturn("hashed_new_password");
        when(userRepository.save(any(User.class))).thenAnswer(inv -> inv.getArgument(0));

        assertNull(testUser.getPasswordChangedAt());
        userProfileService.changePassword(testUser, request);

        assertEquals("hashed_new_password", testUser.getPassword());
        assertNotNull(testUser.getPasswordChangedAt());
        verify(userRepository, times(1)).save(testUser);
    }

    @Test
    public void testChangePassword_IncorrectCurrent_ShouldThrow() {
        PasswordChangeRequest request = new PasswordChangeRequest("wrong_pwd", "NewPassword123");
        when(passwordEncoder.matches("wrong_pwd", testUser.getPassword())).thenReturn(false);

        assertThrows(InvalidPasswordException.class, () -> {
            userProfileService.changePassword(testUser, request);
        });
    }

    @Test
    public void testChangePassword_SamePassword_ShouldThrow() {
        PasswordChangeRequest request = new PasswordChangeRequest("current_pwd", "current_pwd");
        when(passwordEncoder.matches("current_pwd", testUser.getPassword())).thenReturn(true);

        assertThrows(InvalidPasswordException.class, () -> {
            userProfileService.changePassword(testUser, request);
        });
    }

    @Test
    public void testChangePassword_WeakPassword_ShouldThrow() {
        // No digit
        PasswordChangeRequest req1 = new PasswordChangeRequest("current_pwd", "NoDigitPwd");
        // Too short
        PasswordChangeRequest req2 = new PasswordChangeRequest("current_pwd", "Short1");
        
        when(passwordEncoder.matches("current_pwd", testUser.getPassword())).thenReturn(true);

        assertThrows(InvalidPasswordException.class, () -> userProfileService.changePassword(testUser, req1));
        assertThrows(InvalidPasswordException.class, () -> userProfileService.changePassword(testUser, req2));
    }

    @Test
    public void testGetPreferences_DefaultCreatedWhenEmpty() {
        when(userRepository.findById(testUser.getId())).thenReturn(Optional.of(testUser));
        when(userPreferencesRepository.findById(testUser.getId())).thenReturn(Optional.empty());
        when(userPreferencesRepository.save(any(UserPreferences.class))).thenAnswer(inv -> inv.getArgument(0));

        UserPreferencesResponse response = userProfileService.getPreferences(testUser);
        assertEquals(ThemePreference.SYSTEM, response.getTheme());
        assertEquals(FontSizePreference.DEFAULT, response.getFont_size());
        assertFalse(response.isReduce_motion());
        assertEquals(DefaultPagePreference.ASSIGNMENTS, response.getDefault_student_page());
    }

    @Test
    public void testUpdatePreferences_ThemeChange() {
        UserPreferences existing = UserPreferences.builder()
                .userId(testUser.getId())
                .user(testUser)
                .theme(ThemePreference.SYSTEM)
                .fontSize(FontSizePreference.DEFAULT)
                .reduceMotion(false)
                .defaultStudentPage(DefaultPagePreference.ASSIGNMENTS)
                .updatedAt(LocalDateTime.now())
                .build();
        when(userPreferencesRepository.findById(testUser.getId())).thenReturn(Optional.of(existing));
        when(userPreferencesRepository.save(any(UserPreferences.class))).thenAnswer(inv -> inv.getArgument(0));

        UserPreferencesUpdateRequest request = UserPreferencesUpdateRequest.builder()
                .theme(ThemePreference.DARK)
                .build();

        UserPreferencesResponse response = userProfileService.updatePreferences(testUser, request);
        assertEquals(ThemePreference.DARK, response.getTheme());
        assertEquals(ThemePreference.DARK, existing.getTheme());
    }

    @Test
    public void testUploadAvatar_InvalidMagicBytes_ShouldThrow() throws IOException {
        byte[] badBytes = "NotAnImageFile".getBytes();
        MockMultipartFile file = new MockMultipartFile("file", "test.png", "image/png", badBytes);

        assertThrows(UnsupportedMediaTypeException.class, () -> {
            userProfileService.uploadAvatar(testUser, file);
        });
    }

    @Test
    public void testUploadAvatar_Oversized_ShouldThrow() throws IOException {
        byte[] largeBytes = new byte[3 * 1024 * 1024]; // 3MB
        MockMultipartFile file = new MockMultipartFile("file", "large.png", "image/png", largeBytes);

        assertThrows(PayloadTooLargeException.class, () -> {
            userProfileService.uploadAvatar(testUser, file);
        });
    }

    @Test
    public void testUploadAvatar_ValidPng_Success() throws IOException {
        // Create 1x1 PNG image bytes programmatically
        BufferedImage bufferedImage = new BufferedImage(1, 1, BufferedImage.TYPE_INT_RGB);
        ByteArrayOutputStream baos = new ByteArrayOutputStream();
        ImageIO.write(bufferedImage, "png", baos);
        byte[] validPngBytes = baos.toByteArray();

        MockMultipartFile file = new MockMultipartFile("file", "avatar.png", "image/png", validPngBytes);
        when(userRepository.save(any(User.class))).thenAnswer(inv -> inv.getArgument(0));

        assertNull(testUser.getAvatarFilename());
        userProfileService.uploadAvatar(testUser, file);

        assertNotNull(testUser.getAvatarFilename());
        assertTrue(testUser.getAvatarFilename().endsWith(".png"));
        verify(userRepository, times(1)).save(testUser);
        
        // Assert file exists on disk
        Path avatarPath = tempDir.resolve("avatars").resolve(testUser.getAvatarFilename());
        assertTrue(Files.exists(avatarPath));
    }

    @Test
    public void testUploadAvatar_DecompressionBomb_ShouldThrow() throws IOException {
        // Create 2049x2049 PNG image (greater than 2048x2048)
        BufferedImage bombImage = new BufferedImage(2049, 10, BufferedImage.TYPE_INT_RGB);
        ByteArrayOutputStream baos = new ByteArrayOutputStream();
        ImageIO.write(bombImage, "png", baos);
        byte[] bombBytes = baos.toByteArray();

        MockMultipartFile file = new MockMultipartFile("file", "bomb.png", "image/png", bombBytes);

        assertThrows(IllegalArgumentException.class, () -> {
            userProfileService.uploadAvatar(testUser, file);
        });
    }

    @Test
    public void testRemoveAvatar_Success() throws IOException {
        testUser.setAvatarFilename("avatar-to-remove.png");
        Path avatarsDir = tempDir.resolve("avatars");
        Files.createDirectories(avatarsDir);
        Path avatarFile = avatarsDir.resolve("avatar-to-remove.png");
        Files.write(avatarFile, "fake-data".getBytes());

        assertTrue(Files.exists(avatarFile));
        userProfileService.removeAvatar(testUser);

        assertNull(testUser.getAvatarFilename());
        assertFalse(Files.exists(avatarFile));
        verify(userRepository, times(1)).save(testUser);
    }
}
