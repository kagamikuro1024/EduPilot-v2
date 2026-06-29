import os

def generate_tree(start_path, indent_str='    ', ignore_dirs=None, output_file=None):
    if ignore_dirs is None:
        ignore_dirs = {'.git', '.venv', 'node_modules', '.gradle', 'build', '__pycache__', '.pytest_cache', '.ai-log', '.gemini', '.cursor'}
    
    lines = []
    lines.append(f"# Project Directory Scan\n")
    lines.append(f"Generated recursively, ignoring build and virtual environment artifacts.\n\n```text")
    
    def walk(path, prefix=""):
        try:
            items = sorted(os.listdir(path))
        except PermissionError:
            return

        # Filter ignored directories
        filtered_items = []
        for item in items:
            if item in ignore_dirs:
                continue
            filtered_items.append(item)

        for i, name in enumerate(filtered_items):
            full_path = os.path.join(path, name)
            is_last = (i == len(filtered_items) - 1)
            connector = "└── " if is_last else "├── "
            lines.append(f"{prefix}{connector}{name}")
            
            if os.path.isdir(full_path):
                extension = "    " if is_last else "│   "
                walk(full_path, prefix + extension)

    lines.append(os.path.basename(os.path.abspath(start_path)) or ".")
    walk(start_path)
    lines.append("```\n")
    
    with open(output_file, 'w', encoding='utf-8') as f:
        f.write("\n".join(lines))
    print(f"Successfully scanned and saved to {output_file}")

if __name__ == "__main__":
    generate_tree('.', output_file='DIRECTORY_REPORT_NEW.md')
