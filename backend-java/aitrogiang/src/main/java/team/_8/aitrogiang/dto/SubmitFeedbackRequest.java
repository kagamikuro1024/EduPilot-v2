package team._8.aitrogiang.dto;

import jakarta.validation.constraints.NotBlank;
import lombok.Getter;
import lombok.Setter;

@Getter
@Setter
public class SubmitFeedbackRequest {
    @NotBlank(message = "feedback is required")
    private String feedback;
}
