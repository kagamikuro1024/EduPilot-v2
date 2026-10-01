package team._8.aitrogiang.service;

import io.grpc.Status;
import io.grpc.StatusRuntimeException;
import io.grpc.stub.StreamObserver;
import lombok.extern.slf4j.Slf4j;
import net.devh.boot.grpc.client.inject.GrpcClient;
import org.springframework.stereotype.Service;
import org.springframework.beans.factory.annotation.Value;
import reactor.core.publisher.Flux;
import reactor.core.publisher.Sinks;
import team._8.aitrogiang.grpc.AIDocumentServiceGrpc;
import team._8.aitrogiang.grpc.AIRequest;
import team._8.aitrogiang.grpc.AIResponse;
import team._8.aitrogiang.grpc.AIThreadServiceGrpc;
import team._8.aitrogiang.grpc.ClassifyRequest;
import team._8.aitrogiang.grpc.ClassifyResponse;
import team._8.aitrogiang.grpc.DocumentRequest;
import team._8.aitrogiang.grpc.IntentType;
import team._8.aitrogiang.grpc.Message;
import team._8.aitrogiang.util.JavaPreflightTagBuilder;
import team._8.aitrogiang.util.AuthenticatedUserTagBuilder;

import java.util.List;
import java.util.concurrent.TimeUnit;

/**
 * ARCHITECTURAL BOUNDARY: This is the ONLY component in Java that communicates
 * with the Python AI Service. All other Java classes are strictly FORBIDDEN
 * from calling Python directly.
 *
 * Responsibility: Open a gRPC stream to Python, return raw AIResponse Flux.
 * This class has NO business logic. It only handles transport.
 */
@Slf4j
@Service
public class PythonAiOrchestratorClient {

    private static final long CLASSIFY_DEADLINE_SEC = 15L;

    @Value("${app.internal.token:}")
    private String internalToken;

    @GrpcClient("ai-service")
    private AIThreadServiceGrpc.AIThreadServiceStub aiThreadServiceStub;

    @GrpcClient("ai-service")
    private AIThreadServiceGrpc.AIThreadServiceBlockingStub aiThreadServiceBlockingStub;

    @GrpcClient("ai-service")
    private AIDocumentServiceGrpc.AIDocumentServiceBlockingStub documentServiceStub;

    /**
     * Synchronously classifies the user's intent and checks for confidence/violations.
     * Used for graceful fallback logic (TD-05).
     */
    public ClassifyResponse classifyIntent(String content, String channelHint) {
        ClassifyRequest request = ClassifyRequest.newBuilder()
                .setContent(content != null ? content : "")
                .setChannelHint(channelHint != null ? channelHint : "PUBLIC")
                .build();
        try {
            return aiThreadServiceBlockingStub
                    .withDeadlineAfter(CLASSIFY_DEADLINE_SEC, TimeUnit.SECONDS)
                    .classifyIntent(request);
        } catch (StatusRuntimeException e) {
            if (e.getStatus().getCode() == Status.Code.DEADLINE_EXCEEDED) {
                log.warn("[gRPC] classifyIntent deadline exceeded after {}s", CLASSIFY_DEADLINE_SEC);
            } else {
                log.warn("[gRPC] classifyIntent failed: {}", e.getStatus());
            }
            return ClassifyResponse.newBuilder()
                    .setSuggestedChannel(IntentType.UNCERTAIN)
                    .setConfidence(0f)
                    .setReasoning("Classifier temporarily unavailable")
                    .setIsViolation(false)
                    .build();
        }
    }

    /**
     * Opens a server-streaming gRPC call to the Python AI service.
     *
     * @param threadId      The ID of the forum thread for context
     * @param threadTitle   Human-readable title sent to AI for better context grounding
     * @param currentMessage The user's current input message
     * @param history       Full conversation history (built by Java from DB — Python is stateless)
     * @return A hot Flux of raw AIResponse proto messages. Errors are propagated as Flux errors.
     */
    public Flux<AIResponse> streamResponse(
            String contextId,
            String threadTitle,
            String currentMessage,
            List<Message> history,
            String channelHint,
            ClassifyResponse javaPreflight,
            String authenticatedUserId,
            String authenticatedStudentCode,
            String authenticatedRole
    ) {
        AIRequest.Builder req = AIRequest.newBuilder()
                .setThreadId(contextId)
                .setThreadTitle(threadTitle)
                .setCurrentMessage(currentMessage)
                .addAllHistory(history)
                .addTags("channel:" + channelHint);
        String preflightTag = JavaPreflightTagBuilder.buildTag(javaPreflight);
        if (preflightTag != null) {
            req.addTags(preflightTag);
        }
        String authenticatedUserTag = AuthenticatedUserTagBuilder.buildTag(
                authenticatedUserId,
                authenticatedStudentCode,
                authenticatedRole,
                internalToken
        );
        if (authenticatedUserTag != null) {
            req.addTags(authenticatedUserTag);
        }
        AIRequest request = req.build();

        // Unicast sink: single subscriber (ChatStreamingService), backpressure buffered.
        Sinks.Many<AIResponse> sink = Sinks.many().unicast().onBackpressureBuffer();

        log.info("[gRPC → Python] Streaming request: thread={}, message_len={}",
                contextId, currentMessage.length());

        aiThreadServiceStub.streamAIResponse(request, new StreamObserver<>() {

            @Override
            public void onNext(AIResponse value) {
                Sinks.EmitResult result = sink.tryEmitNext(value);
                if (result.isFailure()) {
                    log.warn("[gRPC] Failed to emit chunk to sink: {}", result);
                }
            }

            @Override
            public void onError(Throwable t) {
                log.error("[gRPC → Python] Stream error: {}", t.getMessage());
                sink.tryEmitError(t);
            }

            @Override
            public void onCompleted() {
                log.info("[gRPC → Python] Stream completed for thread={}", contextId);
                sink.tryEmitComplete();
            }
        });

        return sink.asFlux();
    }

    public boolean processDocument(String documentId, String fileUrl) {
        try {
            DocumentRequest req = DocumentRequest.newBuilder()
                    .setDocumentId(documentId)
                    .setFileUrl(fileUrl)
                    .build();
            return documentServiceStub
                    .withDeadlineAfter(15, TimeUnit.SECONDS)
                    .processDocument(req)
                    .getAccepted();
        } catch (StatusRuntimeException e) {
            log.warn("[gRPC] ProcessDocument failed: {}", e.getStatus());
            return false;
        }
    }

    public boolean updateChunkContent(String chunkId, String newContent, String updatedBy) {
        try {
            team._8.aitrogiang.grpc.UpdateChunkRequest req = team._8.aitrogiang.grpc.UpdateChunkRequest.newBuilder()
                    .setChunkId(chunkId)
                    .setNewContent(newContent)
                    .setUpdatedBy(updatedBy)
                    .build();
            return documentServiceStub
                    .withDeadlineAfter(20, TimeUnit.SECONDS)
                    .updateChunkContent(req)
                    .getSuccess();
        } catch (StatusRuntimeException e) {
            log.warn("[gRPC] updateChunkContent failed: {}", e.getStatus());
            return false;
        }
    }
}
