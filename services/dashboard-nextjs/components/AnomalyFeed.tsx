"use client";

import { useEffect, useState } from "react";
import { SeverityBadge } from "./SeverityBadge";
import type { Anomaly } from "@/lib/types";

function timeAgo(isoString: string): string {
  const seconds = Math.floor((Date.now() - new Date(isoString).getTime()) / 1000);
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  return `${hours}h ago`;
}

export function AnomalyFeed() {
  const [anomalies, setAnomalies] = useState<Anomaly[] | null>(null);
  const [error, setError] = useState(false);

  useEffect(() => {
    let cancelled = false;

    async function fetchAnomalies() {
      try {
        const res = await fetch("/api/anomalies?hours=24&limit=30");
        if (!res.ok) throw new Error("request failed");
        const data = await res.json();
        if (!cancelled) {
          setAnomalies(data);
          setError(false);
        }
      } catch {
        if (!cancelled) setError(true);
      }
    }

    fetchAnomalies();
    const interval = setInterval(fetchAnomalies, 10_000);
    return () => {
      cancelled = true;
      clearInterval(interval);
    };
  }, []);

  return (
    <div className="flex h-full flex-col rounded-lg border border-border bg-surface">
      <div className="flex items-center justify-between border-b border-border px-4 py-3">
        <h2 className="font-mono text-sm font-medium text-text">anomaly_feed</h2>
        <span className="font-mono text-xs text-muted">last 24h</span>
      </div>
      <div className="flex-1 overflow-y-auto p-2">
        {error && (
          <p className="p-4 font-mono text-sm text-critical">
            Can&apos;t reach the anomaly feed. Retrying every 10s.
          </p>
        )}
        {!error && anomalies === null && (
          <p className="p-4 font-mono text-sm text-muted">Loading…</p>
        )}
        {!error && anomalies?.length === 0 && (
          <p className="p-4 font-mono text-sm text-muted">
            No anomalies detected in the last 24 hours. All quiet.
          </p>
        )}
        {anomalies?.map((a) => (
          <div
            key={a.id}
            className="flex items-start gap-3 border-b border-border/50 px-2 py-2.5 last:border-0"
          >
            <span className="mt-0.5 shrink-0 font-mono text-xs text-muted">
              {timeAgo(a.detected_at)}
            </span>
            <SeverityBadge severity={a.severity} />
            <div className="min-w-0 flex-1 font-mono text-xs text-text">
              <span className="text-accent">{a.service}</span>
              <span className="text-muted"> · {a.anomaly_type} · </span>
              <span>
                value={a.metric_value.toFixed(2)} baseline={a.baseline_mean.toFixed(2)} z=
                {a.z_score.toFixed(2)}
              </span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}