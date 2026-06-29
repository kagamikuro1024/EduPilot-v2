package team._8.aitrogiang.dto;

import lombok.AllArgsConstructor;
import lombok.Getter;
import lombok.NoArgsConstructor;

@Getter
@NoArgsConstructor
@AllArgsConstructor
public class DocumentStatusCallbackRequest {
    private String status;
    private String reason;
}
