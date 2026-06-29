import json
import subprocess
import os

def log_interaction(prompt, response):
    log_data = {"prompt": prompt, "response": response}
    log_json = json.dumps(log_data)
    
    # Simulate the pipe: echo '...' | python scripts/log_hook.py
    try:
        # On Windows, we might need a different way to pipe
        cmd = 'set AI_TOOL_NAME=antigravity&& python scripts/log_hook.py'
        process = subprocess.Popen(cmd, shell=True, stdin=subprocess.PIPE, text=True)
        process.communicate(input=log_json)
    except Exception as e:
        print(f"Logging failed: {e}")

# Example usage for the current turn
# log_interaction("User requested stabilization...", "I have updated docker-compose, added exception handlers...")
