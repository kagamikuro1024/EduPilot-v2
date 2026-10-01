package team._8.aitrogiang.service;

import io.grpc.Status;
import io.grpc.StatusRuntimeException;
import io.grpc.stub.StreamObserver;
import lombok.extern.slf4j.Slf4j;
import net.devh.boot.grpc.client.inject.GrpcClient;
import org.springframework.stereotype.Service;
import org.springframework.web.servlet.mvc.method.annotation.SseEmitter;
import team._8.aitrogiang.grpc.AIRequest;
import team._8.aitrogiang.grpc.AIResponse;
import team._8.aitrogiang.grpc.AIThreadServiceGrpc;

import java.io.IOException;
import java.util.UUID;

@Slf4j
@Service
public class GrpcChatClientService {

    @GrpcClient("ai-service")
    private AIThreadServiceGrpc.AIThreadServiceStub aiThreadServiceStub;

    /**
     * Streams AI response tokens from Python gRPC service to a Spring SseEmitter.
     * Implements Graceful Fallback for security guardrail violations.
     */
    public void streamChat(String message, SseEmitter emitter) {
        AIRequest request = AIRequest.newBuilder()
                .setCurrentMessage(message)
                .setThreadId(UUID.randomUUID().toString()) // TIP-005: Use dynamic ID to prevent context collision
                .setThreadTitle("General SSE Chat")
                .build();

        aiThreadServiceStub.streamAIResponse(request, new StreamObserver<AIResponse>() {
            @Override
            public void onNext(AIResponse response) {
                try {
                    if (response.getChunk() != null && !response.getChunk().isEmpty()) {
                        emitter.send(response.getChunk());
                    }
                } catch (IOException e) {
                    log.error("Failed to send SSE token, releasing channel: {}", e.getMessage());
                    emitter.completeWithError(e); // TIP-005: Mandatory release of resources
                }
            }

            @Override
            public void onError(Throwable t) {
                if (t instanceof StatusRuntimeException statusException) {
                    if (statusException.getStatus().getCode() == Status.Code.FAILED_PRECONDITION) {
                        log.warn("Guardrail violation detected by Python service. Triggering graceful fallback.");
                        try {
                            // GRACEFUL FALLBACK: Send polite refusal instead of error
                            emitter.send("Vấn đề này vượt ngoài phạm vi trả lời của tôi, thông tin sẽ được chuyển cho TA, bạn chú ý theo dõi nhé.");
                            emitter.complete();
                        } catch (IOException e) {
                            log.error("Failed to send fallback message: {}", e.getMessage());
                            emitter.completeWithError(e);
                        }
                        return;
                    }
                }
                
                log.error("gRPC stream error: {}", t.getMessage());
                emitter.completeWithError(t);
            }

            @Override
            public void onCompleted() {
                log.info("gRPC stream completed successfully.");
                emitter.complete();
            }
        });
    }
}
