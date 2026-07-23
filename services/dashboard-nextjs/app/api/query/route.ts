import { NextResponse } from "next/server";

const LLM_SERVICE_URL = process.env.LLM_SERVICE_URL ?? "http://localhost:9300";

export async function POST(request: Request) {
  const body = await request.json();
  try {
    const response = await fetch(`${LLM_SERVICE_URL}/query`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    const data = await response.json();
    return NextResponse.json(data, { status: response.status });
  } catch (error) {
    console.error("Failed to submit query", error);
    return NextResponse.json({ error: "LLM service unreachable" }, { status: 502 });
  }
}