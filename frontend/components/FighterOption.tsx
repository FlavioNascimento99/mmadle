import type { SearchResult } from "@/lib/api";
import { FighterPhoto } from "./FighterPhoto";

type Props = {
  fighter: SearchResult;
  guessed: boolean;
  onPick: (fighter: SearchResult) => void;
};

export function FighterOption({ fighter, guessed, onPick }: Props) {
  return (
    <li className="border-b-2 border-ink last:border-b-0">
      <button
        type="button"
        role="option"
        aria-selected="false"
        disabled={guessed}
        onClick={() => onPick(fighter)}
        className="flex w-full items-center gap-3 px-3 py-2 text-left text-ink hover:bg-blood hover:text-bone focus-visible:bg-blood focus-visible:text-bone focus-visible:outline-none disabled:opacity-40"
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
}
