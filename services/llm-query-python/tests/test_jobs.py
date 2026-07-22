import pytest
from app.jobs import JobStatus, JobStore


@pytest.mark.asyncio
async def test_job_lifecycle_success():
    store = JobStore()
    job = await store.create("what happened?")
    assert job.status == JobStatus.PENDING
    assert job.result is None

    await store.mark_running(job.job_id)
    running = await store.get(job.job_id)
    assert running.status == JobStatus.RUNNING

    await store.mark_complete(job.job_id, "here is the answer")
    done = await store.get(job.job_id)
    assert done.status == JobStatus.COMPLETE
    assert done.result == "here is the answer"
    assert done.completed_at is not None


@pytest.mark.asyncio
async def test_job_lifecycle_failure():
    store = JobStore()
    job = await store.create("bad query")
    await store.mark_running(job.job_id)
    await store.mark_failed(job.job_id, "ollama unreachable")

    failed = await store.get(job.job_id)
    assert failed.status == JobStatus.FAILED
    assert failed.error == "ollama unreachable"
    assert failed.result is None


@pytest.mark.asyncio
async def test_get_unknown_job_returns_none():
    store = JobStore()
    result = await store.get("does-not-exist")
    assert result is None
