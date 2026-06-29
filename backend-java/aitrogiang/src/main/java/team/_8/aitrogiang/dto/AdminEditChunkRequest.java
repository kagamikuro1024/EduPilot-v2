package team._8.aitrogiang.dto;

import jakarta.validation.constraints.NotBlank;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class AdminEditChunkRequest {
    @NotBlank(message = "Chunk ID is required")
    private String chunk_id;

    @NotBlank(message = "New content is required")
    private String new_content;
}
