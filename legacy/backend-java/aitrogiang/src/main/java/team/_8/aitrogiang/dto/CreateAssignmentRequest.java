package team._8.aitrogiang.dto;

import com.fasterxml.jackson.annotation.JsonProperty;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;

import java.time.OffsetDateTime;

public record CreateAssignmentRequest(
        @NotBlank String title,
        String description,
        @JsonProperty("due_date") @NotNull OffsetDateTime dueDate,
        @JsonProperty("late_penalty_rule") String latePenaltyRule
) {
}
