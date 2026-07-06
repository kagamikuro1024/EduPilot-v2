package team._8.aitrogiang.service;

import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.web.multipart.MultipartFile;
import team._8.aitrogiang.dto.*;
import team._8.aitrogiang.exception.*;
import team._8.aitrogiang.model.*;
import team._8.aitrogiang.repository.UserPreferencesRepository;
import team._8.aitrogiang.repository.UserRepository;

import javax.imageio.ImageIO;
import java.awt.Graphics2D;
import java.awt.image.BufferedImage;
import java.io.ByteArrayInputStream;
import java.io.File;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.time.LocalDateTime;
import java.util.UUID;

@Service
@RequiredArgsConstructor
@Slf4j
@Transactional
public class UserProfileService {

    private final UserRepository userRepository;
    private final UserPreferencesRepository userPreferencesRepository;
    private final PasswordEncoder passwordEncoder;

    @Value("${app.upload.dir:./uploads}")
    private String uploadDir;

    public UserProfileResponse getProfile(User user) {
        return UserProfileResponse.builder()
                .id(user.getId())
                .full_name(user.getFullName())
                .email(user.getEmail())
                .student_code(user.getStudentCode())
                .role(user.getRole())
                .avatar_available(user.getAvatarFilename() != null)
                .created_at(user.getCreatedAt())
                .build();
    }

    public UserProfileResponse updateProfile(User user, UserProfileUpdateRequest request) {
        String trimmed = request.getFull_name() != null ? request.getFull_name().trim() : null;
        if (trimmed == null || trimmed.isEmpty()) {
            throw new IllegalArgumentException("Full name cannot be empty");
        }
        user.setFullName(trimmed);
        User saved = userRepository.save(user);
        return getProfile(saved);
    }

    /**
     * Bind a student code once. It becomes immutable through the student API
     * because this identifier controls access to private grade records.
     */
    public UserProfileResponse setStudentCode(User user, StudentCodeUpdateRequest request) {
        if (user.getRole() != UserRole.STUDENT) {
            throw new IllegalArgumentException("Only student accounts can set a student code");
        }
        String normalized = request.getStudent_code().trim().toUpperCase();
        if (user.getStudentCode() != null && !user.getStudentCode().isBlank()) {
            if (user.getStudentCode().equalsIgnoreCase(normalized)) {
                return getProfile(user);
            }
            throw new IllegalArgumentException("Student code is already set and cannot be changed");
        }
        if (userRepository.existsByStudentCodeIgnoreCase(normalized)) {
            throw new IllegalArgumentException("Student code is already linked to another account");
        }
        user.setStudentCode(normalized);
        return getProfile(userRepository.save(user));
    }

    public void changePassword(User user, PasswordChangeRequest request) {
        if (!passwordEncoder.matches(request.getCurrent_password(), user.getPassword())) {
            throw new InvalidPasswordException("Mật khẩu hiện tại không chính xác");
        }

        String newPassword = request.getNew_password();
        if (newPassword.equals(request.getCurrent_password())) {
            throw new InvalidPasswordException("Mật khẩu mới không được trùng với mật khẩu hiện tại");
        }

        validatePasswordStrength(newPassword);

        // Password change and password_changed_at update must happen in one transaction (truncated to second precision)
        user.setPassword(passwordEncoder.encode(newPassword));
        user.setPasswordChangedAt(LocalDateTime.now().withNano(0));
        userRepository.save(user);
    }

    public UserPreferencesResponse getPreferences(User user) {
        UserPreferences prefs = userPreferencesRepository.findById(user.getId())
                .orElseGet(() -> createDefaultPreferences(user));
        return mapPreferencesToResponse(prefs);
    }

    public UserPreferencesResponse updatePreferences(User user, UserPreferencesUpdateRequest request) {
        UserPreferences prefs = userPreferencesRepository.findById(user.getId())
                .orElseGet(() -> createDefaultPreferences(user));

        if (request.getTheme() != null) {
            prefs.setTheme(request.getTheme());
        }
        if (request.getFont_size() != null) {
            prefs.setFontSize(request.getFont_size());
        }
        if (request.getReduce_motion() != null) {
            prefs.setReduceMotion(request.getReduce_motion());
        }
        if (request.getDefault_student_page() != null) {
            prefs.setDefaultStudentPage(request.getDefault_student_page());
        }

        // PreUpdate handles updatedAt, but we set it explicitly as required for full guarantee
        prefs.setUpdatedAt(LocalDateTime.now());
        UserPreferences saved = userPreferencesRepository.save(prefs);
        return mapPreferencesToResponse(saved);
    }

