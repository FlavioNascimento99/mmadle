import { useEffect, type RefObject } from "react";

export function useClickOutside(ref: RefObject<HTMLElement>, onOutside: () => void) {
  useEffect(() => {
    const onMouseDown = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) onOutside();
    };
    document.addEventListener("mousedown", onMouseDown);
    return () => document.removeEventListener("mousedown", onMouseDown);
  }, [ref, onOutside]);
}
