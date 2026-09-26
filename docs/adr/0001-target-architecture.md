# ADR 0001: Target architecture

- Status: Accepted
- Date: 2026-09-26

## Context

The repository currently combines independently installable initializers with
shared configuration and application lifecycle facilities. This decision
records the dependency boundaries that future changes should move toward;
recording it does not change production behavior.

## Decision

- Every initializer is independently installable.
- An initializer may depend on its native driver, but not on apin, a central
  manifest, or another initializer.
- Every initializer owns its configuration, defaults, and validation.
- The native client remains accessible through its shell.
- apin is optional lifecycle orchestration.
- The configuration loader understands formats and environment overlays, but
  no initializer vocabulary.
- Applications define their own aggregate configuration from only the
  initializers they select.
- App is the single lifecycle-policy owner.
- Libraries do not call `os.Exit`.

## Consequences

Applications choose which initializers and configuration types they need.
Initializer configuration and validation evolve with each initializer, while
the loader remains format-focused. Lifecycle policy belongs to App when an
application opts into apin orchestration. Initializers continue to expose
their native clients through their shells.

These are target dependency rules, not claims that the current implementation
already satisfies every rule. Existing behavior remains unchanged by this ADR.
