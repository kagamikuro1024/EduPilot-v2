"""
Async agent loop using the OpenAI API.
Receives user input, calls tools as needed, and streams results.
"""

import json
import logging
import re
import openai
from datetime import datetime
from openai import AsyncOpenAI
from .config import OPENAI_API_KEY, DEFAULT_MODEL, LOG_LEVEL, DATABASE_URL
from .tools import (
    get_tool_schemas,
    execute_tool,
    consume_last_rag_citations,
    consume_last_rag_runtime,
)

logging.basicConfig(level=LOG_LEVEL, format="%(asctime)s [%(levelname)s] %(message)s")
logger = logging.getLogger(__name__)

_REGULATION_KEYWORDS = (
    "quy chế", "quy che", "quy định", "quy dinh", "buộc thôi học", "buoc thoi hoc",
    "thôi học", "thoi hoc", "cảnh cáo học vụ", "canh cao hoc vu", "học vụ", "hoc vu",
    "điểm lý thuyết", "diem ly thuyet", "điểm thành phần", "diem thanh phan",
    "điều kiện dự thi", "dieu kien du thi", "điều kiện tốt nghiệp", "dieu kien tot nghiep",
    "bảo lưu", "bao luu", "nghỉ học", "nghi hoc", "thi cử", "thi cu",
)
_PROCEDURAL_SQL_KEYWORDS = (
    "deadline", "hạn nộp", "han nop", "nộp bài", "nop bai", "assignment", "bài tập", "bai tap",
    "lms", "quá hạn", "qua han", "ngày nộp", "ngay nop",
)


def _should_prefetch_regulations(user_input: str) -> bool:
    lowered = (user_input or "").strip().lower()
    if not lowered:
        return False
    has_regulation_signal = any(k in lowered for k in _REGULATION_KEYWORDS)
    has_sql_deadline_signal = any(k in lowered for k in _PROCEDURAL_SQL_KEYWORDS)
    # Hybrid queries may ask both policy and deadlines; still prefetch regulations.
    return has_regulation_signal or (("thi" in lowered or "điểm" in lowered or "diem" in lowered) and not has_sql_deadline_signal)


# Ban models from echoing internal context labels (e.g. *_CONTEXT) into student-visible text.
_STUDENT_REPLY_NO_INTERNAL_LABELS = (
    "Trong câu trả lời gửi sinh viên: không in nhãn nội bộ kiểu tên biến (chữ HOA, gạch dưới) "
    "hay chuỗi trong ngoặc vuông giống placeholder; diễn đạt hoàn toàn bằng tiếng Việt tự nhiên.\n\n"
)


