import { PoolSchema, type Pool } from "./api";

export const POOL_STORAGE_KEY = "mmadle-pool";

/** Restores the saved mode; anything missing or unknown means all fighters. */
export function parseStoredPool(raw: string | null): Pool {
  const parsed = PoolSchema.safeParse(raw);
  return parsed.success ? parsed.data : "all";
}
