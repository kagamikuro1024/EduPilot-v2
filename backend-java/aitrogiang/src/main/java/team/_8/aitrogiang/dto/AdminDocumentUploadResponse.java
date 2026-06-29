package team._8.aitrogiang.dto;

import lombok.Builder;
import lombok.Value;

@Value
@Builder
public class AdminDocumentUploadResponse {
    String document_id;
    String status;
}
