package team._8.aitrogiang.dto;

import lombok.*;
import team._8.aitrogiang.model.UserRole;
import java.time.LocalDateTime;
import java.util.UUID;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class UserProfileResponse {
    private UUID id;
    private String full_name;
    private String email;
    private String student_code;
    private UserRole role;
    private boolean avatar_available;
    private LocalDateTime created_at;
}
