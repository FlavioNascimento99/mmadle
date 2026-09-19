import type { GuessOutcome } from "@/lib/api";
import { useLang } from "@/lib/i18n";
import { FighterPhoto } from "./FighterPhoto";

type Props = {
  winner: GuessOutcome;
  attempts: number;
  copied: boolean;
  onShare: () => void;
};

export function WinReveal({ winner, attempts, copied, onShare }: Props) {
  const { t } = useLang();
  const tries = t(attempts === 1 ? "win.tryOne" : "win.tryMany", { n: attempts });
  return (
    <section
      aria-label={t("win.today")}
      className="flex flex-col items-center gap-8 border-4 border-ink bg-blood p-6 text-ink shadow-[8px_8px_0_0_#FAFAF7] sm:flex-row sm:items-center sm:p-8"
    >
      <figure className="-rotate-2 shadow-hard-lg">
        <FighterPhoto name={winner.fighter_name} url={winner.photo_url} credit={winner.photo_credit} size="lg" />
        {winner.photo_credit && (
          <figcaption className="max-w-56 border-4 border-t-0 border-ink bg-bone px-2 py-1 text-[11px] leading-tight text-ink">
            {t("win.photoOf", { credit: winner.photo_credit })}
          </figcaption>
        )}
      </figure>
      <div className="min-w-0 text-center sm:text-left">
        <p className="font-semibold">{t("win.solvedIn", { tries })}</p>
        <h2 className="mt-1 break-words font-display text-5xl uppercase leading-none text-bone sm:text-6xl">
          {winner.fighter_name}
        </h2>
        <p className="mt-3">{t("win.tomorrow")}</p>
        <button
          type="button"
          onClick={onShare}
          className="press mt-5 border-3 border-ink bg-ink px-5 py-2.5 font-display text-lg uppercase tracking-wide text-bone shadow-[4px_4px_0_0_#FAFAF7]"
        >
          {copied ? t("win.copied") : t("win.share")}
        </button>
      </div>
    </section>
  );
}
