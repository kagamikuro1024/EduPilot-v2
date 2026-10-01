package team._8.aitrogiang.dto;

import jakarta.validation.constraints.NotBlank;
import lombok.Getter;
import lombok.Setter;

@Getter
@Setter
public class CorrectMessageRequest {
    @NotBlank(message = "content is required")
    private String content;
}
