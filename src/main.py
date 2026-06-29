"""
FastAPI entrypoint for the AI Agent service.
Exposes REST endpoints for chat, health checks, and document ingestion.
Exposes diagnostic REST endpoints and starts the gRPC server.
Run: python -m src.main
"""

import logging
import asyncio
import sys
import os
from contextlib import asynccontextmanager

import uvicorn
from fastapi import FastAPI, HTTPException
from fastapi.responses import StreamingResponse
from pydantic import BaseModel
import grpc
from grpc import aio

from .agent import create_agent, run_agent_loop
from .config import APP_ENV, LOG_LEVEL, GRPC_PORT
from .grpc_server import serve_grpc_async
from .database.connection import init_db_pool, close_db_pool
from .database.cache_repo import init_semantic_cache_index, close_redis_client
from .observability import setup_tracing

# Import the upload router from its new location in data_pipeline
from data_pipeline.api import upload_router

# Ensure we can load the generated protobuf stubs
sys.path.insert(0, os.path.dirname(__file__))
try:
    import ai_service_pb2
    import ai_service_pb2_grpc
except ImportError:
    pass

from .logging_config import setup_btc_logging

setup_btc_logging()
logger = logging.getLogger(__name__)

# --- Lifespan: initialize shared resources ---

agent_client = None
grpc_server_instance = None
grpc_server_task = None

@asynccontextmanager
async def lifespan(app: FastAPI):
    """Startup/shutdown lifecycle for the FastAPI app."""
    global agent_client, grpc_server_instance, grpc_server_task
    
    try:
        # Initialize Observability (TD-06)
        setup_tracing()
        
        # Initialize Shared Database Pool
        await init_db_pool()
        
        # Initialize Semantic Cache Index (TIP-005)
        init_semantic_cache_index()
        
        agent_client = create_agent()
        logger.info("AI Agent client (Async) initialized successfully")
        
        # Warm up OpenAI client in the background (TIP-007)
        async def warm_up():
            try:
                await agent_client.chat.completions.create(
                    model="gpt-4o-mini",
                    messages=[{"role": "user", "content": "hi"}],
                    max_completion_tokens=1
                )
                logger.info("OpenAI client warm-up successful")
            except Exception as wu_err:
                logger.warning(f"OpenAI warm-up failed (non-critical): {wu_err}")
        
        asyncio.create_task(warm_up())
    except Exception as e:
        logger.warning(f"Startup initialization error: {e}")
        
    # Start Async gRPC server in the background for Chat services
    grpc_server_instance = await serve_grpc_async(agent_client)
    if grpc_server_instance:
        grpc_server_task = asyncio.create_task(grpc_server_instance.wait_for_termination())
        
    yield
    
    # Graceful shutdown
    if grpc_server_instance:
        logger.info("Shutting down async gRPC server")
        await grpc_server_instance.stop(grace=0)

    # Close Database Pool on shutdown
    await close_db_pool()
    
    # Close Redis client (TIP-005)
    close_redis_client()
        
    logger.info("Shutting down AI Agent service")


# --- FastAPI app ---
from fastapi.middleware.cors import CORSMiddleware

