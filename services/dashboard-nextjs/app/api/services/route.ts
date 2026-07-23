import { NextResponse } from "next/server";
import { pool } from "@/lib/db";
import type { ServiceHealth } from "@/lib/types";

export async function GET(request: Request) {
  const { searchParams } = new URL(request.url);
  const hours = Number(searchParams.get("hours") ?? "1");

  try {
    const result = await pool.query(
      `SELECT service,
              sum(event_count) AS total_events,
              sum(error_count) AS total_errors,
              avg(p95_latency_ms) AS avg_p95_latency_ms,
              max(max_latency_ms) AS max_latency_ms
       FROM log_events_1min
       WHERE bucket >= now() - ($1 || ' hours')::interval
       GROUP BY service
       ORDER BY service`,
      [hours]
    );

    const services: ServiceHealth[] = result.rows.map((r) => {
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

    return NextResponse.json(services);
  } catch (error) {
    console.error("Failed to fetch service health", error);
    return NextResponse.json({ error: "Failed to fetch service health" }, { status: 500 });
  }
}