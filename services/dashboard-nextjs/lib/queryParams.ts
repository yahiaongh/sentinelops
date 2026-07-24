/**
 * Parses a numeric query parameter, falling back to a default when missing,
 * non-numeric, or non-positive. Used across API routes instead of a bare
 * `Number(x ?? default)`, which silently produces NaN or accepts negative/zero
 * values for parameters like `hours` or `limit` that must be positive.
 */
export function parsePositiveNumberParam(value: string | null, fallback: number): number {
  if (value === null) return fallback;
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
}