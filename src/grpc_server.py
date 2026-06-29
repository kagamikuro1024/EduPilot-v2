import logging
import asyncio
import grpc
from grpc import aio
import sys
import os

# Add the current directory to sys.path to allow importing generated files correctly
sys.path.insert(0, os.path.dirname(__file__))

try:
    import ai_service_pb2
    import ai_service_pb2_grpc
except ImportError:
    logging.warning("gRPC compiled files not found. Please run the protoc command.")
    ai_service_pb2 = None
    ai_service_pb2_grpc = None

from .agent import run_agent_loop_stream
from .config import GRPC_PORT
from .document_callback import run_ingestion_with_callback
from .guardrails import (
    classify_and_guard,
    classify_channel_intent,
    ChannelIntent,
    IntentType,
    classification_from_java_preflight_tags,
    normalize_classifier_channel_hint,
)

logger = logging.getLogger(__name__)


def _channel_hint_from_grpc_tags(tags) -> str:
    """Java adds tags like channel:FORUM / channel:CHAT → PUBLIC / PRIVATE classifier hints."""
    if not tags:
        return "PRIVATE"
    for tag in tags:
        if tag.startswith("channel:"):
            raw = tag[len("channel:") :].strip()
            return normalize_classifier_channel_hint(raw)
    return "PRIVATE"


