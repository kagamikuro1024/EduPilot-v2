package team._8.aitrogiang.dto;

import lombok.Builder;
import lombok.Value;

import java.util.List;
import java.util.Map;

@Value
@Builder
public class ChatMessageDTO {
    String id;
    AuthorDTO author;
    String content;
    String feedback;
    String created_at;
    List<Map<String, Object>> citations;
}
