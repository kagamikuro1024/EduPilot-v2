package team._8.aitrogiang.dto;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Pattern;
import jakarta.validation.constraints.Size;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class StudentCodeUpdateRequest {
    @NotBlank(message = "Student code is required")
    @Size(min = 4, max = 50, message = "Student code must be between 4 and 50 characters")
    @Pattern(regexp = "[A-Za-z0-9_-]+", message = "Student code contains invalid characters")
    private String student_code;
}
