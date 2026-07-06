package team._8.aitrogiang.service;

import lombok.extern.slf4j.Slf4j;
import lombok.RequiredArgsConstructor;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.data.domain.Sort;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.web.multipart.MultipartFile;
import team._8.aitrogiang.dto.AdminDocumentListItemResponse;
import team._8.aitrogiang.dto.AdminDocumentStatsResponse;
import team._8.aitrogiang.dto.AdminDocumentStatusResponse;
import team._8.aitrogiang.dto.AdminDocumentUploadResponse;
import team._8.aitrogiang.dto.AnalyticsSummaryDTO;
import team._8.aitrogiang.exception.DocumentProcessingException;
import team._8.aitrogiang.exception.ResourceNotFoundException;
import team._8.aitrogiang.model.Document;
import team._8.aitrogiang.model.DocumentStatus;
import team._8.aitrogiang.model.DocumentType;
import team._8.aitrogiang.model.User;
import team._8.aitrogiang.repository.AnalyticsAtRiskStudentRepository;
import team._8.aitrogiang.repository.AnalyticsTopicDifficultyRepository;
import team._8.aitrogiang.repository.DocumentRepository;
import team._8.aitrogiang.repository.ForumPostRepository;
import team._8.aitrogiang.repository.ForumThreadRepository;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.time.LocalDateTime;
import java.time.ZoneOffset;
import java.util.List;
import java.util.UUID;

@Service
@Slf4j
@RequiredArgsConstructor
public class AdminService {

    private final DocumentRepository documentRepository;
    private final AnalyticsAtRiskStudentRepository atRiskStudentRepository;
    private final AnalyticsTopicDifficultyRepository topicDifficultyRepository;
    private final ForumThreadRepository forumThreadRepository;
    private final ForumPostRepository forumPostRepository;
    private final PythonAiOrchestratorClient pythonClient;

    @Value("${app.upload.dir:./uploads}")
    private String uploadDir;

    public AdminDocumentUploadResponse uploadDocument(MultipartFile file, User currentUser, String documentTypeRaw) {
        if (file == null || file.isEmpty()) {
            throw new IllegalArgumentException("Uploaded file must not be empty");
        }

        Path savedPath = storeToSharedVolume(file);
        String originalFilename = file.getOriginalFilename() == null ? "document" : file.getOriginalFilename();
        DocumentType documentType = resolveDocumentType(documentTypeRaw);
        documentType = applyGradeReportFilenameSafetyOverride(documentType, originalFilename);

        Document document = Document.builder()
                .title(originalFilename)
                .uploadedById(currentUser.getId())
                .status(DocumentStatus.PROCESSING)
                .documentType(documentType)
                .fileSizeBytes(file.getSize())
                .build();
        Document saved = documentRepository.save(document);

        boolean accepted = pythonClient.processDocument(saved.getId().toString(), savedPath.toString());
        if (!accepted) {
            saved.setStatus(DocumentStatus.FAILED);
            documentRepository.save(saved);
            throw new DocumentProcessingException("Python orchestrator rejected document processing");
        }

        return AdminDocumentUploadResponse.builder()
                .document_id(saved.getId().toString())
                .status(saved.getStatus().name())
                .build();
    }

    public AdminDocumentStatusResponse getDocumentStatus(UUID documentId) {
        Document document = documentRepository.findById(documentId)
                .orElseThrow(() -> new ResourceNotFoundException("Document not found: " + documentId));
        return AdminDocumentStatusResponse.builder()
                .status(document.getStatus().name())
                .build();
    }

    public List<AdminDocumentListItemResponse> listDocuments() {
        return documentRepository.findAll(Sort.by(Sort.Direction.DESC, "createdAt")).stream()
                .map(document -> AdminDocumentListItemResponse.builder()
                        .document_id(document.getId().toString())
                        .name(resolveDocumentName(document))
                        .status(document.getStatus() == null ? DocumentStatus.PROCESSING.name() : document.getStatus().name())
                        .document_type(document.getDocumentType() == null
                                ? DocumentType.COURSE_MATERIAL.name()
                                : document.getDocumentType().name())
                        .file_size_bytes(document.getFileSizeBytes())
                        .created_at(resolveCreatedAt(document))
                        .build())
                .toList();
    }

    @Transactional
    public void deleteDocument(UUID documentId) {
        if (!documentRepository.existsById(documentId)) {
            throw new ResourceNotFoundException("Document not found: " + documentId);
        }
        documentRepository.deleteChunksByDocumentId(documentId);
        documentRepository.deleteById(documentId);
        log.info("Document {} and its chunks deleted successfully", documentId);
    }

    public void updateDocumentStatus(UUID documentId, DocumentStatus status) {
        Document document = documentRepository.findById(documentId)
                .orElseThrow(() -> new ResourceNotFoundException("Document not found: " + documentId));
        document.setStatus(status);
        documentRepository.save(document);
        log.info("[Callback] Document {} status updated to {}", documentId, status);
    }

    public AdminDocumentStatsResponse getDocumentStats() {
        long totalDocs = documentRepository.count();
        long failedDocs = documentRepository.countByStatus(DocumentStatus.FAILED);
        long readyDocs = documentRepository.countByStatus(DocumentStatus.READY);
        
        double health = totalDocs == 0 ? 100.0 : (double) readyDocs / totalDocs * 100;
        
        long totalBytes = documentRepository.findAll().stream()
                .mapToLong(d -> d.getFileSizeBytes() == null ? 0 : d.getFileSizeBytes())
                .sum();
        
        String volumeLabel = formatVolume(totalBytes);
        
        return AdminDocumentStatsResponse.builder()
                .total_documents(totalDocs)
                .index_health(String.format("%.0f%%", health))
                .knowledge_volume(volumeLabel)
                .build();
    }

