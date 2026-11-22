# D06-05/06/12: UI provider bản thật (ALLOW_REAL=true, không gọi mạng thật)
gate("THREADS_PRECHECK", False, [C]) if False else None
sql(f"UPDATE llm_providers SET api_key_enc=NULL WHERE id='{PID}'"); dprov("decision-fake"); time.sleep(3)
r = subprocess.run(["bash", "-c", "cd ~/Documents/TA_Agent_qc9s/docs/regression/evidence-9s/s106 && ./runpw.sh qc-real-105-ui.spec.ts 'D06-05-06-12' 150 2>&1 | grep -E 'VERDICT|Error|✘|expect|Timeout' | cut -c1-2500"], capture_output=True, text=True); print(r.stdout[-3500:], flush=True)
sql(f"UPDATE llm_providers SET api_key_enc=NULL WHERE id='{PID}'")
