import type { Pool } from "@/lib/api";
import { useLang } from "@/lib/i18n";

type Props = {
  pool: Pool;
  disabled: boolean;
  onChange: (pool: Pool) => void;
};

export function ModeToggle({ pool, disabled, onChange }: Props) {
  const { t } = useLang();
  const options: { value: Pool; label: string }[] = [
    { value: "all", label: t("pool.all") },
    { value: "men", label: t("pool.men") },
  ];
  return (
    <div role="group" aria-label={t("pool.group")} className="inline-flex border-3 border-ink shadow-blood">
      {options.map((option) => {
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
