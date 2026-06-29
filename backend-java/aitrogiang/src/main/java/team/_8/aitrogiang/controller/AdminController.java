package team._8.aitrogiang.controller;

import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.Authentication;
import org.springframework.format.annotation.DateTimeFormat;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.multipart.MultipartFile;
import team._8.aitrogiang.dto.AdminDocumentListItemResponse;
import team._8.aitrogiang.dto.AdminDocumentStatusResponse;
import team._8.aitrogiang.dto.AdminDocumentStatsResponse;
import team._8.aitrogiang.dto.AdminDocumentUploadResponse;
import team._8.aitrogiang.dto.AnalyticsMetricsDTO;
import team._8.aitrogiang.dto.AnalyticsSummaryDTO;
import team._8.aitrogiang.dto.AtRiskStudentRecord;
import team._8.aitrogiang.dto.StudentActivityPointRecord;
import team._8.aitrogiang.dto.TopicDifficultyRecord;
import team._8.aitrogiang.model.User;
import team._8.aitrogiang.service.AdminService;
import team._8.aitrogiang.service.AnalyticsService;

import java.time.LocalDateTime;
import java.util.List;
import java.util.UUID;

@RestController
@RequestMapping("/api/v1/admin")
@RequiredArgsConstructor
public class AdminController {

    private final AdminService adminService;
    private final AnalyticsService analyticsService;

    @GetMapping("/analytics/topics")
    public ResponseEntity<List<TopicDifficultyRecord>> getTopTopics() {
        return ResponseEntity.ok(analyticsService.getTopDifficultTopics());
    }

    @GetMapping("/analytics/at-risk")
    public ResponseEntity<List<AtRiskStudentRecord>> getAtRiskStudents() {
        return ResponseEntity.ok(analyticsService.getAtRiskStudents());
    }

    @PostMapping(value = "/documents", consumes = "multipart/form-data")
    public ResponseEntity<AdminDocumentUploadResponse> uploadDocument(
            @RequestParam("file") MultipartFile file,
            @RequestParam(value = "document_type", required = false) String documentType,
            Authentication authentication
    ) {
        User currentUser = (User) authentication.getPrincipal();
        return ResponseEntity.accepted().body(adminService.uploadDocument(file, currentUser, documentType));
    }

    @GetMapping("/documents/stats")
    public ResponseEntity<AdminDocumentStatsResponse> getDocumentStats() {
        return ResponseEntity.ok(adminService.getDocumentStats());
    }

    @GetMapping("/documents/{document_id}")
    public ResponseEntity<AdminDocumentStatusResponse> getDocumentStatus(@PathVariable("document_id") UUID documentId) {
        return ResponseEntity.ok(adminService.getDocumentStatus(documentId));
    }

    @GetMapping("/documents")
    public ResponseEntity<List<AdminDocumentListItemResponse>> listDocuments() {
        return ResponseEntity.ok(adminService.listDocuments());
    }

    @DeleteMapping("/documents/{document_id}")
    public ResponseEntity<Void> deleteDocument(@PathVariable("document_id") UUID documentId) {
        adminService.deleteDocument(documentId);
        return ResponseEntity.noContent().build();
    }

    @GetMapping("/analytics/summary")
    public ResponseEntity<AnalyticsSummaryDTO> getAnalyticsSummary() {
        return ResponseEntity.ok(adminService.getAnalyticsSummary());
    }

    @GetMapping("/analytics/metrics")
    public ResponseEntity<AnalyticsMetricsDTO> getAnalyticsMetrics(
            @RequestParam(name = "since", required = false)
            @DateTimeFormat(iso = DateTimeFormat.ISO.DATE_TIME)
            LocalDateTime since
    ) {
        LocalDateTime effectiveSince = since == null ? LocalDateTime.now().minusDays(7) : since;
        return ResponseEntity.ok(analyticsService.getMetrics(effectiveSince));
    }

    @GetMapping("/analytics/student-activity")
    public ResponseEntity<List<StudentActivityPointRecord>> getStudentActivity(
            @RequestParam(name = "since", required = false)
            @DateTimeFormat(iso = DateTimeFormat.ISO.DATE_TIME)
            LocalDateTime since
    ) {
        LocalDateTime effectiveSince = since == null ? LocalDateTime.now().minusDays(7) : since;
        return ResponseEntity.ok(analyticsService.getStudentActivity(effectiveSince));
    }

    @PostMapping("/chunks/correction")
    public ResponseEntity<java.util.Map<String, Object>> editChunk(
            @RequestBody @jakarta.validation.Valid team._8.aitrogiang.dto.AdminEditChunkRequest request,
            Authentication authentication
    ) {
        User currentUser = (User) authentication.getPrincipal();
        boolean success = adminService.editChunkContent(request, currentUser);
        return ResponseEntity.ok(java.util.Map.of("success", success));
    }
}
