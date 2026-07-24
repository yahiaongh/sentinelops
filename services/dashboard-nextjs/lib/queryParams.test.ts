import { describe, expect, it } from "vitest";
import { parsePositiveNumberParam } from "./queryParams";

describe("parsePositiveNumberParam", () => {
  it("returns the fallback when value is null", () => {
    expect(parsePositiveNumberParam(null, 50)).toBe(50);
  });

  it("parses a valid numeric string", () => {
    expect(parsePositiveNumberParam("30", 50)).toBe(30);
  });

  it("returns the fallback for non-numeric input", () => {
    expect(parsePositiveNumberParam("abc", 50)).toBe(50);
  });

  it("returns the fallback for zero or negative values", () => {
    expect(parsePositiveNumberParam("0", 50)).toBe(50);
    expect(parsePositiveNumberParam("-10", 50)).toBe(50);
  });

  it("returns the fallback for empty string", () => {
    expect(parsePositiveNumberParam("", 50)).toBe(50);
  });
});