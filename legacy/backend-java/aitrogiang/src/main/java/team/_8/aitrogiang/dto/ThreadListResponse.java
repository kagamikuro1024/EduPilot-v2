package team._8.aitrogiang.dto;

import lombok.Builder;
import lombok.Value;

import java.util.List;

@Value
@Builder
public class ThreadListResponse {
    List<ThreadItemDTO> data;
    Meta meta;

    @Value
    @Builder
    public static class Meta {
        long total;
        int page;
    }
}