class AIThreadServicer(ai_service_pb2_grpc.AIThreadServiceServicer if ai_service_pb2_grpc else object):
    def __init__(self, agent_client):
        self.agent_client = agent_client

    async def StreamAIResponse(self, request, context: grpc.aio.ServicerContext):
        if not self.agent_client:
            yield ai_service_pb2.AIResponse(chunk="Agent client not initialized", is_finished=True)
            return

        try:
            logger.info(f"gRPC StreamAIResponse: Thread ID: {request.thread_id}")

            # 1. Guardrails & Intent — skip duplicate LLM if Java sent java_preflight tag (ClassifyIntent already ran)
            preflight = classification_from_java_preflight_tags(request.tags)
            if preflight is not None:
                classification = preflight
                logger.info("StreamAIResponse: using java_preflight hint (skipping classify_and_guard)")
            else:
                # Single LLM call for guard + intent. TTFT target < 1.2s
                stream_hint = _channel_hint_from_grpc_tags(request.tags)
                classification = await classify_and_guard(
                    self.agent_client,
                    request.current_message,
                    channel_hint=stream_hint,
                )
            
            if classification.is_violation:
                logger.warning(f"Security Violation detected: {classification.violation_reason}")
                
                # Log violation for at-risk student scoring
                try:
                    from .database.analytics_repo import log_security_event
                    await log_security_event(
                        request.thread_id, 
                        request.current_message, 
                        classification.violation_reason
                    )
                except Exception as log_err:
                    logger.error(f"Failed to log security violation: {log_err}")

                await context.abort(
                    grpc.StatusCode.FAILED_PRECONDITION, 
                    f"violation-reason:{classification.violation_reason}"
                )
                return

            # 2. Convert gRPC history to list of dicts for the agent
            history_list = []
            for msg in request.history:
                history_list.append({
                    "author_role": msg.author_role,
                    "author_name": msg.author_name,
                    "content": msg.content
                })
                
            # 2.5 Fetch student risk level
            risk_level = "NORMAL"
            try:
                from .database.analytics_repo import get_student_risk_level
                fetched_risk = await get_student_risk_level(request.thread_id)
                if fetched_risk:
                    risk_level = fetched_risk
            except Exception as r_err:
                logger.error(f"Failed to fetch risk level: {r_err}")
            
            # 3. Pass history + intent to agent loop
            collected_citations = []
            collected_metrics = {"cache_hit": False, "usage": {"input_tokens": None, "output_tokens": None}}
            async for chunk_data in run_agent_loop_stream(
                self.agent_client, 
                request.current_message, 
                history=history_list, 
                intent=classification.intent.value if hasattr(classification.intent, "value") else str(classification.intent),
                intent_confidence=classification.confidence,
                fallback_reason=classification.reasoning,
                risk_level=risk_level,
                thread_id=request.thread_id
            ):
                if context.cancelled():
                    break
                
                if chunk_data["type"] == "TOKEN":
                    yield ai_service_pb2.AIResponse(chunk=chunk_data["content"], is_finished=False)
                elif chunk_data["type"] == "ESCALATION":
                    # Stream plain text instead of JSON to avoid raw payload rendering on UI.
                    yield ai_service_pb2.AIResponse(chunk=chunk_data["content"], is_finished=False)
                elif chunk_data["type"] == "STATUS":
                    # Map technical status to friendly Vietnamese labels
                    status_map = {
                        "academic-prefetch": "Đang tra cứu tài liệu học tập",
                        "procedural-prefetch": "Đang kiểm tra quy chế đào tạo",
                        "intent_low_confidence": "Đang phân tích kỹ câu hỏi",
                        "classifier_unavailable": "Hệ thống đang bận, chuyển sang chế độ dự phòng",
                        "query_course_materials": "Tìm kiếm tài liệu môn học",
                        "query_regulations": "Tra cứu quy định & quy chế"
                    }
                    
                    raw_content = chunk_data.get("content", "")
                    friendly_label = status_map.get(raw_content, raw_content)
                    
                    # Check for partial matches (e.g. "Searching: query_regulations")
                    for key, val in status_map.items():
                        if key in raw_content:
                            friendly_label = val
                            break

                    status_msg = f"🔍 {friendly_label}...\n"
                    yield ai_service_pb2.AIResponse(chunk=status_msg, is_finished=False)
                elif chunk_data["type"] == "CITATIONS":
                    content = chunk_data.get("content", [])
                    if isinstance(content, list):
                        # Safety guardrail: truncate snippet to 2000 chars before sending to Java gateway
                        for c in content:
                            if "snippet" in c and c["snippet"]:
                                c["snippet"] = c["snippet"][:2000]
                        collected_citations = content
                    # Signal that citations are ready
                    yield ai_service_pb2.AIResponse(chunk="", is_finished=False)
                elif chunk_data["type"] == "METRICS":
                    content = chunk_data.get("content", {})
                    if isinstance(content, dict):
                        cache_hit = content.get("cache_hit")
                        usage = content.get("usage", {})
                        if isinstance(cache_hit, bool):
                            collected_metrics["cache_hit"] = cache_hit
                        if isinstance(usage, dict):
                            in_tok = usage.get("input_tokens")
                            out_tok = usage.get("output_tokens")
                            if isinstance(in_tok, int):
                                collected_metrics["usage"]["input_tokens"] = in_tok
                            if isinstance(out_tok, int):
                                collected_metrics["usage"]["output_tokens"] = out_tok
                elif chunk_data["type"] == "DONE":
                    # 4. Set agent metadata based on intent
                    agent_used = "QA_AGENT"
                    if classification.intent == IntentType.PROCEDURAL:
                        agent_used = "ASSIGNMENT_AGENT"
                    # Non-breaking metadata extension: append routing context into agent_used.
                    if classification.intent == IntentType.UNCERTAIN:
                        agent_used = f"{agent_used}|intent=UNCERTAIN|fallback={classification.reasoning or 'none'}"
                    elif classification.confidence < 0.45:
                        agent_used = f"{agent_used}|intent={classification.intent.value if hasattr(classification.intent, 'value') else classification.intent}|fallback=low_confidence"
                    # Non-breaking metadata extension via tag suffix (Java parses these into structured metadata fields).
                    cache_hit = "true" if collected_metrics.get("cache_hit") else "false"
                    in_tok = collected_metrics.get("usage", {}).get("input_tokens")
                    out_tok = collected_metrics.get("usage", {}).get("output_tokens")
                    in_tok_s = str(in_tok) if isinstance(in_tok, int) else ""
                    out_tok_s = str(out_tok) if isinstance(out_tok, int) else ""
                    agent_used = f"{agent_used}|cache_hit={cache_hit}|input_tokens={in_tok_s}|output_tokens={out_tok_s}"
                    
                    proto_citations = []
                    for c in collected_citations:
                        try:
                            source_file = str(c.get("source_file", "")).strip()
                            if not source_file:
                                continue
                            page_number = int(c.get("page_number", 0) or 0)
                            proto_citations.append(
                                ai_service_pb2.Citation(
                                    source_file=source_file,
                                    page_number=page_number,
                                    document_id=str(c.get("document_id", "") or ""),
                                    source_uri=str(c.get("source_uri", "") or ""),
                                    chunk_id=str(c.get("chunk_id", "") or ""),
                                    chunk_index=int(c.get("chunk_index", 0) or 0),
                                    snippet=str(c.get("snippet", "") or ""),
                                )
                            )
                        except Exception:
                            continue
                    metadata = ai_service_pb2.ResponseMetadata(
                        agent_used=agent_used,
                        citations=proto_citations
                    )
                    yield ai_service_pb2.AIResponse(chunk="", is_finished=True, metadata=metadata)
                elif chunk_data["type"] == "ERROR":
                    yield ai_service_pb2.AIResponse(chunk=f"Error: {chunk_data['content']}", is_finished=True)

        except Exception as e:
            if isinstance(e, grpc.RpcError):
                raise e # Re-raise gRPC errors (like the abort above)
            logger.error(f"gRPC StreamAIResponse error: {e}")
            yield ai_service_pb2.AIResponse(chunk=f"Internal error: {str(e)}", is_finished=True)

    async def ClassifyIntent(self, request, context: grpc.aio.ServicerContext):
        """Implementation of ClassifyIntent RPC using the public channel contract."""
        try:
            hint = normalize_classifier_channel_hint(request.channel_hint or "PUBLIC")
            security_result = await classify_and_guard(
                self.agent_client,
                request.content,
                channel_hint=hint,
            )
            classification = classify_channel_intent(request.content, request.channel_hint, security_result)
            intent_map = {
                ChannelIntent.PUBLIC: ai_service_pb2.PUBLIC,
                ChannelIntent.PRIVATE: ai_service_pb2.PRIVATE,
                ChannelIntent.UNCERTAIN: ai_service_pb2.UNCERTAIN
            }
            proto_intent = intent_map.get(classification.suggested_channel, ai_service_pb2.UNCERTAIN)
            
            # Suffix lets the Java gateway extract ACADEMIC/PROCEDURAL/UNCERTAIN for the preflight tag (proto unchanged).
            reasoning_out = (
                classification.reasoning
                + f"||TASK_INTENT:{security_result.intent.value}||"
            )
            return ai_service_pb2.ClassifyResponse(
                suggested_channel=proto_intent,
                confidence=classification.confidence,
                reasoning=reasoning_out,
                is_violation=classification.is_violation
            )
        except Exception as e:
            logger.error(f"gRPC ClassifyIntent error: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return ai_service_pb2.ClassifyResponse()

    async def SuggestSimilarThreads(self, request, context: grpc.aio.ServicerContext):
        response = ai_service_pb2.SuggestResponse()
        return response

class AIDocumentServicer(ai_service_pb2_grpc.AIDocumentServiceServicer if ai_service_pb2_grpc else object):
    def __init__(self, agent_client):
        self.agent_client = agent_client

    async def ProcessDocument(self, request, context: grpc.aio.ServicerContext):
        asyncio.create_task(run_ingestion_with_callback(request.document_id, request.file_url))
        return ai_service_pb2.DocumentResponse(accepted=True)

    async def UpdateChunkContent(self, request, context: grpc.aio.ServicerContext):
        """
        Allows TA to edit a specific chunk. 
        Regenerates embedding and clears relevant semantic cache.
        """
        try:
            from .database.vector_repo import update_chunk_content
            from openai import AsyncOpenAI
            from .config import OPENAI_API_KEY
            
            # 1. Generate new embedding for corrected text
            client = AsyncOpenAI(api_key=OPENAI_API_KEY)
            emb_resp = await client.embeddings.create(
                input=[request.new_content],
                model="text-embedding-3-small"
            )
            new_vector = emb_resp.data[0].embedding
            
            # 2. Update Vector DB
            success = await update_chunk_content(
                chunk_id=request.chunk_id,
                new_content=request.new_content,
                new_embedding=new_vector,
                updated_by=request.updated_by or "TA_CORRECTION"
            )
            
            if success:
                logger.info(f"Chunk correction successful for ID: {request.chunk_id}")
                return ai_service_pb2.UpdateChunkResponse(success=True, message="Updated successfully")
            else:
                return ai_service_pb2.UpdateChunkResponse(success=False, message="Chunk not found or DB error")
                
        except Exception as e:
            logger.error(f"gRPC UpdateChunkContent error: {e}")
            return ai_service_pb2.UpdateChunkResponse(success=False, message=str(e))

async def serve_grpc_async(agent_client):
    if not ai_service_pb2_grpc:
        return None
    
    # Increase the number of workers to handle 50+ concurrent requests
    # Since we are using async gRPC, the thread pool is used for non-async parts
    from concurrent import futures
    server = aio.server(
        options=[
            ('grpc.max_send_message_length', 50 * 1024 * 1024),
            ('grpc.max_receive_message_length', 50 * 1024 * 1024),
            ('grpc.so_reuseport', 1),
        ]
    )
    ai_service_pb2_grpc.add_AIThreadServiceServicer_to_server(AIThreadServicer(agent_client), server)
    ai_service_pb2_grpc.add_AIDocumentServiceServicer_to_server(AIDocumentServicer(agent_client), server)
    listen_addr = f'0.0.0.0:{GRPC_PORT}'
    server.add_insecure_port(listen_addr)
    await server.start()
    return server
