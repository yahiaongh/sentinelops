import { NextResponse } from "next/server";
import { pool } from "@/lib/db";
import { parseHoursParam, toServiceHealth } from "@/lib/serviceHealth";

export async function GET(request: Request) {
  const { searchParams } = new URL(request.url);
  const hours = parseHoursParam(searchParams.get("hours"), 1);

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
    return NextResponse.json(toServiceHealth(result.rows));
  } catch (error) {
    console.error("Failed to fetch service health", error);
    return NextResponse.json({ error: "Failed to fetch service health" }, { status: 500 });
  }
}