package team._8.aitrogiang.service;

import io.grpc.Status;
import io.grpc.StatusRuntimeException;
import lombok.extern.slf4j.Slf4j;
import net.devh.boot.grpc.client.inject.GrpcClient;
import org.springframework.stereotype.Service;
import team._8.aitrogiang.grpc.AIThreadServiceGrpc;
import team._8.aitrogiang.grpc.ClassifyRequest;
import team._8.aitrogiang.grpc.ClassifyResponse;
import team._8.aitrogiang.grpc.IntentType;

import java.util.concurrent.TimeUnit;

@Slf4j
@Service
public class IntentClassifierClient {

    private static final long CLASSIFIER_DEADLINE_MS = 3000;

    @GrpcClient("ai-service")
    private AIThreadServiceGrpc.AIThreadServiceBlockingStub stub;

    public ClassifyResponse classify(String content, String channelHint) {
        ClassifyRequest req = ClassifyRequest.newBuilder()
                .setContent(content == null ? "" : content)
                .setChannelHint(channelHint == null ? "UNCERTAIN" : channelHint)
                .build();
        try {
            return stub.withDeadlineAfter(CLASSIFIER_DEADLINE_MS, TimeUnit.MILLISECONDS)
                    .classifyIntent(req);
        } catch (StatusRuntimeException e) {
            Status.Code code = e.getStatus().getCode();
            if (code == Status.Code.DEADLINE_EXCEEDED) {
                log.warn("[Classifier] Timeout after {}ms, fallback UNCERTAIN", CLASSIFIER_DEADLINE_MS);
            } else if (code == Status.Code.UNAVAILABLE) {
                log.warn("[Classifier] Service unavailable, fallback UNCERTAIN");
            } else {
                log.error("[Classifier] gRPC error {}: {}", code, e.getMessage());
            }
            return fallbackResponse();
        } catch (Exception e) {
            log.error("[Classifier] Unexpected error: {}", e.getMessage());
            return fallbackResponse();
        }
    }

    private ClassifyResponse fallbackResponse() {
        return ClassifyResponse.newBuilder()
                .setSuggestedChannel(IntentType.UNCERTAIN)
                .setConfidence(0f)
                .setReasoning("Classifier temporarily unavailable")
                .setIsViolation(false)
                .build();
    }
}