# Mandatory System Prompt based on PROBLEM_STATEMENT.md 2.4
SYSTEM_PROMPT = """You are the AI Teaching Assistant for the 'Introduction to IT' course.
Your goal is to help students learn using the Socratic method.

MANDATORY TOOL USAGE:
- You MUST call 'query_course_materials' tool BEFORE composing ANY response to the student, EXCEPT when the question is about assignment deadlines, submission status, or logistics (where you MUST use database tools 'check_assignment_deadline' or 'get_assignments' instead).
- Every claim, explanation, or concept you mention MUST be grounded in retrieved course materials.
- If query_course_materials returns no relevant results, explicitly tell the student that the topic is not covered in the current course materials and suggest they consult the TA. (This does not apply to assignment/deadline queries where you should use database tools 'check_assignment_deadline' or 'get_assignments', or to simple greetings/conversational pleasantries where you can respond naturally).
- This rule applies to ALL academic questions. For simple greetings (e.g., 'Hello', 'Hi', 'Xin chào'), meta-questions about your capabilities (e.g., 'bạn có thể làm được gì?', 'bạn là ai?'), or expressions of thanks ('cảm ơn'), respond naturally WITHOUT calling any tools. Ask how you can help with their course work.
- For ACADEMIC / conceptual questions (bài học, kiến thức môn): you MUST call 'query_course_materials' before answering.
- For school REGULATIONS / policy (quy chế, quy định, điểm lý thuyết, thi cử, điều kiện tốt nghiệp, học vụ): you MUST call 'query_regulations' before answering.
- NEVER answer from your own knowledge or training data alone for factual rules or course content.
- If the relevant tool returns no useful results, say so clearly and suggest TA / phòng đào tạo as appropriate.

CONSTRAINTS:
1. NEVER provide direct answers to homework or solve problems for the student.
2. Provide hints, ask guiding questions, or explain underlying concepts.
3. You MUST use IN-TEXT CITATIONS: cite your source INLINE at the end of each claim or paragraph, using the format [FileName, p.X]. Do NOT list all sources at the bottom. Example: 'RabbitMQ sử dụng mô hình Exchange để định tuyến tin nhắn [RabbitMQ_Architecture.pdf, p.3].' (No citations needed for database deadline lookups).
4. If asked about a SPECIFIC assignment deadline, use the 'check_assignment_deadline' tool. If asked about a LIST of deadlines or general upcoming assignments, use the 'get_assignments' tool. DO NOT invent assignment names.
5. If the information is not in the course materials or database, state that you don't know and suggest they wait for a TA.
6. Use a professional, encouraging, and academic tone.
7. Keep answers concise and evidence-first. Avoid generic filler.

Pedagogical approach: Instead of 'The answer is X', say 'Have you considered how concept Y applies to this situation?' or 'You might find the rule for this in the syllabus under section Z'.

GUARDRAILS:
1. You MUST start every response with your confidence score in the format [CONFIDENCE: <number>] where <number> is 0-100.
2. If your confidence score is below 80, still output the score first, then perform your best effort response. The system will handle the escalation.

CRITICAL: Always communicate with the student in Vietnamese.
"""


def create_agent():
    """Create an agent with the Async OpenAI client."""
    if not OPENAI_API_KEY:
        raise ValueError("OPENAI_API_KEY is not configured. Check your .env file")
    return AsyncOpenAI(api_key=OPENAI_API_KEY)


