package team._8.aitrogiang.dto;

import jakarta.validation.constraints.NotBlank;
import lombok.Getter;
import lombok.NoArgsConstructor;
import lombok.Setter;

import java.util.List;

@Getter
@Setter
@NoArgsConstructor
public class CreateThreadRequest {
    @NotBlank
    private String title;
    @NotBlank
    private String content;
    private List<String> tags;
}
