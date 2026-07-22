"""Thin async client for Ollama's generate API."""

import httpx

from app.config import settings


class OllamaError(Exception):
    pass


async def generate(prompt: str, system: str | None = None) -> str:
    """Calls Ollama's /api/generate with streaming disabled. Uses keep_alive
    to avoid paying the ~35s model-load penalty on every request, and caps
    num_predict so a single query has a bounded worst-case latency rather
    than running unbounded on constrained hardware.
    """
    payload = {
        "model": settings.ollama_model,
        "prompt": prompt,
        "stream": False,
        "keep_alive": settings.ollama_keep_alive,
        "options": {
            "num_predict": settings.ollama_num_predict,
        },
    }
    if system:
        payload["system"] = system

    try:
        async with httpx.AsyncClient(timeout=settings.ollama_timeout_seconds) as client:
            response = await client.post(
                f"{settings.ollama_base_url}/api/generate",
                json=payload,
            )
            response.raise_for_status()
            data = response.json()
            return data.get("response", "").strip()
    except httpx.HTTPError as e:
        raise OllamaError(f"Ollama request failed: {e}") from e
