package team._8.aitrogiang.controller;

import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import org.springframework.http.CacheControl;
import org.springframework.http.HttpStatus;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.Authentication;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.multipart.MultipartFile;
import team._8.aitrogiang.dto.*;
import team._8.aitrogiang.model.User;
import team._8.aitrogiang.service.UserProfileService;

import java.io.IOException;
import java.util.Map;
import java.util.concurrent.TimeUnit;

@RestController
@RequestMapping("/api/v1/users/me")
@RequiredArgsConstructor
public class UserController {

    private final UserProfileService userProfileService;

    @GetMapping
    public ResponseEntity<UserProfileResponse> getProfile(Authentication authentication) {
        User user = (User) authentication.getPrincipal();
        return ResponseEntity.ok(userProfileService.getProfile(user));
    }

    @PatchMapping
    public ResponseEntity<UserProfileResponse> updateProfile(
            @Valid @RequestBody UserProfileUpdateRequest request,
            Authentication authentication
    ) {
        User user = (User) authentication.getPrincipal();
        return ResponseEntity.ok(userProfileService.updateProfile(user, request));
    }

    @PatchMapping("/student-code")
    public ResponseEntity<UserProfileResponse> setStudentCode(
            @Valid @RequestBody StudentCodeUpdateRequest request,
            Authentication authentication
    ) {
        User user = (User) authentication.getPrincipal();
        return ResponseEntity.ok(userProfileService.setStudentCode(user, request));
    }

    @GetMapping("/avatar")
    public ResponseEntity<byte[]> getAvatar(Authentication authentication) throws IOException {
        User user = (User) authentication.getPrincipal();
        byte[] imageBytes = userProfileService.getAvatar(user);
        
        return ResponseEntity.ok()
                .contentType(MediaType.IMAGE_PNG)
                .cacheControl(CacheControl.maxAge(1, TimeUnit.HOURS).cachePrivate())
                .body(imageBytes);
    }

    @PostMapping(value = "/avatar", consumes = MediaType.MULTIPART_FORM_DATA_VALUE)
    public ResponseEntity<Map<String, String>> uploadAvatar(
            @RequestParam("file") MultipartFile file,
            Authentication authentication
    ) throws IOException {
        User user = (User) authentication.getPrincipal();
        userProfileService.uploadAvatar(user, file);
        return ResponseEntity.ok(Map.of("message", "Avatar uploaded successfully"));
    }

    @DeleteMapping("/avatar")
    public ResponseEntity<Void> removeAvatar(Authentication authentication) {
        User user = (User) authentication.getPrincipal();
        userProfileService.removeAvatar(user);
        return ResponseEntity.noContent().build();
    }

    @PatchMapping("/password")
    public ResponseEntity<Void> changePassword(
            @Valid @RequestBody PasswordChangeRequest request,
            Authentication authentication
    ) {
        User user = (User) authentication.getPrincipal();
        userProfileService.changePassword(user, request);
        return ResponseEntity.noContent().build();
    }

    @GetMapping("/preferences")
    public ResponseEntity<UserPreferencesResponse> getPreferences(Authentication authentication) {
        User user = (User) authentication.getPrincipal();
        return ResponseEntity.ok(userProfileService.getPreferences(user));
    }

    @PatchMapping("/preferences")
    public ResponseEntity<UserPreferencesResponse> updatePreferences(
            @RequestBody UserPreferencesUpdateRequest request,
            Authentication authentication
    ) {
        User user = (User) authentication.getPrincipal();
        return ResponseEntity.ok(userProfileService.updatePreferences(user, request));
    }
}