    public byte[] getAvatar(User user) throws IOException {
        if (user.getAvatarFilename() == null) {
            throw new ResourceNotFoundException("No avatar configured for user");
        }

        Path avatarsDir = getAvatarsDirectory();
        Path avatarPath = avatarsDir.resolve(user.getAvatarFilename());
        
        if (!Files.exists(avatarPath)) {
            // Return 404 cleanly when database references a missing file
            throw new ResourceNotFoundException("Avatar file not found on disk");
        }

        return Files.readAllBytes(avatarPath);
    }

    public void uploadAvatar(User user, MultipartFile file) throws IOException {
        if (file.isEmpty()) {
            throw new IllegalArgumentException("File upload must not be empty");
        }

        // Limit size <= 2MB
        if (file.getSize() > 2 * 1024 * 1024) {
            throw new PayloadTooLargeException("Avatar file size must not exceed 2 MB");
        }

        byte[] fileBytes = file.getBytes();

        // Magic bytes check
        if (!isJpeg(fileBytes) && !isPng(fileBytes) && !isWebp(fileBytes)) {
            throw new UnsupportedMediaTypeException("Unsupported avatar file type. Only JPEG, PNG, and WebP are allowed.");
        }

        // Animation check
        if (isAnimatedPng(fileBytes) || isAnimatedWebp(fileBytes)) {
            throw new IllegalArgumentException("Animated avatars are not allowed");
        }

        // Decode image using ImageIO (WebP read support is enabled by webp-imageio on classpath)
        BufferedImage srcImage;
        try (ByteArrayInputStream bais = new ByteArrayInputStream(fileBytes)) {
            srcImage = ImageIO.read(bais);
        } catch (Exception e) {
            throw new IllegalArgumentException("Malformed image file", e);
        }

        if (srcImage == null) {
            throw new IllegalArgumentException("Malformed or unreadable image file");
        }

        // Decompression bomb check: width/height <= 2048px
        if (srcImage.getWidth() > 2048 || srcImage.getHeight() > 2048) {
            throw new IllegalArgumentException("Avatar dimensions must not exceed 2048 x 2048 pixels");
        }

        // Normalize image by drawing it onto a new BufferedImage (this strips metadata)
        BufferedImage normalizedImage = new BufferedImage(
                srcImage.getWidth(), srcImage.getHeight(), BufferedImage.TYPE_INT_ARGB);
        Graphics2D g = normalizedImage.createGraphics();
        try {
            g.drawImage(srcImage, 0, 0, null);
        } finally {
            g.dispose();
        }

        Path avatarsDir = getAvatarsDirectory();
        Files.createDirectories(avatarsDir);

        // Generate unique server-side filename (UUID + .png since we normalize to PNG format)
        String newFilename = UUID.randomUUID().toString() + ".png";
        Path tempFilePath = avatarsDir.resolve(newFilename + ".tmp");
        Path finalFilePath = avatarsDir.resolve(newFilename);

        // Write the new avatar safely to a temporary file
        try {
            boolean written = ImageIO.write(normalizedImage, "png", tempFilePath.toFile());
            if (!written) {
                throw new IllegalStateException("PNG Image writer not found");
            }
        } catch (IOException e) {
            throw new IllegalStateException("Failed to write avatar to disk", e);
        }

        String oldFilename = user.getAvatarFilename();

        try {
            // Update database only after new file is validly written
            user.setAvatarFilename(newFilename);
            userRepository.save(user);

            // Rename temporary file to final path
            Files.move(tempFilePath, finalFilePath);

            // Delete old file only after replacement succeeds
            if (oldFilename != null) {
                try {
                    Files.deleteIfExists(avatarsDir.resolve(oldFilename));
                } catch (IOException e) {
                    log.warn("Failed to delete old avatar file: {}", oldFilename, e);
                }
            }
        } catch (Exception e) {
            // Delete new temp file if db transaction fails
            try {
                Files.deleteIfExists(tempFilePath);
                Files.deleteIfExists(finalFilePath);
            } catch (IOException ioex) {
                log.error("Failed to clean up temp avatar files on failure", ioex);
            }
            throw new IllegalStateException("Database update failed, rolling back avatar upload", e);
        }
    }

