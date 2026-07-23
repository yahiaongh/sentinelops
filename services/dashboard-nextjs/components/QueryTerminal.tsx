"use client";

import { useEffect, useRef, useState } from "react";
import type { JobStatus } from "@/lib/types";

interface HistoryEntry {
  query: string;
  status: JobStatus;
  result: string | null;
  error: string | null;
}

export function QueryTerminal() {
  const [input, setInput] = useState("");
  const [history, setHistory] = useState<HistoryEntry[]>([]);
  const [activeJobId, setActiveJobId] = useState<string | null>(null);
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight });
  }, [history]);

  useEffect(() => {
    if (!activeJobId) return;

    const interval = setInterval(async () => {
      const res = await fetch(`/api/query/${activeJobId}`);
      const job = await res.json();

      setHistory((prev) =>
        prev.map((entry, i) =>
          i === prev.length - 1
            ? { ...entry, status: job.status, result: job.result, error: job.error }
            : entry
        )
      );

      if (job.status === "complete" || job.status === "failed") {
        setActiveJobId(null);
        clearInterval(interval);
      }
    }, 3000);

    return () => clearInterval(interval);
  }, [activeJobId]);

  async function submitQuery(e: React.FormEvent) {
    e.preventDefault();
    const query = input.trim();
    if (!query || activeJobId) return;

    setInput("");
    setHistory((prev) => [...prev, { query, status: "pending", result: null, error: null }]);

    try {
      const res = await fetch("/api/query", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ query }),
      });
      const data = await res.json();
      setActiveJobId(data.job_id);
    } catch {
      setHistory((prev) =>
        prev.map((entry, i) =>
          i === prev.length - 1
            ? { ...entry, status: "failed", error: "Could not reach the query service." }
            : entry
        )
      );
    }
  }

  return (
    <div className="flex h-full flex-col rounded-lg border border-border bg-surface">
      <div className="border-b border-border px-4 py-3">
        <h2 className="font-mono text-sm font-medium text-text">incident_query</h2>
      </div>

      <div ref={scrollRef} className="flex-1 overflow-y-auto p-4 font-mono text-sm">
        {history.length === 0 && (
          <p className="text-muted">
            Ask about recent incidents — e.g. &quot;what happened to checkout in the last
            hour?&quot;
          </p>
        )}
        {history.map((entry, i) => (
          <div key={i} className="mb-4">
            <div className="text-accent">
              <span className="text-muted">sentinelops&gt;</span> {entry.query}
            </div>
            <div className="mt-1 whitespace-pre-wrap text-text">
              {entry.status === "pending" || entry.status === "running" ? (
                <ThinkingIndicator />
              ) : entry.status === "failed" ? (
                <span className="text-critical">{entry.error ?? "Query failed."}</span>
              ) : (
                entry.result
              )}
            </div>
          </div>
        ))}
      </div>

      <form onSubmit={submitQuery} className="border-t border-border p-3">
        <div className="flex items-center gap-2 font-mono text-sm">
          <span className="text-accent">sentinelops&gt;</span>
          <input
            value={input}
            onChange={(e) => setInput(e.target.value)}
            disabled={activeJobId !== null}
            placeholder={
              activeJobId ? "waiting on previous query…" : "ask about incidents…"
            }
            className="flex-1 bg-transparent text-text placeholder:text-muted focus:outline-none disabled:opacity-50"
          />
        </div>
      </form>
    </div>
  );
}

function ThinkingIndicator() {
  return (
    <span className="text-muted">
      computing
      <span className="animate-pulse">▊</span>
      <span className="ml-2 text-xs">
        (local inference can take 30-90s on this hardware)
      </span>
    </span>
  );
}