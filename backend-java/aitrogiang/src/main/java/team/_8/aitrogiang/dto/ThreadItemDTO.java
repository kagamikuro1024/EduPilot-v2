package team._8.aitrogiang.dto;

import lombok.Builder;
import lombok.Value;

import java.util.List;

@Value
@Builder
public class ThreadItemDTO {
    String id;
    String title;
    AuthorDTO author;
    Long reply_count;
    String status;
    List<String> tags;
    String last_message_preview;
    String updated_at;
}
