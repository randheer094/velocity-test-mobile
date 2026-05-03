# flows

Common navigation and auth procedures shared across `tests/`. Test files
reference these by name (`Run fixtures/flows.md → goToHome`) instead of
inlining the steps. Each procedure is **idempotent** — running it twice in
a row produces the same end state — so callers do not have to reason about
prior context.

When the same five-step procedure shows up in three or more test files,
extract it here. Procedures that appear in only one test stay inline.

## login

Authenticate as the canonical test user. Run after `fixtures/preparation.md
→ Standard pre-conditions`, which ends on the login screen.

1. "Sign in" is shown within 5s.
2. Type `alice` into the Username field.
3. Type `s3cret` into the Password field.
4. Tap "Sign in".
5. "Welcome, alice" appears within 10s.

> `tests/01-login.md` Test 1 is the canonical assertion that this
> procedure works. Other test files use it as a pre-condition; if it
> regresses there, fix the procedure once here, not in every caller.

## goToHome

Dismisses any pending modal and lands on the notes list. No-op when
already on the home screen.

Up to 3 attempts:

1. Wait for the screen to settle.
2. If a dialog is visible (any node with `className` containing
   `Dialog`), press BACK once.
3. If "All notes" appears within 1s, return.
4. Otherwise press BACK once and retry from step 1.
5. After 3 unsuccessful attempts, fail with the assertion *"All notes" is
   shown* so the report shows the actual top-of-stack screen via the
   captured `print_tree`.

The retry loop exists because the detail and edit screens both push a
back-stack entry; from a deep stack one BACK is not always enough.

## logout

End the authenticated session. Use sparingly — most tests benefit from
session reuse via `Standard pre-conditions` skipping cleanup, and the
next file's pre-conditions clear app data anyway.

1. Tap the avatar.

   > *Match by `contentDescription = "profile-menu"`. The avatar's
   > rendered initial letter ("A" for `alice`) collides with any UI label
   > containing the same character under `text` matching.*

2. Tap "Sign out".
3. "Sign in" appears within 5s.

## Composition pattern

Tests that need an authenticated, on-home starting state stack three
fixtures in their **Pre-conditions** block:

```md
## File-level pre-conditions
1. Run `fixtures/preparation.md` → **Standard pre-conditions**.
2. Run `fixtures/flows.md` → `login`.
3. Run `fixtures/flows.md` → `goToHome`.
```

For tests that also need seeded notes, append a fourth step:

```md
4. Run `fixtures/preparation.md` → **resetNotes**.
```

The agent walks them in order; if any fixture step fails, the test is
marked FAIL before its first real assertion runs.
