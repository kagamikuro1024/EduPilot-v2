package team._8.aitrogiang.service;

import lombok.RequiredArgsConstructor;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.PageRequest;
import org.springframework.data.domain.Sort;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import team._8.aitrogiang.dto.*;
import team._8.aitrogiang.exception.ForbiddenException;
import team._8.aitrogiang.exception.ResourceNotFoundException;
import team._8.aitrogiang.model.*;
import team._8.aitrogiang.repository.*;

import java.time.Clock;
import java.time.LocalDateTime;
import java.time.format.DateTimeFormatter;
import java.time.ZoneOffset;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.util.*;

@Service
@RequiredArgsConstructor
public class ForumService {

    private static final int PAGE_SIZE = 20;
    private static final DateTimeFormatter ISO = DateTimeFormatter.ISO_DATE_TIME;

    private final PublicPrivacyFirewallService privacyFirewallService;
    private final ForumThreadRepository threadRepository;
    private final ForumPostRepository postRepository;
    private final TagRepository tagRepository;
    private final ForumThreadTagRepository threadTagRepository;
    private final UserRepository userRepository;
    private final JdbcClient jdbcClient;

    public String createThread(CreateThreadRequest req, User user) {
        privacyFirewallService.enforcePublicSafeOrThrow(combineTitleAndFirstMessage(req.getTitle(), req.getContent()));
        return persistNewThread(req, user);
    }

    private static String combineTitleAndFirstMessage(String title, String content) {
        String t = title == null ? "" : title.trim();
        String c = content == null ? "" : content.trim();
        if (t.isEmpty()) {
            return c;
        }
        if (c.isEmpty()) {
            return t;
        }
        return t + "\n" + c;
    }

    @Transactional(rollbackFor = Exception.class)
    protected String persistNewThread(CreateThreadRequest req, User user) {
        ForumThread thread = threadRepository.save(ForumThread.builder()
                .authorId(user.getId())
                .title(req.getTitle())
                .status(ThreadStatus.OPEN)
                .build());

        postRepository.save(ForumPost.builder()
                .threadId(thread.getId())
                .authorId(user.getId())
                .authorType(PostAuthorType.STUDENT)
                .content(req.getContent())
                .verificationStatus(VerificationStatus.UNVERIFIED)
                .build());

        resolveOrCreateTags(req.getTags(), thread.getId());
        return thread.getId().toString();
    }

    public ThreadListResponse getThreads(String status, String tag, String search, int page, User currentUser) {
        ThreadStatus parsedStatus = parseStatus(status);
        int safePage = Math.max(page, 1);

        UUID userId = currentUser != null ? currentUser.getId() : null;
        boolean isStaff = currentUser != null && (currentUser.getRole() == UserRole.TA || currentUser.getRole() == UserRole.ADMIN);

        Page<ForumThread> threadPage = threadRepository.findWithFilters(
                parsedStatus,
                blankToNull(tag),
                blankToNull(search),
                userId,
                isStaff,
                PageRequest.of(safePage - 1, PAGE_SIZE, Sort.by(Sort.Direction.DESC, "createdAt"))
        );

        List<ThreadItemDTO> data = threadPage.getContent().stream()
                .map(this::toThreadItem)
                .toList();

        return ThreadListResponse.builder()
                .data(data)
                .meta(ThreadListResponse.Meta.builder()
                        .total(threadPage.getTotalElements())
                        .page(safePage)
                        .build())
                .build();
    }

    public List<PostDTO> getMessages(UUID threadId, User currentUser) {
        ForumThread thread = threadRepository.findById(threadId)
                .orElseThrow(() -> new ResourceNotFoundException("Thread not found: " + threadId));
        
        if (thread.getStatus() == ThreadStatus.ESCALATED) {
            boolean isStaff = currentUser != null && (currentUser.getRole() == UserRole.TA || currentUser.getRole() == UserRole.ADMIN);
            if (!isStaff && (currentUser == null || !thread.getAuthorId().equals(currentUser.getId()))) {
                throw new ForbiddenException("You do not have permission to view this escalated private chat.");
            }
        }

        boolean canViewRejected = canViewRejected(currentUser);
        return postRepository.findByThreadIdOrderByCreatedAtAsc(threadId).stream()
                .filter(post -> canViewRejected || post.getVerificationStatus() != VerificationStatus.REJECTED)
                .map(post -> toPostDTO(post, currentUser))
                .toList();
    }

