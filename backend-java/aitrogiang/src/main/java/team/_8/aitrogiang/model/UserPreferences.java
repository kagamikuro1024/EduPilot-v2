package team._8.aitrogiang.model;

import jakarta.persistence.*;
import lombok.*;
import java.time.LocalDateTime;
import java.util.UUID;

@Entity
@Table(name = "user_preferences")
@Getter
@Setter
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class UserPreferences {

    @Id
    @Column(name = "user_id")
    private UUID userId;

    @OneToOne(fetch = FetchType.LAZY)
    @MapsId
    @JoinColumn(name = "user_id")
    private User user;

    @Enumerated(EnumType.STRING)
    @Column(nullable = false)
    private ThemePreference theme;

    @Enumerated(EnumType.STRING)
    @Column(name = "font_size", nullable = false)
    private FontSizePreference fontSize;

    @Column(name = "reduce_motion", nullable = false)
    private boolean reduceMotion;

    @Enumerated(EnumType.STRING)
    @Column(name = "default_student_page", nullable = false)
    private DefaultPagePreference defaultStudentPage;

    @Column(name = "updated_at", nullable = false)
    private LocalDateTime updatedAt;

    @PrePersist
    @PreUpdate
    protected void onUpdate() {
        updatedAt = LocalDateTime.now();
    }
}
