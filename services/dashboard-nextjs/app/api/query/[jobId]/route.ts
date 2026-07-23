import { NextResponse } from "next/server";

const LLM_SERVICE_URL = process.env.LLM_SERVICE_URL ?? "http://localhost:9300";

export async function GET(
  request: Request,
  { params }: { params: Promise<{ jobId: string }> }
) {
  const { jobId } = await params;
  try {
    const response = await fetch(`${LLM_SERVICE_URL}/query/${jobId}`);
    const data = await response.json();
    return NextResponse.json(data, { status: response.status });
  } catch (error) {
    console.error("Failed to fetch query result", error);
    return NextResponse.json({ error: "LLM service unreachable" }, { status: 502 });
  }
}