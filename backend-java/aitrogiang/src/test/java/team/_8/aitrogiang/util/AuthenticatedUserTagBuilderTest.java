package team._8.aitrogiang.util;

import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

class AuthenticatedUserTagBuilderTest {

    @Test
    void buildsSignedTagWithoutPlaintextStudentIdentity() {
        String tag = AuthenticatedUserTagBuilder.buildTag(
                "70cde17c-1328-46a8-a097-bb421c10592e",
                "sv260115",
                "student",
                "test-secret"
        );

        assertNotNull(tag);
        assertTrue(tag.startsWith(AuthenticatedUserTagBuilder.PREFIX));
        assertTrue(tag.contains("."));
        assertFalse(tag.contains("SV260115"));
    }

    @Test
    void refusesToBuildWithoutSigningSecret() {
        assertNull(AuthenticatedUserTagBuilder.buildTag("user-1", "SV260115", "STUDENT", ""));
    }
}
