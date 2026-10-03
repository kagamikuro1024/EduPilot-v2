import { defineConfig, globalIgnores } from "eslint/config";
import nextVitals from "eslint-config-next/core-web-vitals";
import nextTs from "eslint-config-next/typescript";
import ep from "./eslint-rules/ep.mjs";

// 7 luật ep/* (SRS FEAT-ui-foundation 4.2). Mỗi luật một khối để ngoại lệ đường dẫn không đè lên nhau.
const all = ["src/**/*.{ts,tsx}"];
const only = (name, ignores = []) => ({ files: all, ignores, plugins: { ep }, rules: { [`ep/${name}`]: "error" } });

export default defineConfig([
  ...nextVitals,
  ...nextTs,
  globalIgnores([".next/**", "out/**", "build/**", "next-env.d.ts", "playwright-report/**", "test-results/**"]),
  only("no-raw-fetch", ["src/shared/data/**"]),
  only("no-native-dialogs"),
  only("no-custom-spinner", ["src/shared/ui/**"]),
  only("no-raw-table", ["src/shared/ui/DataTable.tsx", "src/shared/ui/DataTableVirtual.tsx"]),
  only("no-token-in-storage"),
  only("no-tailwind"),
  only("no-literal-color-in-style", ["src/shared/styles/**"]),
]);
