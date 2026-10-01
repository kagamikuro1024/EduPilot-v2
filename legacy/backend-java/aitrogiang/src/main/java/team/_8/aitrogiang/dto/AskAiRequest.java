package team._8.aitrogiang.dto;

import lombok.Getter;
import lombok.NoArgsConstructor;
import lombok.Setter;

@Getter
@Setter
@NoArgsConstructor
public class AskAiRequest {

    /**
     * The user's message content.
     * Can be empty string if this is the first call right after thread creation
     * (Frontend calls ask-ai immediately after POST /threads, body is empty).
     */
    private String message = "";

    /**
     * True when Frontend triggers the first AI answer immediately after creating
     * a thread. The original question is already persisted by POST /threads.
     */
    private boolean autoTriggered = false;
}
