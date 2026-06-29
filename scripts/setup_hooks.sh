#!/bin/bash
# Install git hooks for AI log capturing and submission
set -e

# --- 1. PRE-COMMIT HOOK (Logging only) ---
COMMIT_HOOK=".git/hooks/pre-commit"
cat > "$COMMIT_HOOK" << 'EOF'
#!/bin/sh
if py --version >/dev/null 2>&1; then
    PY_CMD="py"
elif python --version >/dev/null 2>&1; then
    PY_CMD="python"
elif python3 --version >/dev/null 2>&1; then
    PY_CMD="python3"
else
    PY_CMD=""
fi

if [ -n "$PY_CMD" ]; then
    echo '{"prompt": "GIT COMMIT", "response": "User is committing changes to local repository"}' | AI_TOOL_NAME=git "$PY_CMD" scripts/log_hook.py >/dev/null 2>&1 || true
fi
exit 0
EOF
chmod +x "$COMMIT_HOOK"
echo "[ai-log] Git pre-commit hook installed (Local Logging)."

# --- 2. PRE-PUSH HOOK (Logging + Submission) ---
PUSH_HOOK=".git/hooks/pre-push"
cat > "$PUSH_HOOK" << 'EOF'
#!/bin/sh
if py --version >/dev/null 2>&1; then
    PY_CMD="py"
elif python --version >/dev/null 2>&1; then
    PY_CMD="python"
elif python3 --version >/dev/null 2>&1; then
    PY_CMD="python3"
else
    PY_CMD=""
fi

if [ -n "$PY_CMD" ]; then
    echo '{"prompt": "GIT PUSH", "response": "User is pushing commits to remote repository"}' | AI_TOOL_NAME=git "$PY_CMD" scripts/log_hook.py >/dev/null 2>&1 || true
    "$PY_CMD" scripts/submit_log.py || true
fi
exit 0
EOF
chmod +x "$PUSH_HOOK"
echo "[ai-log] Git pre-push hook installed (Logging & Submission)."

# --- 3. Directory Setup ---
mkdir -p .ai-log
touch .ai-log/.gitkeep

echo "[ai-log] Setup complete. Manual Git actions (Commit/Push) will now be tracked in session.jsonl."
