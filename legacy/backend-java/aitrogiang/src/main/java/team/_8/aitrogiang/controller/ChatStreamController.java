package team._8.aitrogiang.controller;

import lombok.RequiredArgsConstructor;
import org.springframework.http.MediaType;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.servlet.mvc.method.annotation.SseEmitter;
import team._8.aitrogiang.service.GrpcChatClientService;

@RestController
@RequestMapping("/api/v1/chat")
@RequiredArgsConstructor
public class ChatStreamController {

    private final GrpcChatClientService grpcChatClientService;

    @GetMapping(value = "/stream", produces = MediaType.TEXT_EVENT_STREAM_VALUE)
    public SseEmitter stream(@RequestParam String message) {
        // TIP-005: Use 60s timeout and implement safety callbacks to prevent resource leaks
        SseEmitter emitter = new SseEmitter(60000L);
        
        emitter.onTimeout(emitter::complete);
        emitter.onCompletion(() -> {}); // Clean up any thread-local resources if needed
        
        grpcChatClientService.streamChat(message, emitter);
        return emitter;
    }
}