    public void removeAvatar(User user) {
        String avatarFilename = user.getAvatarFilename();
        if (avatarFilename == null) {
            throw new ResourceNotFoundException("No avatar configured to remove");
        }

        user.setAvatarFilename(null);
        userRepository.save(user);

        try {
            Path avatarsDir = getAvatarsDirectory();
            Files.deleteIfExists(avatarsDir.resolve(avatarFilename));
        } catch (IOException e) {
            log.warn("Failed to delete avatar file from disk during removal: {}", avatarFilename, e);
            // Handle missing old files without failing the whole request
        }
    }

    private Path getAvatarsDirectory() {
        return Paths.get(uploadDir).resolve("avatars").toAbsolutePath().normalize();
    }

    private UserPreferences createDefaultPreferences(User user) {
        User managedUser = userRepository.findById(user.getId())
                .orElseThrow(() -> new team._8.aitrogiang.exception.ResourceNotFoundException("User not found: " + user.getId()));
        UserPreferences prefs = UserPreferences.builder()
                .user(managedUser)
                .theme(ThemePreference.SYSTEM)
                .fontSize(FontSizePreference.DEFAULT)
                .reduceMotion(false)
                .defaultStudentPage(DefaultPagePreference.ASSIGNMENTS)
                .updatedAt(LocalDateTime.now())
                .build();
        return userPreferencesRepository.save(prefs);
    }

    private UserPreferencesResponse mapPreferencesToResponse(UserPreferences prefs) {
        return UserPreferencesResponse.builder()
                .theme(prefs.getTheme())
                .font_size(prefs.getFontSize())
                .reduce_motion(prefs.isReduceMotion())
                .default_student_page(prefs.getDefaultStudentPage())
                .build();
    }

    private void validatePasswordStrength(String password) {
        if (password.length() < 8 || password.length() > 72) {
            throw new InvalidPasswordException("Mật khẩu phải từ 8 đến 72 ký tự");
        }
        boolean hasUpper = false;
        boolean hasLower = false;
        boolean hasDigit = false;
        for (char c : password.toCharArray()) {
            if (Character.isUpperCase(c)) hasUpper = true;
            else if (Character.isLowerCase(c)) hasLower = true;
            else if (Character.isDigit(c)) hasDigit = true;
        }
        if (!hasUpper || !hasLower || !hasDigit) {
            throw new InvalidPasswordException("Mật khẩu phải chứa ít nhất một chữ hoa, một chữ thường và một chữ số");
        }
    }

    private boolean isJpeg(byte[] bytes) {
        if (bytes.length < 3) return false;
        return (bytes[0] & 0xFF) == 0xFF && (bytes[1] & 0xFF) == 0xD8 && (bytes[2] & 0xFF) == 0xFF;
    }

    private boolean isPng(byte[] bytes) {
        if (bytes.length < 8) return false;
        return (bytes[0] & 0xFF) == 0x89 && (bytes[1] & 0xFF) == 0x50 &&
                (bytes[2] & 0xFF) == 0x4E && (bytes[3] & 0xFF) == 0x47 &&
                (bytes[4] & 0xFF) == 0x0D && (bytes[5] & 0xFF) == 0x0A &&
                (bytes[6] & 0xFF) == 0x1A && (bytes[7] & 0xFF) == 0x0A;
    }

    private boolean isWebp(byte[] bytes) {
        if (bytes.length < 12) return false;
        boolean riff = (bytes[0] & 0xFF) == 0x52 && (bytes[1] & 0xFF) == 0x49 &&
                (bytes[2] & 0xFF) == 0x46 && (bytes[3] & 0xFF) == 0x46;
        boolean webp = (bytes[8] & 0xFF) == 0x57 && (bytes[9] & 0xFF) == 0x45 &&
                (bytes[10] & 0xFF) == 0x42 && (bytes[11] & 0xFF) == 0x50;
        return riff && webp;
    }

    private boolean isAnimatedPng(byte[] bytes) {
        return indexOf(bytes, new byte[]{(byte)'a', (byte)'c', (byte)'T', (byte)'L'}) != -1;
    }

    private boolean isAnimatedWebp(byte[] bytes) {
        return indexOf(bytes, new byte[]{(byte)'A', (byte)'N', (byte)'I', (byte)'M'}) != -1;
    }

    private int indexOf(byte[] array, byte[] target) {
        if (target.length == 0) return 0;
        outer:
        for (int i = 0; i < array.length - target.length + 1; i++) {
            for (int j = 0; j < target.length; j++) {
                if (array[i + j] != target[j]) {
                    continue outer;
                }
            }
            return i;
        }
        return -1;
    }
}
