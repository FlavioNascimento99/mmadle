import Image from "next/image";
import { initials } from "@/lib/fighter";

const SIZES = {
  sm: { px: 40, box: "h-10 w-10 border-2", text: "text-base" },
  md: { px: 88, box: "h-[5.5rem] w-[5.5rem] border-3", text: "text-3xl" },
  lg: { px: 224, box: "h-56 w-56 border-4", text: "text-7xl" },
} as const;

type Props = {
  name: string;
  url?: string | null;
  credit?: string | null;
  size: keyof typeof SIZES;
  className?: string;
};

export function FighterPhoto({ name, url, credit, size, className = "" }: Props) {
  const s = SIZES[size];
  return (
    <div className={`relative shrink-0 overflow-hidden border-ink bg-bruise ${s.box} ${className}`}>
      {url ? (
        <Image
          src={url}
          alt={name}
          title={credit ? `Photo: ${credit}` : undefined}
          width={s.px}
          height={s.px}
          className="h-full w-full object-cover object-top grayscale-[35%] contrast-125"
        />
      ) : (
        <span
          role="img"
          aria-label={`${name} (no photo)`}
          className={`flex h-full w-full items-center justify-center font-display text-bone ${s.text}`}
        >
          {initials(name)}
        </span>
      )}
    </div>
  );
}
