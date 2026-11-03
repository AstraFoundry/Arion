import type { ReactNode } from "react";
import { AuthorAvatar, type AuthorProfile } from "@/shared/ui";

/** Profiles as `rehypeGitHubPeople` resolved them at build time. */
const parsePeople = (people: string): AuthorProfile[] => JSON.parse(people) as AuthorProfile[];

/**
 * `<github-people>`: a named group of GitHub accounts, each avatar and name
 * linked to the profile, such as a release's contributors.
 */
export function GitHubPeople({ label, people }: { label: string; people: string }) {
  return (
    <section className="post-people" aria-label={label}>
      <h3 className="post-people-label">{label}</h3>
      <ul>
        {parsePeople(people).map((person) => (
          <li key={person.profileUrl}>
            <a href={person.profileUrl}>
              <AuthorAvatar author={person} />
              {person.name}
            </a>
          </li>
        ))}
      </ul>
    </section>
  );
}

/** `<voice-quote>`: what someone said, with their GitHub avatar and name. */
export function VoiceQuote({ people, children }: { people: string; children?: ReactNode }) {
  const [person] = parsePeople(people);
  return (
    <figure className="post-voice">
      <blockquote>{children}</blockquote>
      <figcaption>
        <a href={person.profileUrl}>
          <AuthorAvatar author={person} />
          {person.name}
        </a>
      </figcaption>
    </figure>
  );
}
