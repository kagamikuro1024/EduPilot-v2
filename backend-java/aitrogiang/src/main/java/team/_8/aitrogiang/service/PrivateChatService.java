package team._8.aitrogiang.service;

import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import team._8.aitrogiang.dto.AuthorDTO;
import team._8.aitrogiang.dto.ChatMessageDTO;
import team._8.aitrogiang.exception.ForbiddenException;
import team._8.aitrogiang.exception.ResourceNotFoundException;
import team._8.aitrogiang.model.ChatFeedbackType;
import team._8.aitrogiang.model.ChatMessage;
import team._8.aitrogiang.model.ChatSession;
import team._8.aitrogiang.model.MessageSenderType;
import team._8.aitrogiang.model.User;
import team._8.aitrogiang.model.UserRole;
import team._8.aitrogiang.repository.ChatMessageRepository;
import team._8.aitrogiang.repository.ChatSessionRepository;
import team._8.aitrogiang.repository.ForumPostRepository;
import team._8.aitrogiang.repository.ForumThreadRepository;
import team._8.aitrogiang.repository.UserRepository;
import team._8.aitrogiang.model.ForumThread;
import team._8.aitrogiang.model.ForumPost;
import team._8.aitrogiang.model.PostAuthorType;
import team._8.aitrogiang.model.VerificationStatus;
import team._8.aitrogiang.model.ThreadStatus;

import java.time.Clock;
import java.time.LocalDateTime;
import java.time.ZoneOffset;
import java.time.format.DateTimeFormatter;
import java.util.List;
import java.util.UUID;

@Service
@RequiredArgsConstructor
public class PrivateChatService {

    private static final DateTimeFormatter ISO = DateTimeFormatter.ISO_DATE_TIME;
    private final ChatSessionRepository sessionRepository;
    private final ChatMessageRepository messageRepository;
    private final ForumThreadRepository forumThreadRepository;
    private final ForumPostRepository forumPostRepository;
    private final UserRepository userRepository;

    @Transactional
    public ChatSession createSession(User currentUser) {
        return sessionRepository.save(ChatSession.builder()
                .studentId(currentUser.getId())
                .build());
    }

    public List<ChatSession> listSessions(User currentUser) {
        return sessionRepository.findByStudentIdOrderByCreatedAtDesc(currentUser.getId());
    }

    public List<ChatMessageDTO> getMessages(UUID sessionId, User currentUser) {
        ChatSession session = sessionRepository.findById(sessionId)
                .orElseThrow(() -> new ResourceNotFoundException("Session not found: " + sessionId));
        if (!session.getStudentId().equals(currentUser.getId())) {
            throw new ForbiddenException("You do not have permission to access this chat session");
        }
        return messageRepository.findBySessionIdOrderByCreatedAtAsc(sessionId).stream()
                .map(msg -> {
                    List<java.util.Map<String, Object>> parsedCitations = null;
                    if (msg.getSender() == MessageSenderType.AI) {
                        String raw = msg.getCitations();
                        if (raw != null && !raw.isBlank()) {
                            try {
                                parsedCitations = new com.fasterxml.jackson.databind.ObjectMapper()
                                        .readValue(raw, new com.fasterxml.jackson.core.type.TypeReference<List<java.util.Map<String, Object>>>() {});
                            } catch (Exception e) {
                                parsedCitations = List.of();
                            }
                        } else {
                            parsedCitations = List.of();
                        }
                    }
                    return ChatMessageDTO.builder()
                            .id(msg.getId().toString())
                            .author(toAuthor(msg.getSender(), currentUser))
                            .content(msg.getContent())
                            .feedback(msg.getFeedback())
                            .created_at(toIso(msg.getCreatedAt()))
                            .citations(parsedCitations)
                            .build();
                })
                .toList();
    }

