import { PoolSchema, type Pool } from "./api";

export const POOL_STORAGE_KEY = "mmadle-pool";

export const POOL_OPTIONS: { value: Pool; label: string }[] = [
  { value: "all", label: "Men & women" },
  { value: "men", label: "Men only" },
];

/** Restores the saved mode; anything missing or unknown means all fighters. */
export function parseStoredPool(raw: string | null): Pool {
  const parsed = PoolSchema.safeParse(raw);
  return parsed.success ? parsed.data : "all";
}
