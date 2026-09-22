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
      className="animate-pop-in overflow-hidden border-3 border-ink bg-bone text-ink shadow-hard"
    >
      <p className="microlabel border-b-3 border-ink bg-blood px-4 py-1.5 text-bone">
        ✓ {t("win.today")}
      </p>
      <div className="flex flex-col gap-5 p-4 sm:flex-row sm:items-center sm:gap-6 sm:p-5">
        <figure className="flex shrink-0 items-center gap-3 sm:block">
          <div className="-rotate-2 border-3 border-ink shadow-hard-sm">
            <FighterPhoto name={winner.fighter_name} url={winner.photo_url} credit={winner.photo_credit} size="lg" />
          </div>
          {winner.photo_credit && (
            <figcaption className="max-w-56 font-mono text-[10px] leading-tight tracking-wide text-ash sm:mt-2 sm:border-2 sm:border-ink sm:bg-paper sm:px-2 sm:py-1">
              {t("win.photoOf", { credit: winner.photo_credit })}
            </figcaption>
          )}
        </figure>
        <div className="min-w-0 flex-1 text-center sm:text-left">
          <div className="flex items-end justify-center gap-3 sm:justify-start">
            <span className="font-display text-7xl leading-none">{attempts}</span>
            <span className="microlabel pb-1.5 text-ash">{tries}</span>
          </div>
          <h2 className="mt-2 break-words font-display text-4xl uppercase leading-[0.95] sm:text-5xl">
            {winner.fighter_name}
          </h2>
          <p className="mt-2 text-sm text-ash">{t("win.tomorrow")}</p>
          <button
            type="button"
            onClick={onShare}
            className="press mt-4 border-3 border-ink bg-ink px-5 py-2.5 font-mono text-xs font-bold uppercase tracking-[0.18em] text-bone"
          >
            {copied ? t("win.copied") : t("win.share")}
          </button>
        </div>
      </div>
    </section>
  );
}
