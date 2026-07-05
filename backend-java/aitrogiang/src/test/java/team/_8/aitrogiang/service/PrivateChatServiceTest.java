package team._8.aitrogiang.service;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import team._8.aitrogiang.exception.ForbiddenException;
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

import java.util.Optional;
import java.util.UUID;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
class PrivateChatServiceTest {

    @Mock
    private ChatSessionRepository sessionRepository;
    @Mock
    private ChatMessageRepository messageRepository;
    @Mock
    private ForumThreadRepository forumThreadRepository;
    @Mock
    private ForumPostRepository forumPostRepository;
    @Mock
    private UserRepository userRepository;

    private PrivateChatService privateChatService;

    @BeforeEach
    void setUp() {
        privateChatService = new PrivateChatService(sessionRepository, messageRepository, forumThreadRepository, forumPostRepository, userRepository);
    }

    @Test
    void submitFeedback_updatesAiMessageFeedbackForOwner() {
        UUID sessionId = UUID.randomUUID();
        UUID ownerId = UUID.randomUUID();
        UUID messageId = UUID.randomUUID();
        User owner = User.builder().id(ownerId).role(UserRole.STUDENT).build();
        ChatMessage aiMessage = ChatMessage.builder()
                .id(messageId)
                .sessionId(sessionId)
                .sender(MessageSenderType.AI)
                .build();
        ChatSession session = ChatSession.builder()
                .id(sessionId)
                .studentId(ownerId)
                .build();

        when(messageRepository.findById(messageId)).thenReturn(Optional.of(aiMessage));
        when(sessionRepository.findById(sessionId)).thenReturn(Optional.of(session));

        privateChatService.submitFeedback(messageId, "LIKE", owner);

        assertThat(aiMessage.getFeedback()).isEqualTo("LIKE");
        verify(messageRepository).save(aiMessage);
    }

    @Test
    void submitFeedback_rejectsNonAiMessages() {
        UUID messageId = UUID.randomUUID();
        User owner = User.builder().id(UUID.randomUUID()).role(UserRole.STUDENT).build();
        ChatMessage studentMessage = ChatMessage.builder()
                .id(messageId)
                .sessionId(UUID.randomUUID())
                .sender(MessageSenderType.STUDENT)
                .build();
        when(messageRepository.findById(messageId)).thenReturn(Optional.of(studentMessage));

        assertThatThrownBy(() -> privateChatService.submitFeedback(messageId, "DISLIKE", owner))
                .isInstanceOf(ForbiddenException.class)
                .hasMessageContaining("AI messages");
        verify(messageRepository, never()).save(studentMessage);
    }
}