    private String formatVolume(long bytes) {
        if (bytes < 1024) return bytes + " B";
        if (bytes < 1024 * 1024) return String.format("%.1f KB", bytes / 1024.0);
        if (bytes < 1024 * 1024 * 1024) return String.format("%.1f MB", bytes / (1024.0 * 1024.0));
        return String.format("%.1f GB", bytes / (1024.0 * 1024.0 * 1024.0));
    }

    private String resolveDocumentName(Document document) {
        if (document.getTitle() != null && !document.getTitle().isBlank()) {
            return document.getTitle();
        }
        if (document.getFilename() != null && !document.getFilename().isBlank()) {
            return document.getFilename();
        }
        return "Untitled document";
    }

    private String resolveCreatedAt(Document document) {
        LocalDateTime created = document.getCreatedAt() != null ? document.getCreatedAt() : document.getUpdatedAt();
        return created == null ? null : created.atOffset(ZoneOffset.UTC).toString();
    }

    /**
     * Parses optional multipart {@code document_type}; defaults to course material.
     */
    private DocumentType resolveDocumentType(String raw) {
        if (raw == null || raw.isBlank()) {
            return DocumentType.COURSE_MATERIAL;
        }
        String normalized = raw.trim().toUpperCase();
        try {
            return DocumentType.valueOf(normalized);
        } catch (IllegalArgumentException ex) {
            log.warn("Unknown document_type '{}', defaulting to COURSE_MATERIAL", raw);
            return DocumentType.COURSE_MATERIAL;
        }
    }

    DocumentType applyGradeReportFilenameSafetyOverride(DocumentType selectedType, String filename) {
        String normalized = filename == null ? "" : filename.toLowerCase(java.util.Locale.ROOT);
        boolean clearlyGradeReport = normalized.contains("bang_diem")
                || normalized.contains("bang-diem")
                || normalized.contains("bang diem")
                || normalized.contains("bảng_điểm")
                || normalized.contains("bảng-điểm")
                || normalized.contains("bảng điểm")
                || normalized.contains("grade_report")
                || normalized.contains("grade-report")
                || normalized.contains("grade report");
        if (clearlyGradeReport && selectedType != DocumentType.GRADE_REPORT) {
            log.warn("Safety override: classifying grade-like filename '{}' as GRADE_REPORT", filename);
            return DocumentType.GRADE_REPORT;
        }
        return selectedType;
    }

    public AnalyticsSummaryDTO getAnalyticsSummary() {
        LocalDateTime since = LocalDateTime.now().minusDays(7);
        long totalQuestions = forumThreadRepository.countSince(since);
        Double aiResolutionRate = forumPostRepository.aiResolutionRateSince(since);

        return AnalyticsSummaryDTO.builder()
                .total_questions_this_week(totalQuestions)
                .ai_resolution_rate(aiResolutionRate == null ? 0.0 : aiResolutionRate)
                .top_tags(topicDifficultyRepository.findTop10ByOrderByQueryCountDesc().stream()
                        .map(item -> AnalyticsSummaryDTO.TopTagDTO.builder()
                                .name(item.getTopicName())
                                .query_count(item.getQueryCount() == null ? 0 : item.getQueryCount())
                                .difficulty_score(item.getDifficultyScore() == null ? 0f : item.getDifficultyScore())
                                .build())
                        .toList())
                .at_risk_students(atRiskStudentRepository.findTop10ByOrderByReportDateDesc().stream()
                        .map(item -> AnalyticsSummaryDTO.AtRiskStudentDTO.builder()
                                .student_id(item.getStudentId() == null ? "" : item.getStudentId().toString())
                                .risk_level(item.getRiskLevel() == null ? "UNKNOWN" : item.getRiskLevel().name())
                                .reason(item.getReason())
                                .build())
                        .toList())
                .build();
    }

    public boolean editChunkContent(team._8.aitrogiang.dto.AdminEditChunkRequest request, User currentUser) {
        log.info("TA [{}] editing chunk [{}]. Calling Python orchestrator...", currentUser.getUsername(), request.getChunk_id());
        return pythonClient.updateChunkContent(
                request.getChunk_id(),
                request.getNew_content(),
                currentUser.getUsername()
        );
    }

    private Path storeToSharedVolume(MultipartFile file) {
        try {
            Path uploadPath = Paths.get(uploadDir).toAbsolutePath().normalize();
            Files.createDirectories(uploadPath);

            java.util.Set<String> allowedExtensions = java.util.Set.of(".pdf", ".txt", ".docx");
            String originalFilename = file.getOriginalFilename() == null ? "document" : file.getOriginalFilename();
            int lastDot = originalFilename.lastIndexOf('.');
            if (lastDot == -1) {
                throw new IllegalArgumentException("File upload rejected: Missing file extension. Allowed extensions are .pdf, .txt, .docx");
            }
            String extension = originalFilename.substring(lastDot).toLowerCase();
            
            // TIP-006: Zero-Tolerance Extension Allowlist
            if (!allowedExtensions.contains(extension)) {
                throw new IllegalArgumentException("Invalid file extension: " + extension + ". Only PDF, TXT, and DOCX are allowed.");
            }
            
            // TIP-005: Use pure UUID for the physical file name to prevent Path Traversal
            Path destination = uploadPath.resolve(UUID.randomUUID().toString() + extension);
            file.transferTo(destination.toFile());
            return destination;
        } catch (IOException e) {
            throw new IllegalStateException("Cannot persist uploaded file to shared volume", e);
        }
    }
}
