package team._8.aitrogiang.dto;

import lombok.*;
import team._8.aitrogiang.model.ThemePreference;
import team._8.aitrogiang.model.FontSizePreference;
import team._8.aitrogiang.model.DefaultPagePreference;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class UserPreferencesResponse {
    private ThemePreference theme;
    private FontSizePreference font_size;
    private boolean reduce_motion;
    private DefaultPagePreference default_student_page;
}
