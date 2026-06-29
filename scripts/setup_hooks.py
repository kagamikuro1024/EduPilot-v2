#!/usr/bin/env python3
import os
import sys
import platform
import stat
from pathlib import Path

def setup():
    is_windows = platform.system() == "Windows"
    print(f"[ai-log] Detecting OS: {platform.system()}")

    # Determine the python command to use in the hook
    # On Windows, 'python' is more common. On Linux, 'python3' is standard.
    python_cmd = "python" if is_windows else "python3"
    
    # Path to the hook file
    hook_path = Path(".git/hooks/pre-push")
    
    # Ensure .git/hooks directory exists (might not exist if git isn't initialized)
    if not Path(".git").exists():
        print("[ai-log] Error: .git directory not found. Please run this inside a git repository.")
        sys.exit(1)
    
    hook_path.parent.mkdir(parents=True, exist_ok=True)

    # Content of the hook
    # Standard git hooks are shell scripts even on Windows (run by Git Bash's sh)
    hook_content = f"""#!/bin/sh
# Submit AI logs to grading server before push
{python_cmd} scripts/submit_log.py
exit 0  # Never block push
"""

    with open(hook_path, "w", newline="\n") as f:
        f.write(hook_content)

    # Make the hook executable (crucial for Linux/macOS)
    if not is_windows:
        st = os.stat(hook_path)
        os.chmod(hook_path, st.st_mode | stat.S_IEXEC)

    print(f"[ai-log] Git pre-push hook installed at {hook_path}")

    # Create .ai-log directory
    log_dir = Path(".ai-log")
    log_dir.mkdir(exist_ok=True)
    (log_dir / ".gitkeep").touch()

    print("[ai-log] Setup complete. Configure AI_LOG_SERVER in your .env file.")

if __name__ == "__main__":
    setup()
