import type { Config } from "tailwindcss";

export default {
  content: [
    "./src/pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/components/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/app/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    extend: {
      colors: {
        primary: {
          DEFAULT: "var(--primary)",
          foreground: "var(--primary-foreground)",
        },
        secondary: {
          DEFAULT: "var(--secondary)",
          foreground: "var(--secondary-foreground)",
        },
        background: "var(--background)",
        foreground: "var(--foreground)",

        // === Thêm tokens Cohere (via CSS variables) ===
        "cohere-primary":   "var(--cohere-primary)",
        "cohere-black":     "var(--cohere-black)",
        "ink":              "var(--cohere-ink)",
        "deep-green":       "var(--cohere-deep-green)",
        "dark-navy":        "var(--cohere-dark-navy)",
        "canvas":           "var(--cohere-canvas)",
        "soft-stone":       "var(--cohere-soft-stone)",
        "pale-green":       "var(--cohere-pale-green)",
        "pale-blue":        "var(--cohere-pale-blue)",
        "hairline":         "var(--cohere-hairline)",
        "border-light":     "var(--cohere-border-light)",
        "card-border":      "var(--cohere-card-border)",
        "muted":            "var(--cohere-muted)",
        "slate-cohere":     "var(--cohere-slate)",
        "body-muted":       "var(--cohere-body-muted)",
        "action-blue":      "var(--cohere-action-blue)",
        "focus-blue":       "var(--cohere-focus-blue)",
        "coral":            "var(--cohere-coral)",
        "coral-soft":       "var(--cohere-coral-soft)",
        "form-focus":       "var(--cohere-form-focus)",
        "on-primary":       "var(--cohere-on-primary)",
        "error-cohere":     "var(--cohere-error)",
      },
      fontFamily: {
        "cohere-display": ["var(--font-space-grotesk)", "CohereText", "Space Grotesk", "Inter", "ui-sans-serif", "system-ui"],
        "cohere-body":    ["var(--font-inter)", "Unica77 Cohere Web", "Inter", "Arial", "ui-sans-serif", "system-ui"],
        "cohere-mono":    ["CohereMono", "Arial", "ui-monospace", "system-ui"],
      },
      borderRadius: {
        "cohere-xs":   "4px",
        "cohere-sm":   "8px",
        "cohere-md":   "16px",
        "cohere-lg":   "22px",
        "cohere-xl":   "30px",
        "cohere-pill": "32px",
      },
      spacing: {
        "section": "80px",
        "xxl":     "32px",
        "xl-c":    "24px",
        "lg-c":    "16px",
        "md-c":    "12px",
        "sm-c":    "8px",
        "xs-c":    "6px",
        "xxs-c":   "2px",
      },
    },
  },
  plugins: [],
} satisfies Config;
