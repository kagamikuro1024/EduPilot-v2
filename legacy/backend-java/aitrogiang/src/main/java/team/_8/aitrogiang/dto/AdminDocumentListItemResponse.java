package team._8.aitrogiang.dto;

import lombok.Builder;
import lombok.Value;

@Value
@Builder
public class AdminDocumentListItemResponse {
    String document_id;
    String name;
    String status;
    String document_type;
    Long file_size_bytes;
    String created_at;
}