    @Transactional
    public String createHumanReply(UUID threadId, String content, User user) {
        ForumThread thread = threadRepository.findById(threadId)
                .orElseThrow(() -> new ResourceNotFoundException("Thread not found: " + threadId));
        
        if (thread.getStatus() == ThreadStatus.ESCALATED) {
            boolean isStaff = user.getRole() == UserRole.TA || user.getRole() == UserRole.ADMIN;
            if (!isStaff && !thread.getAuthorId().equals(user.getId())) {
                throw new ForbiddenException("You do not have permission to reply to this escalated private chat.");
            }
        }

        privacyFirewallService.enforcePublicSafeOrThrow(content);
        ForumPost post = postRepository.save(ForumPost.builder()
                .threadId(threadId)
                .authorId(user.getId())
                .authorType(user.getRole() == UserRole.TA ? PostAuthorType.TA : PostAuthorType.STUDENT)
                .content(content)
                .verificationStatus(VerificationStatus.UNVERIFIED)
                .build());
        return post.getId().toString();
    }

    @Transactional
    public void verifyMessage(UUID messageId, User user) {
        ensureStaff(user);
        ForumPost post = getAiPostForModeration(messageId);
        post.setVerificationStatus(VerificationStatus.VERIFIED);
        post.setVerifiedByTaId(user.getId());
        postRepository.save(post);
    }

    @Transactional
    public void correctMessage(UUID messageId, String correctedContent, User user) {
        ensureStaff(user);
        ForumPost post = getAiPostForModeration(messageId);
        if (correctedContent == null || correctedContent.isBlank()) {
            throw new ForbiddenException("Corrected content must not be empty");
        }
        if (post.getOriginalAiContent() == null || post.getOriginalAiContent().isBlank()) {
            post.setOriginalAiContent(post.getContent());
        }
        post.setContent(correctedContent);
        post.setVerificationStatus(VerificationStatus.CORRECTED);
        post.setVerifiedByTaId(user.getId());
        postRepository.save(post);
        upsertCorrectionChunk(post, user);
    }

    @Transactional
    public void rejectMessage(UUID messageId, User user) {
        ensureStaff(user);
        ForumPost post = getAiPostForModeration(messageId);
        post.setVerificationStatus(VerificationStatus.REJECTED);
        post.setVerifiedByTaId(user.getId());
        postRepository.save(post);
    }

    private void resolveOrCreateTags(List<String> rawTags, UUID threadId) {
        if (rawTags == null || rawTags.isEmpty()) {
            return;
        }
        for (String rawTag : rawTags) {
            String normalized = normalizeTag(rawTag);
            if (normalized == null) {
                continue;
            }
            Tag tag = tagRepository.findByNameIgnoreCase(normalized)
                    .orElseGet(() -> tagRepository.save(Tag.builder().name(normalized).build()));

            threadTagRepository.save(ForumThreadTag.builder()
                    .threadId(threadId)
                    .tagId(tag.getId())
                    .build());
        }
    }

    private ThreadItemDTO toThreadItem(ForumThread thread) {
        List<String> tags = threadTagRepository.findByThreadId(thread.getId()).stream()
                .map(ForumThreadTag::getTagId)
                .map(tagId -> tagRepository.findById(tagId).map(Tag::getName).orElse(null))
                .filter(Objects::nonNull)
                .toList();

        ForumPost lastMessage = postRepository.findFirstByThreadIdOrderByCreatedAtDesc(thread.getId());
        String preview = lastMessage != null ? truncate(lastMessage.getContent(), 120) : "";

        return ThreadItemDTO.builder()
                .id(thread.getId().toString())
                .title(thread.getTitle())
                .author(toAuthor(thread.getAuthorId(), "STUDENT"))
                .reply_count(postRepository.countByThreadId(thread.getId()))
                .status(thread.getStatus() != null ? thread.getStatus().name() : ThreadStatus.OPEN.name())
                .tags(tags)
                .last_message_preview(preview)
                .updated_at(toIso(lastMessage != null ? lastMessage.getCreatedAt() : thread.getCreatedAt()))
                .build();
    }

