import os
import stat

hooks_dir = ".git/hooks"
os.makedirs(hooks_dir, exist_ok=True)

pre_commit_content = """#!/bin/sh
python3 scripts/log_hook.py || python scripts/log_hook.py || true
exit 0
"""

pre_push_content = """#!/bin/sh
python3 scripts/log_hook.py || python scripts/log_hook.py || true
python3 scripts/submit_log.py || python scripts/submit_log.py || true
exit 0
"""

for name, content in [("pre-commit", pre_commit_content), ("pre-push", pre_push_content)]:
    path = os.path.join(hooks_dir, name)
    with open(path, "w", newline="\n") as f:
        f.write(content)
    os.chmod(path, stat.S_IRWXU | stat.S_IRGRP | stat.S_IXGRP | stat.S_IROTH | stat.S_IXOTH)
    print(f"Created: {path}")

print("Done.")
