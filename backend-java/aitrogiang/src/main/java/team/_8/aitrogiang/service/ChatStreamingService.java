package team._8.aitrogiang.service;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import io.grpc.Metadata;
import io.grpc.Status;
import io.grpc.StatusRuntimeException;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import team._8.aitrogiang.constant.SystemConstants;
import team._8.aitrogiang.config.ChatProperties;
import org.springframework.http.codec.ServerSentEvent;
import reactor.core.publisher.Flux;
import reactor.core.publisher.Mono;
import reactor.core.publisher.SignalType;
import reactor.core.scheduler.Schedulers;
import team._8.aitrogiang.grpc.Message;
import team._8.aitrogiang.grpc.ResponseMetadata;
import team._8.aitrogiang.exception.IntentViolationException;
import team._8.aitrogiang.grpc.ClassifyResponse;
import team._8.aitrogiang.model.*;
import team._8.aitrogiang.repository.ChatMessageRepository;
import team._8.aitrogiang.repository.ChatSessionRepository;
import team._8.aitrogiang.repository.ForumPostRepository;
import team._8.aitrogiang.repository.ForumThreadRepository;
import team._8.aitrogiang.repository.UserRepository;

import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.UUID;
import java.util.stream.Collectors;

/**
 * ARCHITECTURAL ROLE: Java is the STATE MANAGER.
 *
 * This service enforces the 3-step contract for every AI interaction:
 *   1. [PRE-STREAM]  Save the user's message to DB (source of truth).
 *   2. [STREAM]      Relay the gRPC stream to the Frontend as SSE chunks.
 *   3. [POST-STREAM] Aggregate and save the complete AI response to DB.
 */
@Slf4j
@Service
@RequiredArgsConstructor
public class ChatStreamingService {

    /** Cap gRPC history at five question–answer exchanges (10 messages when alternating). */
    private static final int MAX_HISTORY_MESSAGES = 10;

    private final PythonAiOrchestratorClient pythonClient;
    private final ChatSessionRepository sessionRepository;
    private final ChatMessageRepository chatMessageRepository;
    private final ForumPostRepository forumPostRepository;
    private final ForumThreadRepository forumThreadRepository;
    private final UserRepository userRepository;
    private final ObjectMapper objectMapper;
    private final ChatProperties chatProperties;

    /**
     * Main entry point for the SSE streaming gateway.
     * Enforces non-blocking execution and short-circuit logic for poor AI confidence.
     */
    public Flux<ServerSentEvent<Object>> processAndStream(UUID contextId, String userMessage, User currentUser, ChatChannel channel) {
        return processAndStream(contextId, userMessage, currentUser, channel, true);
    }

