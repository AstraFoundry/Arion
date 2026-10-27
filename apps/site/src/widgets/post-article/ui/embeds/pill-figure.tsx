import { DictationPill, type PillState } from "@/shared/ui";

interface PillFigureProps {
  state: PillState["kind"];
  draft?: string;
  term?: string;
  /** What the illustration shows, for screen readers. */
  label: string;
  caption?: string;
}

function pillState({ state, draft, term }: PillFigureProps): PillState {
  if (state === "listening") return { kind: "listening", draft: draft ?? "" };
  if (state === "learned") return { kind: "learned", term: term ?? "" };
  return { kind: "inserted" };
}

/**
 * `<dictation-pill>`: the site's live pill component inside a post, on the
 * same dark stage as the home page illustrations. Like every product mock it
 * is a picture: labelled, and inert.
 */
export function PillFigure(props: PillFigureProps) {
  return (
    <figure className="post-wide post-media">
      <div role="img" aria-label={props.label} className="post-pill-stage">
        <div inert>
          <DictationPill state={pillState(props)} />
        </div>
      </div>
      {props.caption ? <figcaption>{props.caption}</figcaption> : null}
    </figure>
  );
}
