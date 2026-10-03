// 7 luật `ep/*` của EduPilot (docs/specs/FEAT-ui-foundation/SRS.md 4.2). Không thêm thư viện; không có luật thứ 8.
// Ngoại lệ theo đường dẫn khai báo ở eslint.config.mjs (mỗi luật một khối `files`/`ignores`).

const STORAGE_KEY = /token|jwt|access|refresh|secret|password/i;
const COLOR = /#[0-9a-f]{3,8}\b|\b(?:rgba?|hsla?|oklch)\(/i;

const isGlobalCall = (node, names) =>
  (node.callee.type === "Identifier" && names.includes(node.callee.name)) ||
  (node.callee.type === "MemberExpression" &&
    node.callee.object.type === "Identifier" &&
    ["window", "globalThis"].includes(node.callee.object.name) &&
    node.callee.property.type === "Identifier" &&
    names.includes(node.callee.property.name));

const text = (node) => {
  if (!node) return "";
  if (node.type === "Literal") return String(node.value);
  if (node.type === "TemplateLiteral") return node.quasis.map((q) => q.value.cooked ?? "").join("");
  return "";
};

const rule = (message, create) => ({ meta: { type: "problem", schema: [], messages: { m: message } }, create });

const ep = {
  rules: {
    "no-raw-fetch": rule("Không gọi mạng trực tiếp ({{what}}); dùng apiClient ở shared/data.", (ctx) => ({
      CallExpression(n) {
        if (isGlobalCall(n, ["fetch"])) ctx.report({ node: n, messageId: "m", data: { what: "fetch" } });
      },
      NewExpression(n) {
        if (n.callee.type === "Identifier" && n.callee.name === "XMLHttpRequest") ctx.report({ node: n, messageId: "m", data: { what: "XMLHttpRequest" } });
      },
      ImportDeclaration(n) {
        if (n.source.value === "axios") ctx.report({ node: n, messageId: "m", data: { what: "axios" } });
      },
    })),
    "no-native-dialogs": rule("Không dùng hộp thoại của trình duyệt ({{what}}); dùng ConfirmIrreversible / Dialog.", (ctx) => ({
      CallExpression(n) {
        if (isGlobalCall(n, ["confirm", "alert", "prompt"])) {
          ctx.report({ node: n, messageId: "m", data: { what: n.callee.type === "Identifier" ? n.callee.name : n.callee.property.name } });
        }
      },
    })),
    "no-custom-spinner": rule("Không tự làm spinner ({{what}}); dùng <PageState> / trạng thái tải của primitive.", (ctx) => {
      const BAD = new Set(["Spinner", "FullPageSpinner", "LoadingSpinner"]);
      const check = (n, name) => BAD.has(name) && ctx.report({ node: n, messageId: "m", data: { what: name } });
      return {
        Identifier: (n) => check(n, n.name),
        JSXIdentifier: (n) => check(n, n.name),
      };
    }),
    "no-raw-table": rule("Không dùng <table> thô; dùng DataTable.", (ctx) => ({
      JSXOpeningElement(n) {
        if (n.name.type === "JSXIdentifier" && n.name.name === "table") ctx.report({ node: n, messageId: "m" });
      },
    })),
    "no-token-in-storage": rule("Không ghi token/bí mật vào storage hay cookie ({{what}}); token chỉ ở bộ nhớ (tokenStore).", (ctx) => ({
      CallExpression(n) {
        const c = n.callee;
        if (c.type === "MemberExpression" && c.object.type === "Identifier" && ["localStorage", "sessionStorage"].includes(c.object.name) &&
            c.property.type === "Identifier" && c.property.name === "setItem" && STORAGE_KEY.test(text(n.arguments[0]))) {
          ctx.report({ node: n, messageId: "m", data: { what: `${c.object.name}.setItem` } });
        }
      },
      AssignmentExpression(n) {
        const l = n.left;
        if (l.type === "MemberExpression" && l.object.type === "Identifier" && l.object.name === "document" &&
            l.property.type === "Identifier" && l.property.name === "cookie" && STORAGE_KEY.test(text(n.right))) {
          ctx.report({ node: n, messageId: "m", data: { what: "document.cookie" } });
        }
      },
    })),
    "no-tailwind": rule("Không dùng Tailwind (D53); dùng CSS Modules + token --ep-*.", (ctx) => ({
      ImportDeclaration(n) {
        const s = String(n.source.value);
        if (s === "tailwindcss" || s.startsWith("tailwindcss/") || s.startsWith("@tailwindcss/")) ctx.report({ node: n, messageId: "m" });
      },
    })),
    "no-literal-color-in-style": rule("Không viết màu trong style={{}}; dùng token var(--ep-*).", (ctx) => ({
      JSXAttribute(n) {
        if (n.name.name !== "style" || !n.value || n.value.type !== "JSXExpressionContainer") return;
        const walk = (x) => {
          if (!x || typeof x.type !== "string") return;
          if ((x.type === "Literal" || x.type === "TemplateLiteral") && COLOR.test(text(x))) ctx.report({ node: x, messageId: "m" });
          for (const k of Object.keys(x)) {
            if (k === "parent") continue;
            const v = x[k];
            if (Array.isArray(v)) v.forEach(walk);
            else if (v && typeof v === "object") walk(v);
          }
        };
        walk(n.value.expression);
      },
    })),
  },
};

export default ep;
