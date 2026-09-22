import { forwardRef } from "react";
import type { SearchResult } from "@/lib/api";
import { useLang } from "@/lib/i18n";
import { FighterPhoto } from "./FighterPhoto";

type Props = {
  fighter: SearchResult;
  guessed: boolean;
  /** Keyboard/mouse highlight (mirrors focus). */
  active?: boolean;
  optionId?: string;
  onPick: (fighter: SearchResult) => void;
  onHighlight?: () => void;
};

export const FighterOption = forwardRef<HTMLButtonElement, Props>(function FighterOption(
  { fighter, guessed, active = false, optionId, onPick, onHighlight }: Props,
  ref,
) {
  const { t } = useLang();
  return (
    <li className="border-b-2 border-ink/10 last:border-b-0">
      <button
        type="button"
        ref={ref}
        id={optionId}
        role="option"
        aria-selected={active}
        disabled={guessed}
        tabIndex={active ? 0 : -1}
        onClick={() => onPick(fighter)}
        onMouseMove={onHighlight}
        onFocus={onHighlight}
        className={`flex w-full items-center gap-3 px-3 py-2 text-left focus-visible:outline-none disabled:opacity-40 ${
          active ? "bg-ink text-bone" : "text-ink hover:bg-paper focus-visible:bg-paper"
        }`}
      >
        <FighterPhoto name={fighter.name} url={fighter.photo_url} credit={fighter.photo_credit} size="sm" />
        <span className="min-w-0">
          <span className="block truncate font-display text-lg uppercase leading-tight tracking-wide">
            {fighter.name}
            {guessed ? t("option.guessed") : ""}
          </span>
          <span className="block truncate font-mono text-[11px] tracking-wide opacity-60">
            {[fighter.nickname && `“${fighter.nickname}”`, fighter.division, fighter.nationality]
              .filter(Boolean)
              .join(" · ")}
          </span>
        </span>
      </button>
    </li>
  );
});
