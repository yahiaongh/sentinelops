import { describe, expect, it } from "vitest";
import { parseHoursParam, toServiceHealth } from "./serviceHealth";

describe("toServiceHealth", () => {
  it("computes error rate correctly from string-typed pg aggregate results", () => {
    const result = toServiceHealth([
      {
        service: "checkout",
        total_events: "200",
        total_errors: "20",
        avg_p95_latency_ms: "150.5",
        max_latency_ms: "300.25",
      },
    ]);
    expect(result).toEqual([
      {
        service: "checkout",
        total_events: 200,
        total_errors: 20,
        error_rate: 0.1,
        avg_p95_latency_ms: 150.5,
        max_latency_ms: 300.25,
      },
    ]);
  });

  it("does not divide by zero when there are no events", () => {
    const result = toServiceHealth([
      {
        service: "idle-service",
        total_events: 0,
        total_errors: 0,
        avg_p95_latency_ms: null,
        max_latency_ms: null,
      },
    ]);
    expect(result[0].error_rate).toBe(0);
    expect(result[0].avg_p95_latency_ms).toBeNull();
    expect(result[0].max_latency_ms).toBeNull();
  });

  it("treats null total_events/total_errors as zero", () => {
    const result = toServiceHealth([
      {
        service: "new-service",
        total_events: null,
        total_errors: null,
        avg_p95_latency_ms: null,
        max_latency_ms: null,
      },
    ]);
    expect(result[0].total_events).toBe(0);
    expect(result[0].total_errors).toBe(0);
    expect(result[0].error_rate).toBe(0);
  });

  it("handles multiple services independently", () => {
    const result = toServiceHealth([
      { service: "a", total_events: 100, total_errors: 50, avg_p95_latency_ms: 1, max_latency_ms: 2 },
      { service: "b", total_events: 100, total_errors: 1, avg_p95_latency_ms: 1, max_latency_ms: 2 },
    ]);
    expect(result[0].error_rate).toBe(0.5);
    expect(result[1].error_rate).toBe(0.01);
  });
});

describe("parseHoursParam", () => {
  it("returns the fallback when value is null", () => {
    expect(parseHoursParam(null, 24)).toBe(24);
  });

  it("parses a valid numeric string", () => {
    expect(parseHoursParam("6", 24)).toBe(6);
  });

  it("returns the fallback for non-numeric input", () => {
    expect(parseHoursParam("not-a-number", 24)).toBe(24);
  });

  it("returns the fallback for zero or negative values", () => {
    expect(parseHoursParam("0", 24)).toBe(24);
    expect(parseHoursParam("-5", 24)).toBe(24);
  });
});