    @Transactional
    public void submitFeedback(UUID messageId, String feedbackRaw, User currentUser) {
        ChatMessage message = messageRepository.findById(messageId)
                .orElseThrow(() -> new ResourceNotFoundException("Message not found: " + messageId));
        if (message.getSender() != MessageSenderType.AI) {
            throw new ForbiddenException("Feedback can only be submitted for AI messages");
        }

        ChatSession session = sessionRepository.findById(message.getSessionId())
                .orElseThrow(() -> new ResourceNotFoundException("Session not found: " + message.getSessionId()));
        boolean isOwner = session.getStudentId().equals(currentUser.getId());
        boolean isStaff = currentUser.getRole() == UserRole.TA || currentUser.getRole() == UserRole.ADMIN;
        if (!isOwner && !isStaff) {
            throw new ForbiddenException("You do not have permission to submit feedback for this message");
        }

        ChatFeedbackType feedback;
        try {
            feedback = ChatFeedbackType.valueOf(feedbackRaw.trim().toUpperCase());
        } catch (Exception ex) {
            throw new ForbiddenException("Invalid feedback value");
        }
        message.setFeedback(feedback.name());
        messageRepository.save(message);

        if (feedback == ChatFeedbackType.NEEDS_TA) {
            escalateSpecificMessageToTA(session, messageId);
        }
    }

    private void escalateSpecificMessageToTA(ChatSession session, UUID aiMessageId) {
        String studentName = "Student";
        try {
            User student = userRepository.findById(session.getStudentId()).orElse(null);
            if (student != null && student.getFullName() != null) {
                studentName = student.getFullName();
            }
        } catch (Exception e) {
            // Ignore
        }

        // Create the escalated thread
        ForumThread escalatedThread = forumThreadRepository.save(ForumThread.builder()
                .authorId(session.getStudentId())
                .title("[ESCALATED CHAT] " + studentName + " - " + session.getId().toString().substring(0, 8))
                .status(ThreadStatus.ESCALATED)
                .build());

        // Find the last student message sent before this AI message
        ChatMessage aiMsg = messageRepository.findById(aiMessageId).orElse(null);
        if (aiMsg != null) {
            messageRepository.findBySessionIdOrderByCreatedAtAsc(session.getId()).stream()
                    .filter(m -> m.getSender() == MessageSenderType.STUDENT && m.getCreatedAt().isBefore(aiMsg.getCreatedAt()) || m.getCreatedAt().isEqual(aiMsg.getCreatedAt()))
                    .reduce((first, second) -> second) // Get last student message before the AI message
                    .ifPresent(lastMsg -> {
                        forumPostRepository.save(ForumPost.builder()
                                .threadId(escalatedThread.getId())
                                .authorId(session.getStudentId())
                                .authorType(PostAuthorType.STUDENT)
                                .content(lastMsg.getContent())
                                .verificationStatus(VerificationStatus.UNVERIFIED)
                                .build());
                    });
        }
    }

    public void validateOwnership(UUID sessionId, User currentUser) {
        ChatSession session = sessionRepository.findById(sessionId)
                .orElseThrow(() -> new ResourceNotFoundException("Session not found: " + sessionId));
        if (!session.getStudentId().equals(currentUser.getId())) {
            throw new ForbiddenException("You do not have permission to access this chat session");
        }
    }

    @Transactional
    public void deleteSession(UUID sessionId, User currentUser) {
        ChatSession session = sessionRepository.findById(sessionId)
                .orElseThrow(() -> new ResourceNotFoundException("Session not found: " + sessionId));
        if (!session.getStudentId().equals(currentUser.getId())) {
            throw new ForbiddenException("You do not have permission to delete this chat session");
        }
        // Xóa các messages trước (nếu có constraint FK)
        messageRepository.deleteBySessionId(sessionId);
        // Xóa session
        sessionRepository.delete(session);
    }

    private AuthorDTO toAuthor(MessageSenderType sender, User currentUser) {
        return switch (sender) {
            case STUDENT -> AuthorDTO.builder()
                    .id(currentUser.getId().toString())
                    .role("STUDENT")
                    .name(currentUser.getFullName() != null ? currentUser.getFullName() : currentUser.getEmail())
                    .avatar(null)
                    .build();
            case TA -> AuthorDTO.builder()
                    .id("ta")
                    .role("TA")
                    .name("Teaching Assistant")
                    .avatar(null)
                    .build();
            case AI -> AuthorDTO.builder()
                    .id("ai_bot")
                    .role("AI")
                    .name("Tro Giang AI")
                    .avatar(null)
                    .build();
        };
    }

    private String toIso(LocalDateTime dateTime) {
        LocalDateTime value = dateTime != null ? dateTime : LocalDateTime.now(Clock.systemUTC());
        return value.atOffset(ZoneOffset.UTC).format(ISO);
    }
}
