package team._8.aitrogiang.constant;

import java.util.UUID;

/**
 * Global system constants to avoid hardcoding across the application.
 */
public class SystemConstants {
    
    /**
     * Fixed UUID for the AI System User (AI_TUTOR).
     * Must match the ID inserted in V8 migration script.
     */
    public static final UUID SYSTEM_AI_ID = UUID.fromString("00000000-0000-0000-0000-000000000000");

    private SystemConstants() {
        // Prevent instantiation
    }
}