    public Flux<ServerSentEvent<Object>> processAndStream(UUID contextId, String userMessage, User currentUser, ChatChannel channel, boolean persistUserMessage) {
        
        // --- 1. Pre-Stream: Save User Message (Atomic first step) ---
        if (persistUserMessage) {
            saveUserMessage(contextId, userMessage, currentUser, channel);
        }

        // --- 2. Async Classification & Short-circuit (TD-02/TD-05/TIP-006) ---
        // We use boundedElastic to prevent blocking the Netty event loop (Scenario 1)
        return Mono.fromCallable(() -> pythonClient.classifyIntent(userMessage, channel.name()))
                .subscribeOn(Schedulers.boundedElastic())
                .flatMapMany(classification -> {
                    
                    // --- 3. Short-circuit Escalate (Scenario 2) ---
                    if (classification.getConfidence() < chatProperties.getConfidenceThreshold()) {
                        log.info("[Fallback] Low confidence ({}) for message in context {}. Prompting user to manually escalate.", 
                                 classification.getConfidence(), contextId);
                        
                        // Return a single message informing the user and complete the stream
                        return Flux.just(formatChunk("🔍 Trợ giảng AI nhận thấy câu hỏi này khá phức tạp. " +
                                "Nếu bạn cần hỗ trợ, vui lòng bấm nút 'Needs TA' để gửi câu hỏi này cho Mentor/TA.", true, null));
                    }

                    // --- 4. Proceed with Normal AI Stream ---
                    List<Message> grpcHistory = buildHistory(contextId, currentUser, channel);
                    StringBuilder aiResponseAccumulator = new StringBuilder();
                    // Accumulate citations JSON to persist alongside the AI response
                    final String[] citationsJsonHolder = {null};

                    return pythonClient.streamResponse(
                            contextId.toString(),
                            channel == ChatChannel.FORUM ? "Thread #" + contextId : "Private chat session",
                            userMessage,
                            grpcHistory,
                            channel.name(),
                            classification,
                            currentUser.getId().toString(),
                            currentUser.getStudentCode(),
                            currentUser.getRole().name()
                    )
                    .flatMap(aiResponse -> {
                        String chunk = aiResponse.getChunk();
                        boolean isFinished = aiResponse.getIsFinished();

                        if (chunk != null && !chunk.isEmpty()) {
                            aiResponseAccumulator.append(chunk);
                            return Mono.just(formatChunk(chunk, false, null));
                        }
                        if (isFinished) {
                            ResponseMetadata metadata = aiResponse.hasMetadata() ? aiResponse.getMetadata() : null;
                            // Capture citations JSON for DB persistence
                            if (metadata != null && metadata.getCitationsCount() > 0) {
                                try {
                                    List<Map<String, Object>> citationsList = metadata.getCitationsList().stream()
                                            .map(c -> Map.<String, Object>of(
                                                    "source_file", c.getSourceFile(),
                                                    "page_number", c.getPageNumber(),
                                                    "document_id", c.getDocumentId(),
                                                    "source_uri", c.getSourceUri(),
                                                    "chunk_id", c.getChunkId(),
                                                    "chunk_index", c.getChunkIndex(),
                                                    "snippet", c.getSnippet()
                                            ))
                                            .collect(Collectors.toList());
                                    citationsJsonHolder[0] = objectMapper.writeValueAsString(citationsList);
                                } catch (JsonProcessingException e) {
                                    log.warn("Failed to serialize citations for persistence: {}", e.getMessage());
                                }
                            }
                            return Mono.just(formatChunk("", true, metadata));
                        }
                        return Mono.empty();
                    })
                    .doFinally(signalType -> {
                        if (signalType == SignalType.CANCEL) {
                            String partial = aiResponseAccumulator.toString();
                            if (!partial.isBlank()) {
                                String contentToSave = partial + "\n\n*[Interrupted by user]*";
                                saveAiResponse(contextId, contentToSave, channel, citationsJsonHolder[0]);
                                log.info("[Gateway] Stream cancelled for context {}. Saved {} chars as partial response.",
                                        contextId, contentToSave.length());
                            }
                        } else if (signalType == SignalType.ON_COMPLETE) {
                            saveAiResponse(contextId, aiResponseAccumulator.toString(), channel, citationsJsonHolder[0]);
                        }
                        // SignalType.ON_ERROR: handled by onErrorResume; do not persist partial error output.
                    });
                })
                .onErrorResume(StatusRuntimeException.class, e -> {
                    if (e.getStatus().getCode() == Status.Code.FAILED_PRECONDITION) {
                        String reason = extractViolationReason(e);
                        return Flux.error(new IntentViolationException(reason));
                    }
                    return Flux.error(e);
                })
                .onErrorResume(error -> {
                    if (error instanceof IntentViolationException) {
                        return Flux.error(error); 
                    }
                    
                    log.error("[Gateway] Python AI service error: {}", error.getMessage());
                    return Flux.just(formatError(
                            "Trợ giảng AI đang tạm thời không hoạt động. Câu hỏi của bạn đã được ghi nhận."
                    ));
                });
    }

    @Transactional
    protected void saveUserMessage(UUID contextId, String content, User user, ChatChannel channel) {
        if (content == null || content.isBlank()) {
            return;
        }
        if (channel == ChatChannel.FORUM) {
            forumPostRepository.save(ForumPost.builder()
                    .threadId(contextId)
                    .authorId(user.getId())
                    .authorType(PostAuthorType.STUDENT)
                    .content(content)
                    .verificationStatus(VerificationStatus.UNVERIFIED)
                    .build());
            return;
        }
        ChatSession session = sessionRepository.findById(contextId)
                .orElseGet(() -> sessionRepository.save(ChatSession.builder()
                        .id(contextId)
                        .studentId(user.getId())
                        .build()));
        chatMessageRepository.save(ChatMessage.builder()
                .sessionId(session.getId())
                .sender(MessageSenderType.STUDENT)
                .content(content)
                .build());
    }

