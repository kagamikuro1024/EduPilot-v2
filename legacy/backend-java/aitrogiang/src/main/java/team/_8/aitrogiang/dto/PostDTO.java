package team._8.aitrogiang.dto;

import lombok.Builder;
import lombok.Value;

import java.util.List;
import java.util.Map;

@Value
@Builder
public class PostDTO {
    String id;
    AuthorDTO author;
    String content;
    String original_ai_content;
    String verification_status;
    AuthorDTO verified_by_ta;
    String created_at;
    Map<String, Integer> reactions;
    Boolean is_accepted;
    List<Map<String, Object>> citations;
}
