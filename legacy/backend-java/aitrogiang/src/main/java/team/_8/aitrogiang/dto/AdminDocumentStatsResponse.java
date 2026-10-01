package team._8.aitrogiang.dto;

import lombok.Builder;
import lombok.Value;

@Value
@Builder
public class AdminDocumentStatsResponse {
    long total_documents;
    String index_health;
    String knowledge_volume;
}