    @Transactional
    protected void saveAiResponse(UUID contextId, String fullResponse, ChatChannel channel, String citationsJson) {
        if (fullResponse == null || fullResponse.isBlank()) {
            return;
        }
        if (channel == ChatChannel.FORUM) {
            forumPostRepository.save(ForumPost.builder()
                    .threadId(contextId)
                    .authorId(SystemConstants.SYSTEM_AI_ID)
                    .authorType(PostAuthorType.AI)
                    .content(fullResponse)
                    .originalAiContent(fullResponse)
                    .verificationStatus(VerificationStatus.UNVERIFIED)
                    .citations(citationsJson)
                    .build());
            return;
        }
        chatMessageRepository.save(ChatMessage.builder()
                .sessionId(contextId)
                .sender(MessageSenderType.AI)
                .content(fullResponse)
                .citations(citationsJson)
                .build());
    }

    private List<Message> buildHistory(UUID contextId, User currentUser, ChatChannel channel) {
        if (channel == ChatChannel.FORUM) {
            List<Message> full = forumPostRepository.findByThreadIdOrderByCreatedAtAsc(contextId).stream()
                    .map(post -> Message.newBuilder()
                            .setAuthorRole(post.getAuthorType() != null ? post.getAuthorType().name() : "STUDENT")
                            .setAuthorName(resolveForumAuthorName(post, currentUser))
                            .setContent(post.getContent() != null ? post.getContent() : "")
                            .build())
                    .collect(Collectors.toCollection(ArrayList::new));
            return trimHistoryTail(full);
        }
        List<Message> full = chatMessageRepository.findBySessionIdOrderByCreatedAtAsc(contextId).stream()
                .map(msg -> Message.newBuilder()
                        .setAuthorRole(msg.getSender().name())
                        .setAuthorName(resolveAuthorName(msg.getSender(), currentUser))
                        .setContent(msg.getContent() != null ? msg.getContent() : "")
                        .build())
                .collect(Collectors.toCollection(ArrayList::new));
        return trimHistoryTail(full);
    }

    private List<Message> trimHistoryTail(List<Message> chronological) {
        if (chronological.size() <= MAX_HISTORY_MESSAGES) {
            return chronological;
        }
        return new ArrayList<>(chronological.subList(
                chronological.size() - MAX_HISTORY_MESSAGES,
                chronological.size()));
    }

    // ─── PRIVATE HELPERS ────────────────────────────────────────────────────────

    private ServerSentEvent<Object> formatChunk(String text, boolean isFinished, ResponseMetadata metadata) {
        Map<String, Object> payload = new LinkedHashMap<>();
        payload.put("chunk", text);

        if (isFinished) {
            payload.put("is_finished", true);
            if (metadata != null) {
                Map<String, Object> metaMap = new LinkedHashMap<>();
                String agentUsed = metadata.getAgentUsed();
                metaMap.put("agent_used", agentUsed);
                metaMap.put("cache_hit", parseBooleanTag(agentUsed, "cache_hit"));
                Map<String, Object> usageMap = new LinkedHashMap<>();
                usageMap.put("input_tokens", parseIntegerTag(agentUsed, "input_tokens"));
                usageMap.put("output_tokens", parseIntegerTag(agentUsed, "output_tokens"));
                metaMap.put("usage", usageMap);
                List<Map<String, Object>> citations = metadata.getCitationsList().stream()
                        .map(c -> Map.<String, Object>of(
                                "source_file", c.getSourceFile(),
                                "page_number", c.getPageNumber(),
                                "document_id", c.getDocumentId(),
                                "source_uri", c.getSourceUri(),
                                "chunk_id", c.getChunkId(),
                                "chunk_index", c.getChunkIndex(),
                                "snippet", c.getSnippet()
                        ))
                        .collect(Collectors.toList());
                metaMap.put("citations", citations);
                payload.put("metadata", metaMap);
            }
        }

        return ServerSentEvent.builder()
                .data(payload)
                .build();
    }

    private ServerSentEvent<Object> formatError(String message) {
        return ServerSentEvent.builder()
                .event("error")
                .data(Map.of("error", message))
                .build();
    }