app = FastAPI(
    title="AI Agent Service (Compliance Version)",
    description=(
        "Python AI Agent for document ingestion and chat inference; "
        "gRPC per API_CONTRACT.md."
    ),
    version="2.1.0",
    lifespan=lifespan,
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Register the ingestion router
app.include_router(upload_router.router)
app.include_router(upload_router.admin_router)


# --- Request/Response models (Diagnostic) ---

class ChatRequest(BaseModel):
    thread_id: str = "default_thread"
    message: str

class ChatResponse(BaseModel):
    response: str

class CorrectionRequest(BaseModel):
    chunk_id: str
    new_content: str
    updated_by: str = "TA"

class AssignmentCreate(BaseModel):
    title: str
    description: str = ""
    due_date: str
    late_penalty_rule: str = ""


# --- Endpoints ---

@app.get("/api/v1/assignments")
async def get_assignments_endpoint():
    try:
        from .tools import _run_get_assignments
        import json
        result = _run_get_assignments()
        return json.loads(result)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/api/v1/assignments")
async def create_assignment_endpoint(request: AssignmentCreate):
    try:
        import psycopg2
        from .config import DATABASE_URL
        conn = psycopg2.connect(DATABASE_URL)
        cur = conn.cursor()
        cur.execute(
            """
            INSERT INTO assignments (title, description, due_date, late_penalty_rule)
            VALUES (%s, %s, %s, %s)
            RETURNING id
            """,
            (request.title, request.description, request.due_date, request.late_penalty_rule)
        )
        new_id = cur.fetchone()[0]
        conn.commit()
        conn.close()
        return {"success": True, "id": str(new_id)}
    except Exception as e:
        logger.error(f"Failed to create assignment: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/")
async def root():
    """Root endpoint — service status."""
    return {
        "status": "ok",
        "service": "ai-agent-service",
        "environment": APP_ENV,
        "grpc_port": GRPC_PORT
    }


@app.get("/health")
async def health():
    """Health check endpoint for container monitoring."""
    return {
        "status": "healthy",
        "agent_ready": agent_client is not None,
        "grpc_ready": grpc_server_instance is not None,
    }


@app.post("/api/v1/diagnostic/chat", response_model=ChatResponse)
async def chat_diagnostic(request: ChatRequest):
    """
    Standard chat endpoint.
    Sends a message to the AI Agent and returns the FULL response.
    Diagnostic unary chat endpoint.
    """
    if agent_client is None:
        raise HTTPException(status_code=503, detail="Agent not initialized.")

    try:
        logger.info(f"Chat request received: {request.message[:100]}...")
        logger.info(f"Diagnostic Chat request: {request.message[:100]}...")
        result = await run_agent_loop(
            client=agent_client,
            user_input=request.message,
            max_turns=10,
        )
        return ChatResponse(response=result)
    except Exception as e:
        logger.error(f"Agent error: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/api/v1/chunks/correction")
async def update_chunk_correction(request: CorrectionRequest):
    """
    TA manual correction endpoint.
    Updates the knowledge base with human-verified content.
    """
    try:
        from .database.vector_repo import update_chunk_content
        from openai import AsyncOpenAI
        from .config import OPENAI_API_KEY

        # 1. Generate new embedding
        client = AsyncOpenAI(api_key=OPENAI_API_KEY)
        emb_resp = await client.embeddings.create(
            input=[request.new_content],
            model="text-embedding-3-small"
        )
        new_vector = emb_resp.data[0].embedding

        # 2. Update DB
        success = await update_chunk_content(
            chunk_id=request.chunk_id,
            new_content=request.new_content,
            new_embedding=new_vector,
            updated_by=request.updated_by
        )

        if not success:
            raise HTTPException(status_code=404, detail="Chunk not found.")

        return {"success": True, "message": "Knowledge base updated successfully."}
    except Exception as e:
        logger.error(f"Correction error: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/api/v1/diagnostic/stream")
async def chat_stream_diagnostic(message: str, thread_id: str = "diag_thread"):
    """
    Server-Sent Events (SSE) streaming chat endpoint.
    Connects to the local async gRPC stream and pushes events to the HTTP client.
    Diagnostic SSE chat endpoint using the local gRPC server.
    """
    if not ai_service_pb2_grpc:
        raise HTTPException(status_code=500, detail="gRPC files missing. Compile proto first.")
        
    async def sse_generator():
        # Connect to the local gRPC server
        async with aio.insecure_channel(f'localhost:{GRPC_PORT}') as channel:
            stub = ai_service_pb2_grpc.AIThreadServiceStub(channel)
            grpc_req = ai_service_pb2.AIRequest(
                thread_id=thread_id,
                current_message=message,
                thread_title="Diagnostic Thread"
            )
            
            try:
                async for response in stub.StreamAIResponse(grpc_req):
                    if response.chunk:
                        yield f"data: {response.chunk}\n\n"
                    if response.is_finished:
                        yield "event: done\ndata: \n\n"
                        break
            except grpc.aio.AioRpcError as e:
                logger.error(f"gRPC stream error: {e.details()}")
                yield f"event: error\ndata: {e.details()}\n\n"
            except Exception as e:
                logger.error(f"Unexpected error: {e}")
                yield "event: error\ndata: Internal Server Error\n\n"

    return StreamingResponse(sse_generator(), media_type="text/event-stream")


# --- Main ---

if __name__ == "__main__":
    host_binding = os.getenv("APP_HOST", "0.0.0.0")  # nosec B104
    uvicorn.run(
        "src.main:app",
        host=host_binding,
        port=8000,
        reload=(APP_ENV == "development"),
        timeout_keep_alive=60,
    )
