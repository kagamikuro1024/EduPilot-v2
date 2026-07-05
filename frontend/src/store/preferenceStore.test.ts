import { describe, it, expect, beforeEach, vi } from "vitest";
import { usePreferenceStore } from "./preferenceStore";

describe("usePreferenceStore", () => {
  beforeEach(() => {
    // Reset Zustand store state before each test
    usePreferenceStore.setState({
      preferences: {
        theme: "SYSTEM",
        font_size: "DEFAULT",
        reduce_motion: false,
        default_student_page: "ASSIGNMENTS",
      },
    });
    if (typeof window !== "undefined") {
      window.matchMedia = vi.fn().mockImplementation((query) => ({
        matches: false,
        media: query,
        onchange: null,
        addListener: vi.fn(),
        removeListener: vi.fn(),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
      }));
    }
    vi.clearAllMocks();
  });

  it("should initialize with default preferences", () => {
    const state = usePreferenceStore.getState();
    expect(state.preferences.theme).toBe("SYSTEM");
    expect(state.preferences.font_size).toBe("DEFAULT");
    expect(state.preferences.reduce_motion).toBe(false);
    expect(state.preferences.default_student_page).toBe("ASSIGNMENTS");
  });

  it("should update theme, font size, and reduced motion preferences", () => {
    const state = usePreferenceStore.getState();
    
    state.setPreferences({ theme: "DARK", font_size: "LARGE", reduce_motion: true });
    
    const updated = usePreferenceStore.getState().preferences;
    expect(updated.theme).toBe("DARK");
    expect(updated.font_size).toBe("LARGE");
    expect(updated.reduce_motion).toBe(true);
  });

  it("should update default student page preference", () => {
    const state = usePreferenceStore.getState();
    
    state.setPreferences({ default_student_page: "CHAT" });
    
    const updated = usePreferenceStore.getState().preferences;
    expect(updated.default_student_page).toBe("CHAT");
  });

  it("should synchronize preferences to document.documentElement", () => {
    if (typeof document !== "undefined") {
      const state = usePreferenceStore.getState();
      
      state.setPreferences({ theme: "DARK", font_size: "SMALL", reduce_motion: true });
      
      expect(document.documentElement.classList.contains("dark")).toBe(true);
      expect(document.documentElement.getAttribute("data-font-size")).toBe("SMALL");
      expect(document.documentElement.classList.contains("reduce-motion")).toBe(true);
    }
  });
});
