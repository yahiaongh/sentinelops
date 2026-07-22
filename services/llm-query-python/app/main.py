"""SentinelOps LLM Query Service.

Exposes a natural-language query interface over detected anomalies and
service stats, grounded via retrieval against TimescaleDB (RAG), with
generation via a locally-hosted Ollama model.
"""

import asyncio
import logging
from contextlib import asynccontextmanager

import asyncpg
from fastapi import FastAPI, HTTPException
from fastapi.responses import PlainTextResponse
from prometheus_client import Counter, Histogram, generate_latest, CONTENT_TYPE_LATEST
from pydantic import BaseModel

from app.config import settings
from app.jobs import Job, JobStatus, job_store
from app.ollama_client import OllamaError, generate
from app.retrieval import format_context_for_prompt, retrieve_context

logging.basicConfig(
    level=logging.INFO, format="%(asctime)s [%(levelname)s] %(name)s: %(message)s"
)
logger = logging.getLogger("llm-query-service")

QUERIES_TOTAL = Counter(
    "sentinelops_llm_queries_total", "Total number of queries submitted."
)
QUERY_FAILURES_TOTAL = Counter(
    "sentinelops_llm_query_failures_total", "Total number of failed queries."
)
QUERY_DURATION = Histogram(
    "sentinelops_llm_query_duration_seconds",
    "Time from job creation to completion, including LLM generation.",
    buckets=[1, 5, 15, 30, 60, 120, 180, 300],
)

SYSTEM_PROMPT = (
    "You are an SRE assistant analyzing distributed system telemetry. "
    "Answer questions using ONLY the context data provided below. "
    "If the context doesn't contain enough information to answer, say so "
    "explicitly rather than guessing. Be concise and specific — cite actual "
    "numbers from the context. Do not invent services, timestamps, or metrics "
    "that are not present in the context."
)


@asynccontextmanager
async def lifespan(app: FastAPI):
    app.state.pool = await asyncpg.create_pool(
        settings.database_url, min_size=1, max_size=5
    )
    logger.info("connected to TimescaleDB")
    yield
    await app.state.pool.close()


app = FastAPI(title="SentinelOps LLM Query Service", lifespan=lifespan)


class QueryRequest(BaseModel):
    query: str
    service: str | None = None
    lookback_hours: int | None = None


class QueryAccepted(BaseModel):
    job_id: str
    status: JobStatus


async def _run_query_job(
    job_id: str, request: QueryRequest, pool: asyncpg.Pool
) -> None:
    await job_store.mark_running(job_id)
    try:
        context = await retrieve_context(
            pool,
            lookback_hours=request.lookback_hours or settings.default_lookback_hours,
            service_filter=request.service,
        )
        context_text = format_context_for_prompt(context)

        prompt = f"{context_text}\n\nQuestion: {request.query}\n\nAnswer:"
        answer = await generate(prompt, system=SYSTEM_PROMPT)

        await job_store.mark_complete(job_id, answer)
    except OllamaError as e:
        logger.error("ollama generation failed", exc_info=e)
        QUERY_FAILURES_TOTAL.inc()
        await job_store.mark_failed(job_id, str(e))
    except Exception as e:  # noqa: BLE001 - job failures must never crash the worker
        logger.error("unexpected error processing query job", exc_info=e)
        QUERY_FAILURES_TOTAL.inc()
        await job_store.mark_failed(job_id, "internal error processing query")


@app.post("/query", response_model=QueryAccepted, status_code=202)
async def submit_query(request: QueryRequest) -> QueryAccepted:
    QUERIES_TOTAL.inc()
    job = await job_store.create(request.query)

    async def _tracked_run():
        with QUERY_DURATION.time():
            await _run_query_job(job.job_id, request, app.state.pool)

    asyncio.create_task(_tracked_run())
    return QueryAccepted(job_id=job.job_id, status=job.status)


@app.get("/query/{job_id}", response_model=Job)
async def get_query_result(job_id: str) -> Job:
    job = await job_store.get(job_id)
    if job is None:
        raise HTTPException(status_code=404, detail="job not found")
    return job


@app.get("/healthz")
async def healthz() -> dict[str, str]:
    return {"status": "ok"}


@app.get("/metrics")
async def metrics() -> PlainTextResponse:
    return PlainTextResponse(generate_latest(), media_type=CONTENT_TYPE_LATEST)
