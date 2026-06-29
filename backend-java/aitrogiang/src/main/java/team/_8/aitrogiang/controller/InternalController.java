package team._8.aitrogiang.controller;

import lombok.RequiredArgsConstructor;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PatchMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;
import team._8.aitrogiang.dto.DocumentStatusCallbackRequest;
import team._8.aitrogiang.model.DocumentStatus;
import team._8.aitrogiang.service.AdminService;

import java.util.Locale;
import java.util.Map;
import java.util.UUID;

@RestController
@RequestMapping("/api/v1/internal")
@RequiredArgsConstructor
public class InternalController {

    private final AdminService adminService;

    @Value("${app.internal.token:}")
    private String internalToken;

    @PatchMapping("/documents/{id}/status")
    public ResponseEntity<?> updateDocumentStatus(
            @PathVariable("id") UUID documentId,
            @RequestBody DocumentStatusCallbackRequest body,
            @RequestHeader(value = "X-Internal-Token", required = false) String token
    ) {
        if (!internalToken.isBlank() && !internalToken.equals(token)) {
            return ResponseEntity.status(HttpStatus.FORBIDDEN)
                    .body(Map.of("error", "FORBIDDEN", "message", "Invalid internal callback token"));
        }

        if (body == null || body.getStatus() == null || body.getStatus().isBlank()) {
            return ResponseEntity.badRequest()
                    .body(Map.of("error", "VALIDATION_ERROR", "message", "Status is required"));
        }

        try {
            DocumentStatus targetStatus = DocumentStatus.valueOf(body.getStatus().toUpperCase(Locale.ROOT));
            adminService.updateDocumentStatus(documentId, targetStatus);
            return ResponseEntity.noContent().build();
        } catch (IllegalArgumentException ex) {
            return ResponseEntity.badRequest()
                    .body(Map.of("error", "VALIDATION_ERROR", "message", "Invalid status value: " + body.getStatus()));
        }
    }
}