    private PostDTO toPostDTO(ForumPost post, User currentUser) {
        List<Map<String, Object>> parsedCitations = null;
        if (post.getAuthorType() == PostAuthorType.AI) {
            String raw = post.getCitations();
            if (raw != null && !raw.isBlank()) {
                try {
                    parsedCitations = new com.fasterxml.jackson.databind.ObjectMapper()
                            .readValue(raw, new com.fasterxml.jackson.core.type.TypeReference<List<Map<String, Object>>>() {});
                } catch (Exception e) {
                    parsedCitations = List.of();
                }
            } else {
                parsedCitations = List.of();
            }
        }
        AuthorDTO verifiedByTa = post.getVerifiedByTaId() != null
                ? toAuthor(post.getVerifiedByTaId(), "TA")
                : null;

        String originalAiContent = null;
        if (currentUser != null && canViewRejected(currentUser)) {
            originalAiContent = post.getOriginalAiContent();
        }

        return PostDTO.builder()
                .id(post.getId().toString())
                .author(toPostAuthor(post))
                .content(post.getContent())
                .original_ai_content(originalAiContent)
                .verification_status(post.getVerificationStatus() != null
                        ? post.getVerificationStatus().name()
                        : VerificationStatus.UNVERIFIED.name())
                .verified_by_ta(verifiedByTa)
                .created_at(toIso(post.getCreatedAt()))
                .reactions(Map.of())
                .is_accepted(post.getVerificationStatus() == VerificationStatus.VERIFIED)
                .citations(parsedCitations)
                .build();
    }

    private AuthorDTO toPostAuthor(ForumPost post) {
        String role = post.getAuthorType() != null ? post.getAuthorType().name() : "STUDENT";
        return toAuthor(post.getAuthorId(), role);
    }

    private AuthorDTO toAuthor(UUID userId, String defaultRole) {
        if (userId == null) {
            return AuthorDTO.builder()
                    .id("unknown")
                    .role(defaultRole)
                    .name(defaultRole.equals("TA") ? "Teaching Assistant" : "Sinh vien")
                    .avatar(null)
                    .build();
        }
        if (team._8.aitrogiang.constant.SystemConstants.SYSTEM_AI_ID.equals(userId)) {
            return AuthorDTO.builder()
                    .id(userId.toString())
                    .role("AI")
                    .name("EduBot")
                    .avatar(null)
                    .build();
        }
        User user = userRepository.findById(userId).orElse(null);
        if (user == null) {
            return AuthorDTO.builder()
                    .id(userId.toString())
                    .role(defaultRole)
                    .name(defaultRole.equals("TA") ? "Teaching Assistant" : "Sinh vien")
                    .avatar(null)
                    .build();
        }
        return AuthorDTO.builder()
                .id(userId.toString())
                .role(user.getRole().name())
                .name(user.getFullName() != null ? user.getFullName() : user.getEmail())
                .avatar(null)
                .build();
    }

    private ThreadStatus parseStatus(String status) {
        if (status == null || status.isBlank()) {
            return null;
        }
        try {
            return ThreadStatus.valueOf(status.trim().toUpperCase(Locale.ROOT));
        } catch (IllegalArgumentException ex) {
            return null;
        }
    }

    private String normalizeTag(String tag) {
        if (tag == null) {
            return null;
        }
        String cleaned = tag.trim();
        return cleaned.isEmpty() ? null : cleaned;
    }

    private String blankToNull(String value) {
        return value == null || value.isBlank() ? null : value.trim();
    }

    private void ensureStaff(User user) {
        if (user.getRole() != UserRole.TA && user.getRole() != UserRole.ADMIN) {
            throw new ForbiddenException("Only TA or ADMIN can verify AI messages");
        }
    }