    private Boolean parseBooleanTag(String raw, String key) {
        if (raw == null || raw.isBlank() || key == null || key.isBlank()) {
            return null;
        }
        String prefix = key + "=";
        for (String token : raw.split("\\|")) {
            String t = token.trim();
            if (t.startsWith(prefix)) {
                String val = t.substring(prefix.length()).trim().toLowerCase();
                if ("true".equals(val)) return Boolean.TRUE;
                if ("false".equals(val)) return Boolean.FALSE;
            }
        }
        return null;
    }

    private Integer parseIntegerTag(String raw, String key) {
        if (raw == null || raw.isBlank() || key == null || key.isBlank()) {
            return null;
        }
        String prefix = key + "=";
        for (String token : raw.split("\\|")) {
            String t = token.trim();
            if (t.startsWith(prefix)) {
                String val = t.substring(prefix.length()).trim();
                if (val.isBlank()) return null;
                try {
                    return Integer.parseInt(val);
                } catch (NumberFormatException ignored) {
                    return null;
                }
            }
        }
        return null;
    }

    private String resolveAuthorName(MessageSenderType sender, User currentUser) {
        return switch (sender) {
            case STUDENT -> currentUser.getFullName() != null ? currentUser.getFullName() : "Student";
            case AI -> "Tro Giang AI";
            case TA -> "Teaching Assistant";
        };
    }

    private String resolveForumAuthorName(ForumPost post, User currentUser) {
        if (post.getAuthorType() == null) {
            return currentUser.getFullName() != null ? currentUser.getFullName() : "Student";
        }
        return switch (post.getAuthorType()) {
            case STUDENT -> currentUser.getFullName() != null ? currentUser.getFullName() : "Student";
            case AI -> "Tro Giang AI";
            case TA -> "Teaching Assistant";
        };
    }

    private String extractViolationReason(StatusRuntimeException e) {
        Metadata trailers = e.getTrailers();
        if (trailers == null) {
            return "[INTENT_VIOLATION] Message must be sent via private chat";
        }
        Metadata.Key<String> key = Metadata.Key.of("violation-reason", Metadata.ASCII_STRING_MARSHALLER);
        String reason = trailers.get(key);
        return reason == null || reason.isBlank()
                ? "[INTENT_VIOLATION] Message must be sent via private chat"
                : reason;
    }

    @Transactional
    protected void escalateToTA(UUID contextId, ChatChannel channel) {
        if (channel == ChatChannel.FORUM) {
            forumThreadRepository.findById(contextId).ifPresent(thread -> {
                thread.setStatus(ThreadStatus.ESCALATED);
                forumThreadRepository.save(thread);
                log.info("[Escalation] Thread {} status set to ESCALATED", contextId);
            });
        } else if (channel == ChatChannel.CHAT) {
            // TIP-006: For private chat escalations, create a forum thread so TAs can see it.
            sessionRepository.findById(contextId).ifPresent(session -> {
                String studentName = "Student";
                User student = null;
                try {
                    // Fetch user details to get the name
                    student = userRepository.findById(session.getStudentId()).orElse(null);
                    if (student != null) {
                        studentName = student.getFullName();
                    }
                } catch (Exception e) {
                    log.warn("Failed to fetch student details for escalation: {}", e.getMessage());
                }

                ForumThread escalatedThread = forumThreadRepository.save(ForumThread.builder()
                        .authorId(session.getStudentId())
                        .title("[ESCALATED CHAT] " + studentName + " - " + contextId.toString().substring(0, 8))
                        .status(ThreadStatus.ESCALATED)
                        .build());

                // Fetch the latest user message to put in the thread
                chatMessageRepository.findBySessionIdOrderByCreatedAtAsc(contextId).stream()
                        .filter(m -> m.getSender() == MessageSenderType.STUDENT)
                        .reduce((first, second) -> second) // Get last
                        .ifPresent(lastMsg -> {
                            forumPostRepository.save(ForumPost.builder()
                                    .threadId(escalatedThread.getId())
                                    .authorId(session.getStudentId())
                                    .authorType(PostAuthorType.STUDENT)
                                    .content(lastMsg.getContent())
                                    .verificationStatus(VerificationStatus.UNVERIFIED)
                                    .build());
                        });

                log.info("[Escalation] Created forum thread {} for private chat {}", escalatedThread.getId(), contextId);
            });
        }
    }
}
