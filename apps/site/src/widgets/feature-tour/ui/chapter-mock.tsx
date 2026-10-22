"use client";

import type { ComponentType } from "react";
import { useCopy } from "@/shared/i18n";
import type { ChapterId } from "../config/chapters";
import { KeyboardMock } from "./mocks/keyboard-mock";
import { LearningMock } from "./mocks/learning-mock";
import { MixedLanguageMock } from "./mocks/mixed-language-mock";
import { OfflineMock } from "./mocks/offline-mock";
import { PolishMock } from "./mocks/polish-mock";

const MOCKS: Record<ChapterId, ComponentType<{ playing: boolean }>> = {
  offline: OfflineMock,
  mixed: MixedLanguageMock,
  learning: LearningMock,
  polish: PolishMock,
  keyboard: KeyboardMock,
};

/**
 * One chapter's product picture. It is an image to assistive technology,
 * labelled from the dictionary, and its content is inert.
 */
export function ChapterMock({ id, playing }: { id: ChapterId; playing: boolean }) {
  const { tour } = useCopy();
  const Mock = MOCKS[id];
  return (
    <div role="img" aria-label={tour[id].mockLabel} className="flex w-full justify-center">
      <div inert className="flex w-full justify-center select-none">
        <Mock playing={playing} />
      </div>
    </div>
  );
}
