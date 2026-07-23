import { NextResponse } from "next/server";
import { pool } from "@/lib/db";
import type { Anomaly } from "@/lib/types";

export async function GET(request: Request) {
  const { searchParams } = new URL(request.url);
  const hours = Number(searchParams.get("hours") ?? "24");
  const limit = Number(searchParams.get("limit") ?? "50");

  try {
    const result = await pool.query<Anomaly>(
      `SELECT id, detected_at, bucket_ts, service, anomaly_type,
              metric_value, baseline_mean, baseline_stddev, z_score, severity
       FROM anomalies
       WHERE detected_at >= now() - ($1 || ' hours')::interval
       ORDER BY detected_at DESC
       LIMIT $2`,
      [hours, limit]
    );
    return NextResponse.json(result.rows);
  } catch (error) {
    console.error("Failed to fetch anomalies", error);
    return NextResponse.json({ error: "Failed to fetch anomalies" }, { status: 500 });
  }
}