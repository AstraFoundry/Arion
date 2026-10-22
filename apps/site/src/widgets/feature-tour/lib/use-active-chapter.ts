"use client";

import { useCallback, useEffect, useRef, useState } from "react";

interface ActiveChapter {
  index: number;
  /** 1 when the reader moved down to this chapter, -1 when up. */
  direction: 1 | -1;
}

/**
 * Tracks which chapter crosses the middle of the viewport. Each chapter
 * element is registered with `register(index)` and carries `data-chapter`.
 */
export function useActiveChapter() {
  const elements = useRef(new Map<number, Element>());
  const [active, setActive] = useState<ActiveChapter>({ index: 0, direction: 1 });

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (!entry.isIntersecting) continue;
          const index = Number((entry.target as HTMLElement).dataset.chapter);
          setActive((previous) =>
            index === previous.index ? previous : { index, direction: index > previous.index ? 1 : -1 },
          );
        }
      },
      { rootMargin: "-50% 0px -50% 0px" },
    );
    for (const element of elements.current.values()) observer.observe(element);
    return () => observer.disconnect();
  }, []);

  const register = useCallback(
    (index: number) => (element: HTMLElement | null) => {
      if (element) elements.current.set(index, element);
      else elements.current.delete(index);
    },
    [],
  );

  return { ...active, register };
}