    private ForumPost getAiPostForModeration(UUID messageId) {
        ForumPost post = postRepository.findById(messageId)
                .orElseThrow(() -> new ResourceNotFoundException("Message not found: " + messageId));
        if (post.getAuthorType() != PostAuthorType.AI) {
            throw new ForbiddenException("Only AI messages can be moderated");
        }
        return post;
    }

    private boolean canViewRejected(User user) {
        return user != null && (user.getRole() == UserRole.TA || user.getRole() == UserRole.ADMIN);
    }

    private String truncate(String content, int maxLen) {
        if (content == null) {
            return "";
        }
        if (content.length() <= maxLen) {
            return content;
        }
        return content.substring(0, maxLen - 3) + "...";
    }

    private String toIso(LocalDateTime dateTime) {
        LocalDateTime value = dateTime != null ? dateTime : LocalDateTime.now(Clock.systemUTC());
        return value.atOffset(ZoneOffset.UTC).format(ISO);
    }

    private void upsertCorrectionChunk(ForumPost post, User user) {
        String content = post.getContent();
        if (content == null || content.isBlank()) {
            return;
        }
        String contentHash = sha256(content.trim());
        String sourceUrl = "forum://posts/" + post.getId();
        String metadata = buildCorrectionMetadata(post, user);

        try {
            String sql = """
                    INSERT INTO document_chunks (source_type, document_id, content_hash, source_url, forum_post_id, content, embedding, metadata)
                    VALUES ('FORUM_CORRECTION', NULL, :contentHash, :sourceUrl, :postId, :content, NULL, CAST(:metadata AS jsonb))
                    ON CONFLICT (content_hash)
                    DO UPDATE SET
                        source_type = EXCLUDED.source_type,
                        source_url = EXCLUDED.source_url,
                        forum_post_id = EXCLUDED.forum_post_id,
                        content = EXCLUDED.content,
                        metadata = EXCLUDED.metadata,
                        updated_at = CURRENT_TIMESTAMP
                    """;
            jdbcClient.sql(sql)
                    .param("contentHash", contentHash)
                    .param("sourceUrl", sourceUrl)
                    .param("postId", post.getId())
                    .param("content", content)
                    .param("metadata", metadata)
                    .update();
        } catch (Exception e) {
            System.err.println("[ERROR] Failed to upsert correction chunk to RAG database: " + e.getMessage());
            e.printStackTrace();
            // We don't rethrow to avoid failing the whole post correction if RAG sync fails,
            // or we could rethrow if we want it to be atomic. 
            // The user said "sửa lại" because of 500 error, so it's currently failing the whole request.
            throw e;
        }
    }

    private String buildCorrectionMetadata(ForumPost post, User user) {
        String original = post.getOriginalAiContent() != null ? post.getOriginalAiContent() : "";
        String verifiedBy = user != null && user.getId() != null ? user.getId().toString() : "";
        String threadId = post.getThreadId() != null ? post.getThreadId().toString() : "";
        return """
                {"feedback_source":"TA_CORRECTION","verification_status":"CORRECTED","verified_by_ta_id":"%s","thread_id":"%s","forum_post_id":"%s","original_ai_content":%s}
                """.formatted(
                verifiedBy,
                threadId,
                post.getId(),
                jsonString(original)
        );
    }

    private String jsonString(String value) {
        String safe = value == null ? "" : value;
        return "\"" + safe
                .replace("\\", "\\\\")
                .replace("\"", "\\\"")
                .replace("\n", "\\n")
                .replace("\r", "\\r")
                .replace("\t", "\\t") + "\"";
    }

    private String sha256(String input) {
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            byte[] hash = digest.digest(input.getBytes(StandardCharsets.UTF_8));
            StringBuilder hex = new StringBuilder();
            for (byte b : hash) {
                hex.append(String.format("%02x", b));
            }
            return hex.toString();
        } catch (Exception ex) {
            throw new IllegalStateException("Could not hash correction content", ex);
        }
    }
}
