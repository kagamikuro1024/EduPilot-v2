package team._8.aitrogiang.dto;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Size;
import lombok.Getter;
import lombok.NoArgsConstructor;
import lombok.Setter;

/**
 * Request body for {@code POST /api/v1/threads/{thread_id}/messages} (human reply).
 */
@Getter
@Setter
@NoArgsConstructor
public class SendThreadMessageRequest {

    @NotBlank(message = "content must not be blank")
    @Size(max = 50_000, message = "content exceeds maximum length")
    private String content;
}
