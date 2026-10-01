package team._8.aitrogiang.dto;

import lombok.Getter;
import lombok.NoArgsConstructor;
import lombok.Setter;

@Getter
@Setter
@NoArgsConstructor
public class ClassifyIntentRequestDTO {
    private String content;
    private String channelHint;
}
