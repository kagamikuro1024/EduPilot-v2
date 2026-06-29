package team._8.aitrogiang.controller;

import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.Authentication;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.server.ResponseStatusException;
import team._8.aitrogiang.dto.AssignmentResponse;
import team._8.aitrogiang.dto.CreateAssignmentRequest;
import team._8.aitrogiang.model.Assignment;
import team._8.aitrogiang.model.User;
import team._8.aitrogiang.model.UserRole;
import team._8.aitrogiang.repository.AssignmentRepository;

import java.time.ZoneOffset;
import java.util.List;

@RestController
@RequestMapping("/api/v1/assignments")
@RequiredArgsConstructor
public class AssignmentController {

    private final AssignmentRepository assignmentRepository;

    @GetMapping
    public List<AssignmentResponse> getAssignments() {
        return assignmentRepository.findAllByOrderByDueDateAsc().stream()
                .map(AssignmentResponse::from)
                .toList();
    }

    @PostMapping
    public ResponseEntity<AssignmentResponse> createAssignment(
            @Valid @RequestBody CreateAssignmentRequest request,
            Authentication authentication
    ) {
        if (authentication == null
                || !(authentication.getPrincipal() instanceof User currentUser)
                || (currentUser.getRole() != UserRole.TA && currentUser.getRole() != UserRole.ADMIN)) {
            throw new ResponseStatusException(HttpStatus.FORBIDDEN, "Only TA or ADMIN can create assignments");
        }

        Assignment assignment = Assignment.builder()
                .title(request.title().trim())
                .description(request.description())
                .dueDate(request.dueDate().withOffsetSameInstant(ZoneOffset.UTC).toLocalDateTime())
                .latePenaltyRule(request.latePenaltyRule())
                .build();

        Assignment saved = assignmentRepository.save(assignment);
        return ResponseEntity.status(HttpStatus.CREATED).body(AssignmentResponse.from(saved));
    }
}