async def run_agent_loop_stream(
    client: AsyncOpenAI,
    user_input: str,
    history: list = None,
    max_turns: int = 10,
    intent: str = "UNCERTAIN",
    intent_confidence: float = 0.0,
    fallback_reason: str = "",
    risk_level: str = "NORMAL",
    thread_id: str = None,
):
    """
    Run the agent loop with Server Streaming Events.
    Yields dictionary chunks with keys 'type' (TOKEN, STATUS, DONE, ERROR) and 'content'.
    """
    # Lấy thời gian thực tế mỗi khi có request mới
    current_time = datetime.now().strftime('%Y-%m-%d %H:%M:%S')

    system_prompt = SYSTEM_PROMPT + f"\n\nCURRENT SERVER TIME: {current_time}. Use this to resolve relative time references like 'today', 'tomorrow', 'nay', or 'mai'."

    # Inject routing instructions based on Intent Classification (Kaizen: dynamic specialization)
    if intent == "CONVERSATIONAL":
        system_prompt += (
            "\n\nROUTING: This is a CONVERSATIONAL message (greeting, thanks, or meta-question about your capabilities). "
            "Respond naturally and warmly in Vietnamese. Do NOT call any tools. "
            "If the student asks what you can do, list your capabilities clearly:\n"
            "1. Giải thích khái niệm và thuật ngữ trong khóa học\n"
            "2. Hướng dẫn học tập theo phương pháp Socratic\n"
            "3. Tìm kiếm và trích dẫn tài liệu môn học\n"
            "4. Tra cứu deadline bài tập và lịch thi\n"
            "5. Giải đáp thắc mắc về quy chế, quy định học vụ\n"
            "Keep the response friendly, concise, and invite them to ask a specific question."
        )
    elif intent == "PROCEDURAL":
        system_prompt += (
            "\n\nROUTING: This question is about PROCEDURAL matters. "
            "Use 'check_assignment_deadline' for specific assignments, or 'get_assignments' to list assignments/deadlines. "
            "For quy chế đào tạo, điểm số lý thuyết, thi cử, kỷ luật, học vụ, or other official school rules, "
            "you MUST call 'query_regulations' (in addition to SQL tools if the user also asks about concrete deadlines)."
        )
        system_prompt += "\nRULE: Do not invent deadlines/logistics. If DB tool cannot verify, explicitly say unsure and advise LMS."
    elif intent == "ACADEMIC":
        system_prompt += "\n\nROUTING: This question is about ACADEMIC concepts. Prioritize using 'query_course_materials' tool to provide detailed explanations."
    elif intent == "UNCERTAIN":
        system_prompt += "\n\nROUTING: Intent is UNCERTAIN. Prefer cautious response. If not sure, ask user to clarify and escalate to TA."
        
    if risk_level in ["CRITICAL", "WARNING"]:
        system_prompt += f"\n\nRISK ALERT: This student has been flagged as at-risk (Level: {risk_level})."
        system_prompt += "\nWhen answering procedural/logistics queries, format your answer using a NUMBERED STEP list."
        system_prompt += "\nYou MUST include a strict warning for them to check the LMS carefully and strongly advise them to contact the TA if they need help."

    if intent_confidence < 0.25:
        yield {
            "type": "STATUS",
            "content": "fallback:intent_low_confidence"
        }
        # Keep stream clean for benchmark quality: avoid generic disclaimer filler.

    messages = [{"role": "system", "content": system_prompt}]
    
    # Add history if provided
    if history:
        for msg in history:
            role = msg.get("author_role", "").lower()
            if role == "student":
                role = "user"
            elif role == "ai":
                role = "assistant"
            elif role == "ta":
                role = "user" # Treat TA as an authoritative user
            
            if role in ["user", "assistant", "system"]:
                messages.append({"role": role, "content": msg.get("content", "")})
    
    messages.append({"role": "user", "content": user_input})
    
    tools = get_tool_schemas()
    used_deadline_tool = False
    used_assignments_tool = False
    rag_was_called = False
    rag_cache_hit = None
    usage_prompt_tokens = None
    usage_completion_tokens = None
    is_escalated = False

    if intent == "ACADEMIC":
        rag_query = user_input
        logger.info("Academic RAG prefetch query=%s", rag_query)
        yield {"type": "STATUS", "content": "Searching: query_course_materials (academic-prefetch)"}
        rag_result = await execute_tool("query_course_materials", {"query": rag_query})
        rag_was_called = True
        runtime = consume_last_rag_runtime()
        if isinstance(runtime, dict):
            rag_cache_hit = bool(runtime.get("cache_hit", False))
        citations = consume_last_rag_citations()
        if citations:
            yield {"type": "CITATIONS", "content": citations}
        messages.append(
            {
                "role": "system",
                "content": (
                    "Use ONLY the grounded course-material excerpts below to answer. "
                    "If the context is insufficient, say so and suggest contacting TA.\n"
                    + _STUDENT_REPLY_NO_INTERNAL_LABELS
                    + "When the excerpts below are a numbered list with inline citations, preserve EVERY item and its "
                    "[File, p.X] citation; do not merge, drop, or shorten the list.\n\n"
                    f"Tài liệu môn học đã tra cứu:\n{rag_result}"
                ),
            }
        )
        # Keep academic flow deterministic by preventing model-side query_course_materials tool calls,
        # but KEEP database tools check_assignment_deadline and get_assignments available as fallback.
        tools = [t for t in tools if t["function"]["name"] in ["check_assignment_deadline", "get_assignments"]]
        # Keep academic flow deterministic by preventing model-side tool argument rewrites.
        tools = []
    elif intent == "PROCEDURAL" and _should_prefetch_regulations(user_input):
        regulation_query = user_input
        logger.info("Procedural regulation prefetch query=%s", regulation_query)
        yield {"type": "STATUS", "content": "Searching: query_regulations (procedural-prefetch)"}
        regulation_result = await execute_tool("query_regulations", {"query": regulation_query})
        rag_was_called = True
        runtime = consume_last_rag_runtime()
        if isinstance(runtime, dict):
            rag_cache_hit = bool(runtime.get("cache_hit", False))
        citations = consume_last_rag_citations()
        if citations:
            yield {"type": "CITATIONS", "content": citations}
        messages.append(
            {
                "role": "system",
                "content": (
                    "Use ONLY the grounded regulation excerpts below to answer this procedural/policy question. "
                    "Do NOT introduce facts outside this context. Preserve citations exactly in [File, p.X] format. "
                    "If context is insufficient, say so and advise the student to verify with LMS/Phòng đào tạo.\n"
                    + _STUDENT_REPLY_NO_INTERNAL_LABELS
                    + "LUẬT BỔ SUNG (quy chế / final answer):\n"
                    "- Đồng nghĩa: các từ ngữ thông dụng của sinh viên (ví dụ: đuổi học, cấm thi) có giá trị "
                    "tương đương với thuật ngữ pháp lý trong các đoạn quy chế bên dưới (buộc thôi học, kỉ luật...).\n"
                    "- TUYỆT ĐỐI KHÔNG DỪNG LẠI GIỮA CHỪNG: nếu các đoạn quy chế bên dưới liệt kê nhiều điều kiện hoặc "
                    "gạch đầu dòng, phải trình bày đầy đủ tất cả; cấm lược bỏ.\n\n"
                    f"Đoạn trích quy chế (chỉ dùng làm căn cứ; trình bày lại bằng tiếng Việt cho sinh viên):\n"
                    f"{regulation_result}"
                ),
            }
        )
        # Deterministic regulation flow: skip extra tool planning for this turn.
        tools = []

    for turn in range(max_turns):
        logger.info(f"Turn {turn + 1}/{max_turns}")

        full_content = ""
        tool_calls_accum = {}
        finish_reason = None
        confidence_score = 100
        tag_buffer = ""
        is_tag_parsed = False
        is_escalated = False

        try:
            stream = await client.chat.completions.create(
                model=DEFAULT_MODEL,
                messages=messages,
                tools=tools if tools else None,
                max_completion_tokens=2048,
                temperature=0,
                stream_options={"include_usage": True},
                stream=True,
            )
            
            async for chunk in stream:
                if getattr(chunk, "usage", None):
                    usage = chunk.usage
                    usage_prompt_tokens = getattr(usage, "prompt_tokens", usage_prompt_tokens)
                    usage_completion_tokens = getattr(usage, "completion_tokens", usage_completion_tokens)
                delta = chunk.choices[0].delta if chunk.choices else None
                if delta is None:
                    continue

                if delta.content:
                    if not is_tag_parsed:
                        tag_buffer += delta.content
                        if "]" in tag_buffer:
                            # Try to parse [CONFIDENCE: XX]
                            match = re.search(r"\[CONFIDENCE:\s*(\d+)\]", tag_buffer)
                            if match:
                                confidence_score = int(match.group(1))
                                is_tag_parsed = True
                                # Remove the tag from what we will yield
                                remaining_content = tag_buffer[match.end():].lstrip()
                                
                                # Guardrail: Lower threshold for at-risk students so they get specialized instructions
                                # instead of immediate escalation for slightly ambiguous queries.
                                # Guardrail: Lower threshold for at-risk students so they get specialized instructions
                                # instead of immediate escalation for slightly ambiguous queries.
                                effective_threshold = 80 if risk_level == "NORMAL" else 40
                                
                                if confidence_score < effective_threshold:
                                    logger.warning(
                                        f"Confidence score {confidence_score} < {effective_threshold} (Risk: {risk_level}). "
                                        "Emit escalation signal but continue answer generation."
                                    )
                                    yield {
                                        "type": "ESCALATION",
                                        "content": "Câu hỏi có độ chắc chắn thấp, vui lòng kiểm tra kỹ nguồn trích dẫn."
                                    }
                                elif remaining_content:
                                    yield {"type": "TOKEN", "content": remaining_content}
                            elif len(tag_buffer) > 80: # Increased tolerance for LLM preamble before tag
                                is_tag_parsed = True
                                yield {"type": "TOKEN", "content": tag_buffer}
                        continue

                    full_content += delta.content
                    yield {"type": "TOKEN", "content": delta.content}

                if delta.tool_calls:
                    for tc_delta in delta.tool_calls:
                        idx = tc_delta.index
                        if idx not in tool_calls_accum:
                            tool_calls_accum[idx] = {"id": "", "function": {"name": "", "arguments": ""}}
                        if tc_delta.id:
                            tool_calls_accum[idx]["id"] = tc_delta.id
                        if tc_delta.function:
                            if tc_delta.function.name:
                                tool_calls_accum[idx]["function"]["name"] += tc_delta.function.name
                            if tc_delta.function.arguments:
                                tool_calls_accum[idx]["function"]["arguments"] += tc_delta.function.arguments

                    finish_reason = chunk.choices[0].finish_reason

            # Flush remaining buffer if tag was never fully parsed (TIP-009)
            if not is_tag_parsed and tag_buffer:
                is_tag_parsed = True
                yield {"type": "TOKEN", "content": tag_buffer}
                full_content += tag_buffer
                tag_buffer = ""

        except (openai.RateLimitError, openai.APITimeoutError, TimeoutError) as e:
            logger.error(f"LLM Provider RateLimit/Timeout: {e}")
            # TIP-005: Yield system_error and stop processing, DO NOT RETRY
            yield {
                "type": "system_error",
                "code": 429,
                "message": "Hệ thống đang quá tải do lượng truy cập lớn. Vui lòng tải lại trang (F5) và thử lại sau ít phút."
            }
            # Log to session.jsonl directly (using standard btc format)
            try:
                log_entry = {
                    "event": "429_rate_limit",
                    "error": str(e)
                }
                with open(".ai-log/session.jsonl", "a", encoding="utf-8") as f:
                    f.write(json.dumps(log_entry, ensure_ascii=False) + "\n")
            except Exception as log_ex:
                logger.error(f"Failed to log 429 event to session.jsonl: {log_ex}")
            return


        assistant_msg = {"role": "assistant", "content": full_content or None}
        if tool_calls_accum:
            assistant_msg["tool_calls"] = [
                {
                    "id": tc["id"],
                    "type": "function",
                    "function": {"name": tc["function"]["name"], "arguments": tc["function"]["arguments"]},
                }
                for idx, tc in sorted(tool_calls_accum.items())
            ]
        messages.append(assistant_msg)

        if not tool_calls_accum:
            break

        # Always execute tool calls if present, even if finish_reason != "tool_calls" (TIP-010)
        # This prevents 400 errors from OpenAI when a tool_call is left unresponded.
        for tc in sorted(tool_calls_accum.values(), key=lambda x: x["id"]):
            func_name = tc["function"]["name"]
            try:
                func_args = json.loads(tc["function"]["arguments"])
            except json.JSONDecodeError:
                func_args = {}

            if func_name == "query_course_materials":
                logger.info("Model tool query for RAG=%s (user_input=%s)", func_args.get("query"), user_input)
            if func_name == "query_regulations":
                logger.info("Model tool query for regulations=%s (user_input=%s)", func_args.get("query"), user_input)
            yield {"type": "STATUS", "content": f"Searching: {func_name}"}
            result = await execute_tool(func_name, func_args)
            if func_name == "check_assignment_deadline":
                used_deadline_tool = True
            if func_name == "get_assignments":
                used_assignments_tool = True
            if func_name in ("query_course_materials", "query_regulations"):
                rag_was_called = True
                citations = consume_last_rag_citations()
                runtime = consume_last_rag_runtime()
                if isinstance(runtime, dict) and runtime.get("cache_hit") is True:
                    rag_cache_hit = True
                elif rag_cache_hit is None:
                    rag_cache_hit = False
                if citations:
                    yield {"type": "CITATIONS", "content": citations}
            
            messages.append({"role": "tool", "tool_call_id": tc["id"], "content": result})

    # ── GUARDRAIL: Force RAG if LLM skipped it ──────────────────────────────
    # Expanded greeting/identity/meta-question detection (TIP-008)
    greetings = [
        "hello", "hi", "xin chào", "chào bạn", "hey", "bạn là ai", "ai đấy", 
        "tên là gì", "who are you", "là trợ lý", "là robot", "là bot", "là ai",
        "trợ lý ảo", "trợ lý ai", "bot à", "robot à",
        # Meta-questions about capabilities
        "làm được gì", "có thể làm gì", "giúp gì", "hỗ trợ gì",
        "làm gì được", "biết làm gì", "có khả năng gì", "what can you do",
        "cảm ơn", "thank", "thanks",
    ]
    is_greeting = (intent in ("UNCERTAIN", "GREETING", "CONVERSATIONAL")) and any(greet in user_input.lower() for greet in greetings)
    # Also treat CONVERSATIONAL intent as greeting regardless of keyword match
    is_greeting = is_greeting or intent == "CONVERSATIONAL"
    
    procedural_sql_only = used_deadline_tool or used_assignments_tool
    fallback_citations: list = []

    if not is_escalated and not rag_was_called and not is_greeting:
        if intent == "PROCEDURAL" and procedural_sql_only:
            # Pure LMS / deadline questions — no forced regulation RAG
            pass
        else:
            # Determine which tool to fallback to
            if intent == "PROCEDURAL":
                logger.warning("Agent did not call query_regulations. Force-triggering regulation RAG fallback.")
                yield {"type": "STATUS", "content": "Searching: query_regulations (fallback)"}
                rag_result = await execute_tool("query_regulations", {"query": user_input})
            else:
                logger.warning("Agent did not call query_course_materials. Force-triggering RAG fallback.")
                yield {"type": "STATUS", "content": "Searching: query_course_materials (fallback)"}
                rag_result = await execute_tool("query_course_materials", {"query": user_input})
            
            fallback_citations = consume_last_rag_citations()
            if fallback_citations:
                yield {"type": "CITATIONS", "content": fallback_citations}

            # AFTER fallback tool execution, we MUST call LLM again to generate the final answer (BUG-FIX)
            messages.append({
                "role": "system",
                "content": f"FALLBACK CONTEXT: The student query '{user_input}' was not handled by any tool. Use this context to answer if relevant:\n{rag_result}"
            })
            
            try:
                final_stream = await client.chat.completions.create(
                    model=DEFAULT_MODEL,
                    messages=messages,
                    max_completion_tokens=1024,
                    temperature=0,
                    stream=True,
                )
                async for chunk in final_stream:
                    delta = chunk.choices[0].delta if chunk.choices else None
                    if delta and delta.content:
                        yield {"type": "TOKEN", "content": delta.content}
            except Exception as e:
                logger.error(f"Error in final fallback LLM call: {e}")
                yield {"type": "TOKEN", "content": "Xin lỗi, tôi gặp chút trục trặc khi tổng hợp câu trả lời. Bạn vui lòng thử lại nhé."}

    yield {
        "type": "METRICS",
        "content": {
            "cache_hit": bool(rag_cache_hit) if rag_cache_hit is not None else False,
            "usage": {
                "input_tokens": int(usage_prompt_tokens) if usage_prompt_tokens is not None else None,
                "output_tokens": int(usage_completion_tokens) if usage_completion_tokens is not None else None,
            },
        },
    }
    yield {"type": "DONE", "content": ""}

async def run_agent_loop(client: AsyncOpenAI, user_input: str, history: list = None, max_turns: int = 10) -> str:
    full_response = ""
    async for chunk in run_agent_loop_stream(client, user_input, history, max_turns):
        if chunk["type"] == "TOKEN":
            full_response += chunk["content"]
    return full_response
