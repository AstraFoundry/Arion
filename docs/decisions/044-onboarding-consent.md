# ADR 044: Onboarding consent beside sign-in

Status: Accepted

## Context

Onboarding already includes account sign-in on Welcome. A separate data-sharing step or post-sign-in switch adds a decision away from the sign-in documents.

## Decision

Show terms acceptance and optional usage sharing immediately below browser sign-in. Terms starts checked; usage sharing starts unchecked. Unchecking terms disables browser sign-in but preserves local continuation. Each choice has its own accessible checkbox, and usage sharing links to the site's localized data disclosure.

Reuse the existing consent-bound sign-in contract: only an explicitly checked usage choice records consent after successful authentication. Unchecked sign-in reads the existing account answer. No reporting starts before platform confirmation. Cancellation and failure discard the pending choice. Settings retains the account sharing switch for later changes. There is no independent sharing page or post-sign-in onboarding switch.

[PRV-10](../../specs/005-privacy.md) owns reporting boundaries. The disclosure lists actual aggregate fields and account linkage; it must not imply anonymity or authorize audio/text uploads. [Account sign-in](../project/account-signin.md) owns the call contract.

## Validation

`TestOnboardingConsent` covers both interface languages and both issuer layouts, default-off sharing, terms gating, localized disclosure links and absence of a post-sign-in sharing control. Existing account tests cover consent confirmation, cancellation, failure and reporting authorization.
