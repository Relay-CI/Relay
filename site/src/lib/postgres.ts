import { Pool, type QueryResultRow } from "pg";
import { siteEnv } from "@/lib/env";

declare global {
  var __relaySitePgPool__: Pool | undefined;
}

function createPool() {
  return new Pool({
    connectionString: siteEnv.databaseUrl,
    ssl:
      process.env.PGSSL === "disable"
        ? false
        : process.env.NODE_ENV === "production"
          ? { rejectUnauthorized: false }
          : false,
    max: 10,
  });
}

export function isDatabaseConfigured() {
  return true;
}

export function getPool() {
  if (!globalThis.__relaySitePgPool__) {
    globalThis.__relaySitePgPool__ = createPool();
  }

  return globalThis.__relaySitePgPool__;
}

export async function query<T extends QueryResultRow>(
  text: string,
  params: readonly unknown[] = [],
) {
  const pool = getPool();
  return pool.query<T>(text, [...params]);
}
