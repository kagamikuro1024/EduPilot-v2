from fastapi import Header, HTTPException, status
import os

# Internal secret for Admin API access
# TIP-006: Zero-Tolerance Security - No fallback allowed. Must fail fast if env is not set.
ADMIN_SECRET = os.getenv("ADMIN_TOKEN")

if not ADMIN_SECRET:
    raise RuntimeError("CRITICAL SECURITY ERROR: ADMIN_TOKEN environment variable is not set! System shutdown initiated.")

async def verify_admin_token(x_admin_token: str = Header(None)):
    """
    Dependency to verify the admin token in the request header.
    """
    if x_admin_token != ADMIN_SECRET:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Unauthorized: Admin access required."
        )
    return x_admin_token
