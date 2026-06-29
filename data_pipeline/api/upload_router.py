"""
Module: data_pipeline.api.upload_router
Description: Controller for handling document upload requests.
             Implements Validation Guardrails (format, size) and offloads processing tasks to the background worker.
"""

import os
import shutil
import uuid
import logging
import json
from fastapi import APIRouter, UploadFile, File, HTTPException, BackgroundTasks, Form
from typing import Optional
from pydantic import BaseModel
from fastapi import Depends

# Internal imports updated for the new structure
from data_pipeline.workers.ingestion_worker import process_document_task
from src.api.auth import verify_admin_token

logger = logging.getLogger(__name__)

router = APIRouter(
    prefix="/api/v1/documents",
    tags=["documents"]
)

# Directory for temporary file storage before processing
UPLOAD_DIR = "data/tmp"
os.makedirs(UPLOAD_DIR, exist_ok=True)

# Validation constraints
MAX_FILE_SIZE = 50 * 1024 * 1024  # 50MB
ALLOWED_EXTENSIONS = {".pdf", ".docx", ".txt"}

@router.post("/upload", status_code=202)
async def upload_document(
    background_tasks: BackgroundTasks,
    file: UploadFile = File(...),
    metadata: Optional[str] = Form(None)
):
    """
    Document ingestion endpoint:
    1. Validates file format and size.
    2. Stores the file temporarily on disk.
    3. Triggers the background ETL process.
    """
    
    # 1. Validation: File extension check
    extension = os.path.splitext(file.filename)[1].lower()
    if extension not in ALLOWED_EXTENSIONS:
        raise HTTPException(
            status_code=400, 
            detail="Unsupported file type"
        )

    # 2. Validation: File size check (max 50MB)
    file.file.seek(0, os.SEEK_END)
    file_size = file.file.tell()
    file.file.seek(0)
    
    if file_size > MAX_FILE_SIZE:
        raise HTTPException(
            status_code=413, 
            detail=f"File too large ({file_size / 1024 / 1024:.1f}MB). Maximum limit is 50MB."
        )

    # 3. Secure temporary storage
    # Uses UUID to prevent name collisions
    temp_filename = f"{uuid.uuid4()}{extension}"
    temp_path = os.path.join(UPLOAD_DIR, temp_filename)
    
    try:
        os.makedirs(UPLOAD_DIR, exist_ok=True)
        with open(temp_path, "wb") as buffer:
            shutil.copyfileobj(file.file, buffer)
            
        logger.info(f"Temporary file saved: {temp_path}")
    except Exception as e:
        logger.error(f"Error saving temporary file {temp_path}: {e}")
        raise HTTPException(status_code=500, detail=f"Failed to save temporary file: {e}")

    # 4. Offload to background worker for ETL processing
    metadata_dict = {}
    if metadata:
        try:
            metadata_dict = json.loads(metadata)
        except Exception:
            logger.warning("Failed to parse metadata JSON. Proceeding with empty metadata.")
            
    background_tasks.add_task(
        process_document_task, 
        temp_path, 
        file.filename, 
        metadata_dict
    )

    return {
        "status": "processing",
        "message": "Document is being processed in the background.",
        "source_uri": file.filename,
        "request_id": str(uuid.uuid4())
    }

# --- Admin Admin Ingestion Route (TIP-002) ---

admin_router = APIRouter(
    prefix="/api/admin",
    tags=["admin"]
)

@admin_router.post("/documents/upload", status_code=200, dependencies=[Depends(verify_admin_token)])
async def admin_upload_document(
    background_tasks: BackgroundTasks,
    file: UploadFile = File(...),
    metadata: Optional[str] = Form(None)
):
    """
    Compliance target for TIP-002: Vector Upload API.
    Calls the same core logic but returns 200 OK as per spec.
    """
    return await upload_document(background_tasks, file, metadata)

# --- HITL API (Human-in-the-loop Knowledge Correction) ---

class ChunkUpdate(BaseModel):
    new_content: str
    reason: Optional[str] = None

@router.get("/{doc_id}/chunks")
async def get_document_chunks(doc_id: str):
    """
    HITL API: Returns the list of document chunks for manual verification.
    """
    from src.database.vector_repo import get_chunks_by_document_id
    try:
        chunks = await get_chunks_by_document_id(doc_id)
        return {
            "document_id": doc_id, 
            "total_chunks": len(chunks),
            "chunks": chunks
        }
    except Exception as e:
        logger.error(f"Error in get_chunks for {doc_id}: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@router.patch("/chunks/{chunk_id}")
async def patch_chunk(chunk_id: str, update: ChunkUpdate):
    """
    HITL API: Corrects knowledge base content.
    - Triggers re-embedding for the corrected text.
    - Synchronously updates content, hash, and vector in the database.
    """
    from src.database.vector_repo import update_chunk_content
    from data_pipeline.pipeline.embedding import generate_embeddings
    
    try:
        # 1. Re-embedding: Generate new vector for corrected content
        embeddings = await generate_embeddings([update.new_content])
        if not embeddings:
            raise HTTPException(status_code=500, detail="Failed to generate embeddings for corrected content.")
        
        # 2. Atomic Database Update with Audit Trail
        success = await update_chunk_content(
            chunk_id=chunk_id,
            new_content=update.new_content,
            new_embedding=embeddings[0],
            updated_by="TA"
        )
        
        if not success:
            raise HTTPException(status_code=404, detail="Chunk ID not found for update.")
            
        return {
            "status": "success",
            "message": "Knowledge content and corresponding vector updated successfully.",
            "chunk_id": chunk_id
        }
    except Exception as e:
        logger.error(f"Error in patch_chunk for {chunk_id}: {e}")
        raise HTTPException(status_code=500, detail=str(e))
