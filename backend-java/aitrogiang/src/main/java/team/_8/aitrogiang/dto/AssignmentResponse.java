package team._8.aitrogiang.dto;

import com.fasterxml.jackson.annotation.JsonProperty;
import team._8.aitrogiang.model.Assignment;

import java.time.LocalDateTime;
import java.util.UUID;

public record AssignmentResponse(
        UUID id,
        String title,
        String description,
        @JsonProperty("due_date") LocalDateTime dueDate,
        @JsonProperty("late_penalty_rule") String latePenaltyRule
) {
    public static AssignmentResponse from(Assignment assignment) {
        return new AssignmentResponse(
                assignment.getId(),
                assignment.getTitle(),
                assignment.getDescription(),
                assignment.getDueDate(),
                assignment.getLatePenaltyRule()
        );
    }
}
