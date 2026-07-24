import type { ServiceHealth } from "@/lib/types";

interface RawServiceStatsRow {
  service: string;
  total_events: string | number | null;
  total_errors: string | number | null;
  avg_p95_latency_ms: string | number | null;
  max_latency_ms: string | number | null;
}

/**
 * Transforms raw aggregate rows from log_events_1min into ServiceHealth
 * objects, computing error_rate safely (no division by zero for services
 * with no events in the window).
 */
export function toServiceHealth(rows: RawServiceStatsRow[]): ServiceHealth[] {
  return rows.map((r) => {
    const totalEvents = Number(r.total_events) || 0;
    const totalErrors = Number(r.total_errors) || 0;
    return {
      service: r.service,
      total_events: totalEvents,
      total_errors: totalErrors,
      error_rate: totalEvents > 0 ? totalErrors / totalEvents : 0,
      avg_p95_latency_ms: r.avg_p95_latency_ms !== null ? Number(r.avg_p95_latency_ms) : null,
      max_latency_ms: r.max_latency_ms !== null ? Number(r.max_latency_ms) : null,
    };
  });
}

export { parsePositiveNumberParam as parseHoursParam } from "./queryParams";