"""
Module: data_pipeline.pipeline.embedding
Description: Asynchronous tool for generating Vector Embeddings using OpenAI API.
             Supports batch processing without blocking the FastAPI Event Loop.
"""

import logging
from openai import AsyncOpenAI
from src.config import OPENAI_API_KEY

logger = logging.getLogger(__name__)

# Initialize asynchronous OpenAI client with optimized connection handling
client = AsyncOpenAI(api_key=OPENAI_API_KEY)

async def generate_embeddings(texts: list[str]) -> list[list[float]]:
    """
    Generates vector embeddings for a list of texts.
    
    Args:
        texts (list[str]): List of text segments to embed.
        
    Returns:
        list[list[float]]: List of corresponding vectors (size 1536).
    """
    if not texts:
        return []

    try:
        # Use text-embedding-3-small for cost efficiency and high performance
        logger.info(f"Generating embeddings for {len(texts)} segments...")
        
        response = await client.embeddings.create(
            input=texts,
            model="text-embedding-3-small"
        )

        # Extract vector data from the API response
        embeddings = [data.embedding for data in response.data]
        
        logger.info(f"Successfully generated {len(embeddings)} embeddings.")
        return embeddings
        
    except Exception as e:
        logger.error(f"Error calling OpenAI Embedding API: {e}")
        raise
