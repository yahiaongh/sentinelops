"use client";

import { useEffect, useState } from "react";
import { StatusPulse } from "./StatusPulse";
import type { ServiceHealth } from "@/lib/types";

function healthStatus(errorRate: number): "healthy" | "warning" | "critical" {
  if (errorRate >= 0.3) return "critical";
  if (errorRate >= 0.1) return "warning";
  return "healthy";
}

export function ServiceHealthGrid() {
  const [services, setServices] = useState<ServiceHealth[] | null>(null);
  const [error, setError] = useState(false);

  useEffect(() => {
    let cancelled = false;

    async function fetchServices() {
      try {
        const res = await fetch("/api/services?hours=1");
        if (!res.ok) throw new Error("request failed");
        const data = await res.json();
        if (!cancelled) {
          setServices(data);
          setError(false);
        }
      } catch {
        if (!cancelled) setError(true);
      }
    }

    fetchServices();
    const interval = setInterval(fetchServices, 15_000);
    return () => {
      cancelled = true;
      clearInterval(interval);
    };
  }, []);

  if (error) {
    return (
      <p className="rounded-lg border border-border bg-surface p-4 font-mono text-sm text-critical">
        Can&apos;t reach service stats. Retrying every 15s.
      </p>
    );
  }

  if (services === null) {
    return (
      <p className="rounded-lg border border-border bg-surface p-4 font-mono text-sm text-muted">
        Loading…
      </p>
    );
  }

  return (
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
      {services.map((s) => {
        const status = healthStatus(s.error_rate);
        return (
          <div key={s.service} className="rounded-lg border border-border bg-surface p-4">
            <div className="flex items-center justify-between">
              <h3 className="font-medium text-text">{s.service}</h3>
              <StatusPulse status={status} />
            </div>
            <dl className="mt-3 space-y-1.5 font-mono text-xs">
              <div className="flex justify-between">
                <dt className="text-muted">events (1h)</dt>
                <dd className="text-text">{s.total_events.toLocaleString()}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-muted">error rate</dt>
                <dd
                  className={
                    status === "critical"
                      ? "text-critical"
                      : status === "warning"
                        ? "text-warning"
                        : "text-healthy"
                  }
                >
                  {(s.error_rate * 100).toFixed(1)}%
                </dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-muted">p95 latency</dt>
                <dd className="text-text">
                  {s.avg_p95_latency_ms !== null ? `${s.avg_p95_latency_ms.toFixed(0)}ms` : "—"}
                </dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-muted">max latency</dt>
                <dd className="text-text">
                  {s.max_latency_ms !== null ? `${s.max_latency_ms.toFixed(0)}ms` : "—"}
                </dd>
              </div>
            </dl>
          </div>
        );
      })}
      {services.length === 0 && (
        <p className="col-span-full rounded-lg border border-border bg-surface p-4 font-mono text-sm text-muted">
          No service activity in the last hour.
        </p>
      )}
    </div>
  );
}