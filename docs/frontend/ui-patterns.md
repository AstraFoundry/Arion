# UI Patterns
Arion native mapping: use Go package exports and MyGo native primitives; TypeScript, DOM and CSS examples illustrate the original generic conventions. See [repository ownership](../project/repository.md) for the authoritative native mapping.


This document defines how UI is written in this project. The goal is a consistent, maintainable, and accessible interface.

## Language

- English is the default interface language; Simplified Chinese is supported through the approved [i18n catalogs](i18n.md). Add every message to both catalogs; do not embed text in views.
- All code, comments, prop names, and CSS custom properties must be in English.

## No hardcoded strings

- Every user-facing string must come from a centralized source: i18n keys, design-system tokens, or a constants file.
- Do not write labels, placeholders, or error messages directly in components.

```tsx
// ✅ Good
import { t } from 'shared/i18n';

<Button>{t('order.submit')}</Button>
```

```tsx
// ❌ Bad
<Button>Submit order</Button>
```

## No redundant copy

- Do not repeat information already shown by a title, icon, selected state, or surrounding section.
- Prefer concise labels over explanatory text when the state is self-evident.
- Remove disabled placeholder actions unless they teach a real next step.

```tsx
// ❌ Bad
<div>
  <h1>Orders</h1>
  <p>This section shows your orders.</p>
</div>
```

```tsx
// ✅ Good
<div>
  <h1>{t('orders.title')}</h1>
  <OrderList />
</div>
```

## Semantic styling

- Use semantic design tokens instead of physical colors.
- ❌ Avoid: `bg-white`, `text-black`, `zinc-500`.
- ✅ Prefer: `bg-bg`, `text-text`, `border-border`, `text-muted`.
- Theme differences must live in one place (global theme file or CSS variables). Do not scatter `dark:` or media queries across components.
- Arion: colors come only from `theme.Tokens` (`ui/shared/theme`), buttons from `kit.Button`/`kit.IconButton`, and no view calls `ui.RGB`, `ui.Hex` or `ui.Oklch`. See [design system](design-system.md).

## Components are small and focused

- A component should do one thing.
- Extract a new component when a file grows beyond ~300 lines or when a block is reused.
- Components receive data and callbacks through props; they do not fetch their own data unless they live in a feature slice and that is the slice's explicit responsibility.

## Layout rules

- Use Flexbox to partition the screen into stable regions (header, content, footer).
- Only designated containers scroll. Do not allow the body or arbitrary containers to scroll.
- Hidden-title-bar windows must keep interactive controls outside the native caption bounds reported by `Context.TitleBar()`. Onboarding places its progress/language header below right-side caption buttons; left-side macOS controls remain over the illustration. Use DIP metrics from the window, not fixed pixel offsets.
- Set `min-height: 0` on every flex container that participates in the scroll chain.
- The page root should fill the viewport (`min-h-dvh` / `h-dvh`).

## Agent skills

When building or reviewing UI, apply the project Agent Skills in [`.claude/skills/`](../../.claude/skills/). See [`docs/quality/agent-skills.md`](../quality/agent-skills.md) for the catalog and the `.agents` alias.

- `make-interfaces-feel-better` and `emil-design-eng` for polish, motion, and detail work.
- `kill-ai-slop` when the UI or copy looks generic, templated, or machine-default.
- `animate` / `review-animations` / `improve-animations` when adding or auditing motion.

Do not introduce a second styling system just to apply a polish fix. Express the change in the project's existing tokens and FSD layers.

## Accessibility

The shared Pinyin worktree uses the same candidate appearance, ranking and key controls on both platforms, while preserving Windows-specific setup and error guidance. Windows replacement offers removal and reinstall rather than macOS's live-update action. These controls must ship together with the verified full native keyboard runtime; the older installed voice-only package does not support them. Native Windows Pinyin acceptance remains pending.

- Use semantic HTML (`button`, `a`, `label`, `nav`, `main`).
- Every interactive element must have an accessible name.
- Do not build fake buttons or links with `div` + click handlers.
