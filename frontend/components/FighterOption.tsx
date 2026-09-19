import { forwardRef } from "react";
import type { SearchResult } from "@/lib/api";
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
  return (
    <li className="border-b-2 border-ink last:border-b-0">
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
        className={`flex w-full items-center gap-3 px-3 py-2 text-left text-ink hover:bg-blood hover:text-bone focus-visible:bg-blood focus-visible:text-bone focus-visible:outline-none disabled:opacity-40 ${
          active ? "bg-blood text-bone" : ""
        }`}
      >
        <FighterPhoto name={fighter.name} url={fighter.photo_url} credit={fighter.photo_credit} size="sm" />
        <span className="min-w-0">
          <span className="block truncate font-bold">
            {fighter.name}
            {guessed ? " (guessed)" : ""}
          </span>
          <span className="block truncate text-xs opacity-75">
            {[fighter.nickname && `“${fighter.nickname}”`, fighter.division, fighter.nationality]
              .filter(Boolean)
              .join(", ")}
          </span>
        </span>
      </button>
    </li>
  );
});
