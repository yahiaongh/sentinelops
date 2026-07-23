import { Pool } from "pg";

// A single shared pool per process, matching the pattern used by our other
// services (Go's pgxpool, Python's asyncpg). Next.js API routes run in a
// long-lived Node process in this deployment (not edge/serverless), so a
// module-level singleton is correct here rather than reconnecting per request.
declare global {
  var _pgPool: Pool | undefined;
}

export const pool =
  global._pgPool ??
  new Pool({
    connectionString:
      process.env.DATABASE_URL ??
      "postgresql://sentinelops:devpassword@localhost:5432/sentinelops",
    max: 5,
  });

if (process.env.NODE_ENV !== "production") {
  global._pgPool = pool;
}