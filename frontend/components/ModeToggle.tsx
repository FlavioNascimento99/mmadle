import type { Pool } from "@/lib/api";
import { POOL_OPTIONS } from "@/lib/pool";

type Props = {
  pool: Pool;
  disabled: boolean;
  onChange: (pool: Pool) => void;
};

export function ModeToggle({ pool, disabled, onChange }: Props) {
  return (
    <div role="group" aria-label="Game mode" className="inline-flex border-3 border-ink shadow-blood">
      {POOL_OPTIONS.map((option) => {
        const active = option.value === pool;
        return (
          <button
            key={option.value}
            type="button"
            aria-pressed={active}
            disabled={disabled}
            onClick={() => !active && onChange(option.value)}
            className={`px-4 py-2 text-sm font-bold uppercase tracking-wide first:border-r-3 first:border-ink disabled:opacity-50 ${
              active ? "bg-blood text-bone" : "bg-bone text-ink hover:bg-steel"
            }`}
          >
            {option.label}
          </button>
        );
      })}
    </div>
  );
}
