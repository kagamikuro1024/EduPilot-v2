import pytest
import openai
from unittest.mock import AsyncMock, patch, MagicMock
from src.agent import run_agent_loop_stream

@pytest.mark.asyncio
async def test_agent_yields_429_on_rate_limit_error():
    # Arrange
    client_mock = AsyncMock()
    # Setting up the completions.create to raise RateLimitError
    response_mock = MagicMock()
    response_mock.request = MagicMock()
    error = openai.RateLimitError(message="Rate limit exceeded", response=response_mock, body=None)
    client_mock.chat.completions.create.side_effect = error
    
    # Act
    chunks = []
    # Using patch to prevent actual database connections during the fallback/escalation logic if it hits
    with patch('src.agent.open'):
        generator = run_agent_loop_stream(client_mock, user_input="Hello", intent_confidence=1.0)
        async for chunk in generator:
            chunks.append(chunk)
            
    # Assert
    # The generator should catch the error and yield exactly one chunk indicating system_error and 429
    assert len(chunks) == 1
    error_chunk = chunks[0]
    
    assert error_chunk.get("type") == "system_error"
    assert error_chunk.get("code") == 429
    assert "Hệ thống đang quá tải" in error_chunk.get("message")
