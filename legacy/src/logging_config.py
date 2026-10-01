import logging
import json
import os
from logging.handlers import RotatingFileHandler
from datetime import datetime, timezone

class JSONBtcFormatter(logging.Formatter):
    """
    BTC-compliant JSON formatter.
    Ensures every log line is a valid JSON object.
    """
    def format(self, record):
        log_entry = {
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "level": record.levelname,
            "module": record.module,
            "message": record.getMessage(),
        }
        
        # Add common BTC metadata if present in extra
        for field in ["session_id", "event_type", "model", "student_id", "violation_reason", "event", "tool_name", "params", "status"]:
            if hasattr(record, field):
                log_entry[field] = getattr(record, field)
                
        return json.dumps(log_entry, ensure_ascii=False)

def setup_btc_logging(log_file="session.jsonl", log_level=logging.INFO):
    """
    Sets up a RotatingFileHandler for BTC-compliant logging.
    Max size: 50MB, keeps 3 backups.
    """
    # Ensure directory exists if path is not root
    log_dir = os.path.dirname(log_file)
    if log_dir:
        os.makedirs(log_dir, exist_ok=True)
        
    # TIP-005: Use a specific logger name to avoid global side effects
    logger = logging.getLogger("aitrogiang_app")
    logger.setLevel(log_level)
    logger.propagate = False # Prevent logs from bubbling up to root and duplicating
    
    if not logger.handlers:
        # File Handler
        handler = RotatingFileHandler(
            log_file, 
            maxBytes=50 * 1024 * 1024, 
            backupCount=3,
            encoding='utf-8'
        )
        handler.setFormatter(JSONBtcFormatter())
        logger.addHandler(handler)
        
        # Console Handler
        console_handler = logging.StreamHandler()
        console_handler.setFormatter(JSONBtcFormatter())
        logger.addHandler(console_handler)

    logging.info("BTC-compliant logging initialized.")
