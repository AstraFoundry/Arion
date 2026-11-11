/** The attribute of `<html>` that marks a page whose window is not drawn as
 * active, for CSS: `:root[data-mygo-window-inactive] .sidebar { … }`. */
export const INACTIVE_ATTRIBUTE = "data-mygo-window-inactive";

/** Marks root, a page's `<html>`, as of a window drawn as active or not. */
export function markWindowActive(root: Pick<Element, "toggleAttribute"> | null | undefined, active: boolean): void {
  root?.toggleAttribute(INACTIVE_ATTRIBUTE, !active);
}
