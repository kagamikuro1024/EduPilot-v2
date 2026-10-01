package team._8.aitrogiang.service;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Answers;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.jdbc.core.simple.JdbcClient;
import team._8.aitrogiang.constant.SystemConstants;
import team._8.aitrogiang.exception.ForbiddenException;
import team._8.aitrogiang.model.ForumPost;
import team._8.aitrogiang.model.ForumThread;
import team._8.aitrogiang.model.PostAuthorType;
import team._8.aitrogiang.model.User;
import team._8.aitrogiang.model.UserRole;
import team._8.aitrogiang.model.VerificationStatus;
import team._8.aitrogiang.repository.ForumPostRepository;
import team._8.aitrogiang.repository.ForumThreadRepository;
import team._8.aitrogiang.repository.ForumThreadTagRepository;
import team._8.aitrogiang.repository.TagRepository;
import team._8.aitrogiang.repository.UserRepository;

import java.util.List;
import java.util.Optional;
import java.util.UUID;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
class ForumServiceTest {

    @Mock
    private PublicPrivacyFirewallService privacyFirewallService;
    @Mock
    private ForumThreadRepository threadRepository;
    @Mock
    private ForumPostRepository postRepository;
    @Mock
    private TagRepository tagRepository;
    @Mock
    private ForumThreadTagRepository threadTagRepository;
    @Mock
    private UserRepository userRepository;
    @Mock(answer = Answers.RETURNS_DEEP_STUBS)
    private JdbcClient jdbcClient;

    private ForumService forumService;

    @BeforeEach
    void setUp() {
        forumService = new ForumService(
                privacyFirewallService,
                threadRepository,
                postRepository,
                tagRepository,
                threadTagRepository,
                userRepository,
                jdbcClient
        );
    }

    @Test
    void verifyMessage_throwsWhenTargetMessageIsNotAi() {
        UUID messageId = UUID.randomUUID();
        User ta = User.builder().id(UUID.randomUUID()).role(UserRole.TA).build();
        ForumPost studentPost = ForumPost.builder()
                .id(messageId)
                .authorType(PostAuthorType.STUDENT)
                .verificationStatus(VerificationStatus.UNVERIFIED)
                .build();
        when(postRepository.findById(messageId)).thenReturn(Optional.of(studentPost));

        assertThatThrownBy(() -> forumService.verifyMessage(messageId, ta))
                .isInstanceOf(ForbiddenException.class)
                .hasMessageContaining("Only AI messages");
        verify(postRepository, never()).save(any(ForumPost.class));
    }

    @Test
    void correctMessage_preservesOriginalAndSetsCorrectedStatus() {
        UUID messageId = UUID.randomUUID();
        UUID taId = UUID.randomUUID();
        User ta = User.builder().id(taId).role(UserRole.TA).build();
        ForumPost aiPost = ForumPost.builder()
                .id(messageId)
                .authorId(SystemConstants.SYSTEM_AI_ID)
                .authorType(PostAuthorType.AI)
                .content("old content")
                .verificationStatus(VerificationStatus.UNVERIFIED)
                .build();
        when(postRepository.findById(messageId)).thenReturn(Optional.of(aiPost));

        forumService.correctMessage(messageId, "new corrected content", ta);

        assertThat(aiPost.getOriginalAiContent()).isEqualTo("old content");
        assertThat(aiPost.getContent()).isEqualTo("new corrected content");
        assertThat(aiPost.getVerificationStatus()).isEqualTo(VerificationStatus.CORRECTED);
        assertThat(aiPost.getVerifiedByTaId()).isEqualTo(taId);
        verify(postRepository).save(aiPost);
    }

    @Test
    void getMessages_hidesRejectedAiPostsFromStudents() {
        UUID threadId = UUID.randomUUID();
        User student = User.builder().id(UUID.randomUUID()).role(UserRole.STUDENT).build();
        ForumPost rejectedAiPost = ForumPost.builder()
                .id(UUID.randomUUID())
                .authorId(SystemConstants.SYSTEM_AI_ID)
                .authorType(PostAuthorType.AI)
                .content("wrong answer")
                .verificationStatus(VerificationStatus.REJECTED)
                .build();
        ForumPost verifiedAiPost = ForumPost.builder()
                .id(UUID.randomUUID())
                .authorId(SystemConstants.SYSTEM_AI_ID)
                .authorType(PostAuthorType.AI)
                .content("safe answer")
                .verificationStatus(VerificationStatus.VERIFIED)
                .build();
        when(threadRepository.findById(threadId)).thenReturn(Optional.of(ForumThread.builder().id(threadId).build()));
        when(postRepository.findByThreadIdOrderByCreatedAtAsc(threadId))
                .thenReturn(List.of(rejectedAiPost, verifiedAiPost));

        List<team._8.aitrogiang.dto.PostDTO> visible = forumService.getMessages(threadId, student);

        assertThat(visible).hasSize(1);
        assertThat(visible.get(0).getContent()).isEqualTo("safe answer");
    }
}
