"""In-memory async job queue for LLM query requests.

Ollama inference on constrained hardware can take tens of seconds to
minutes, which makes a blocking HTTP request/response the wrong shape for
this API. Instead, POST /query enqueues a job and returns immediately;
GET /query/{job_id} polls for status and result.

This is intentionally in-memory (a dict), not Redis-backed, for now: it's
simple, correct for a single-process deployment, and the documented
upgrade path (Redis is already provisioned in the stack) is to swap the
JobStore implementation without changing the API contract, once multi-
worker or multi-process deployment is actually needed.
"""

import asyncio
import uuid
from datetime import datetime, timezone
from enum import Enum

from pydantic import BaseModel


class JobStatus(str, Enum):
    PENDING = "pending"
    RUNNING = "running"
    COMPLETE = "complete"
    FAILED = "failed"


class Job(BaseModel):
    job_id: str
    status: JobStatus
    query: str
    created_at: datetime
    completed_at: datetime | None = None
    result: str | None = None
    error: str | None = None


class JobStore:
    def __init__(self) -> None:
        self._jobs: dict[str, Job] = {}
        self._lock = asyncio.Lock()

    async def create(self, query: str) -> Job:
        job = Job(
            job_id=str(uuid.uuid4()),
            status=JobStatus.PENDING,
            query=query,
            created_at=datetime.now(timezone.utc),
        )
        async with self._lock:
            self._jobs[job.job_id] = job
        return job

    async def get(self, job_id: str) -> Job | None:
        async with self._lock:
            return self._jobs.get(job_id)

    async def mark_running(self, job_id: str) -> None:
        async with self._lock:
            if job_id in self._jobs:
                self._jobs[job_id].status = JobStatus.RUNNING

    async def mark_complete(self, job_id: str, result: str) -> None:
        async with self._lock:
            if job_id in self._jobs:
                job = self._jobs[job_id]
                job.status = JobStatus.COMPLETE
                job.result = result
                job.completed_at = datetime.now(timezone.utc)

    async def mark_failed(self, job_id: str, error: str) -> None:
        async with self._lock:
            if job_id in self._jobs:
                job = self._jobs[job_id]
                job.status = JobStatus.FAILED
                job.error = error
                job.completed_at = datetime.now(timezone.utc)


job_store = JobStore()
